package queryrunner

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func fetchMongoSchema(ctx context.Context, target *contracts.DatabaseTarget) (contracts.SchemaInfo, error) {
	targetConn, err := openMongoTarget(ctx, target)
	if err != nil {
		return contracts.SchemaInfo{}, err
	}
	defer targetConn.Close()

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	names, err := targetConn.database.ListCollectionNames(queryCtx, bson.M{})
	if err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("list mongodb collections: %w", err)
	}
	sort.Strings(names)

	result := emptySchemaInfo()
	for _, name := range names {
		table, err := inferMongoCollectionSchema(queryCtx, targetConn.database.Collection(name), targetConn.database.Name())
		if err != nil {
			return contracts.SchemaInfo{}, err
		}
		result.Tables = append(result.Tables, table)
	}
	return result, nil
}

func explainMongoQuery(_ context.Context, _ *contracts.DatabaseTarget, _ contracts.QueryPlanRequest) (contracts.QueryPlanResponse, error) {
	return contracts.QueryPlanResponse{Supported: false}, nil
}

func introspectMongoQuery(ctx context.Context, target *contracts.DatabaseTarget, req contracts.QueryIntrospectionRequest) (contracts.QueryIntrospectionResponse, error) {
	targetConn, err := openMongoTarget(ctx, target)
	if err != nil {
		return contracts.QueryIntrospectionResponse{}, err
	}
	defer targetConn.Close()

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	ref := parseObjectRef(req.Target, "")
	switch req.Type {
	case "indexes":
		collection, err := requireMongoCollection(targetConn.database, ref.Name)
		if err != nil {
			return contracts.QueryIntrospectionResponse{}, err
		}
		cursor, err := collection.Indexes().List(queryCtx)
		if err != nil {
			return contracts.QueryIntrospectionResponse{}, fmt.Errorf("list mongodb indexes: %w", err)
		}
		defer cursor.Close(queryCtx)
		var docs []bson.M
		if err := cursor.All(queryCtx, &docs); err != nil {
			return contracts.QueryIntrospectionResponse{}, fmt.Errorf("decode mongodb indexes: %w", err)
		}
		return contracts.QueryIntrospectionResponse{Supported: true, Data: normalizeMongoDocuments(docs)}, nil
	case "statistics":
		collectionName := ref.Name
		if collectionName == "" {
			return contracts.QueryIntrospectionResponse{}, fmt.Errorf("target is required for introspection type %q", req.Type)
		}
		var doc bson.M
		if err := targetConn.database.RunCommand(queryCtx, bson.M{"collStats": collectionName}).Decode(&doc); err != nil {
			return contracts.QueryIntrospectionResponse{}, fmt.Errorf("run mongodb collStats: %w", err)
		}
		return contracts.QueryIntrospectionResponse{Supported: true, Data: normalizeMongoDocument(doc)}, nil
	case "foreign_keys":
		return contracts.QueryIntrospectionResponse{Supported: false}, nil
	case "table_schema":
		collection, err := requireMongoCollection(targetConn.database, ref.Name)
		if err != nil {
			return contracts.QueryIntrospectionResponse{}, err
		}
		table, err := inferMongoCollectionSchema(queryCtx, collection, targetConn.database.Name())
		if err != nil {
			return contracts.QueryIntrospectionResponse{}, err
		}
		fields := make([]map[string]any, 0, len(table.Columns))
		for _, column := range table.Columns {
			fields = append(fields, map[string]any{
				"name":       column.Name,
				"data_type":  column.DataType,
				"nullable":   column.Nullable,
				"is_primary": column.IsPrimaryKey,
			})
		}
		return contracts.QueryIntrospectionResponse{Supported: true, Data: fields}, nil
	case "row_count":
		collection, err := requireMongoCollection(targetConn.database, ref.Name)
		if err != nil {
			return contracts.QueryIntrospectionResponse{}, err
		}
		count, err := collection.EstimatedDocumentCount(queryCtx)
		if err != nil {
			return contracts.QueryIntrospectionResponse{}, fmt.Errorf("count mongodb documents: %w", err)
		}
		return contracts.QueryIntrospectionResponse{Supported: true, Data: map[string]any{"approximate_count": count}}, nil
	case "database_version":
		var doc bson.M
		if err := targetConn.database.RunCommand(queryCtx, bson.M{"buildInfo": 1}).Decode(&doc); err != nil {
			return contracts.QueryIntrospectionResponse{}, fmt.Errorf("run mongodb buildInfo: %w", err)
		}
		return contracts.QueryIntrospectionResponse{Supported: true, Data: normalizeMongoDocument(doc)}, nil
	default:
		return contracts.QueryIntrospectionResponse{Supported: false}, nil
	}
}

func requireMongoCollection(database *mongo.Database, name string) (*mongo.Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("mongodb queries require collection")
	}
	return database.Collection(name), nil
}

func inferMongoCollectionSchema(ctx context.Context, collection *mongo.Collection, schemaName string) (contracts.SchemaTable, error) {
	table := contracts.SchemaTable{
		Name:    collection.Name(),
		Schema:  schemaName,
		Columns: []contracts.SchemaColumn{},
	}

	var sample bson.M
	err := collection.FindOne(ctx, bson.M{}).Decode(&sample)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return table, nil
		}
		return contracts.SchemaTable{}, fmt.Errorf("sample mongodb collection %s: %w", collection.Name(), err)
	}

	keys := make([]string, 0, len(sample))
	for key := range sample {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		table.Columns = append(table.Columns, contracts.SchemaColumn{
			Name:         key,
			DataType:     mongoTypeName(sample[key]),
			Nullable:     true,
			IsPrimaryKey: key == "_id",
		})
	}

	return table, nil
}
