package queryrunner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func executeMongoReadOnly(ctx context.Context, target *contracts.DatabaseTarget, req contracts.QueryExecutionRequest) (contracts.QueryExecutionResponse, error) {
	spec, err := parseMongoQuerySpec(req.SQL)
	if err != nil {
		return contracts.QueryExecutionResponse{}, err
	}
	if err := validateMongoReadOnlySpec(spec); err != nil {
		return contracts.QueryExecutionResponse{}, err
	}
	return executeMongoAny(ctx, target, req)
}

func executeMongoAny(ctx context.Context, target *contracts.DatabaseTarget, req contracts.QueryExecutionRequest) (contracts.QueryExecutionResponse, error) {
	spec, err := parseMongoQuerySpec(req.SQL)
	if err != nil {
		return contracts.QueryExecutionResponse{}, err
	}

	maxRows := req.MaxRows
	switch {
	case maxRows <= 0:
		maxRows = defaultMaxRows
	case maxRows > maxAllowedRows:
		maxRows = maxAllowedRows
	}
	if spec.Limit <= 0 || spec.Limit > int64(maxRows) {
		spec.Limit = int64(maxRows)
	}

	targetConn, err := openMongoTarget(ctx, target)
	if err != nil {
		return contracts.QueryExecutionResponse{}, err
	}
	defer targetConn.Close()

	if database := strings.TrimSpace(spec.Database); database != "" && database != targetConn.database.Name() {
		targetConn.database = targetConn.client.Database(database)
	}

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	start := time.Now()
	result, err := executeMongoSpec(queryCtx, targetConn.database, spec)
	if err != nil {
		return contracts.QueryExecutionResponse{}, err
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func executeMongoSpec(ctx context.Context, database *mongo.Database, spec mongoQuerySpec) (contracts.QueryExecutionResponse, error) {
	if database == nil {
		return contracts.QueryExecutionResponse{}, fmt.Errorf("mongodb database is unavailable")
	}

	switch spec.Operation {
	case "find":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		findOpts := options.Find()
		findOpts.SetLimit(spec.Limit)
		if len(spec.Projection) > 0 {
			findOpts.SetProjection(spec.Projection)
		}
		if len(spec.Sort) > 0 {
			findOpts.SetSort(spec.Sort)
		}
		if spec.Skip > 0 {
			findOpts.SetSkip(spec.Skip)
		}
		cursor, err := collection.Find(ctx, defaultMongoMap(spec.Filter), findOpts)
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb find: %w", err)
		}
		defer cursor.Close(ctx)
		var docs []bson.M
		if err := cursor.All(ctx, &docs); err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("decode mongodb documents: %w", err)
		}
		return mongoDocumentsToResult(docs), nil
	case "aggregate":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		cursor, err := collection.Aggregate(ctx, spec.Pipeline)
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb aggregate: %w", err)
		}
		defer cursor.Close(ctx)
		var docs []bson.M
		if err := cursor.All(ctx, &docs); err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("decode mongodb aggregate: %w", err)
		}
		return mongoDocumentsToResult(docs), nil
	case "count":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		count, err := collection.CountDocuments(ctx, defaultMongoMap(spec.Filter))
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb count: %w", err)
		}
		return singleRowMongoResult(map[string]any{"count": count}), nil
	case "distinct":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		if strings.TrimSpace(spec.Field) == "" {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("mongodb distinct requires field")
		}
		distinctResult := collection.Distinct(ctx, spec.Field, defaultMongoMap(spec.Filter))
		if err := distinctResult.Err(); err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb distinct: %w", err)
		}
		var values []any
		if err := distinctResult.Decode(&values); err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("decode mongodb distinct: %w", err)
		}
		rows := make([]map[string]any, 0, len(values))
		for _, value := range values {
			rows = append(rows, map[string]any{"value": normalizeMongoValue(value)})
		}
		return contracts.QueryExecutionResponse{Columns: []string{"value"}, Rows: rows, RowCount: len(rows)}, nil
	case "insertone":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		if len(spec.Document) == 0 {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("mongodb insertOne requires document")
		}
		res, err := collection.InsertOne(ctx, spec.Document)
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb insertOne: %w", err)
		}
		return singleRowMongoResult(map[string]any{"insertedId": normalizeMongoValue(res.InsertedID)}), nil
	case "insertmany":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		if len(spec.Documents) == 0 {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("mongodb insertMany requires documents")
		}
		docs := make([]any, 0, len(spec.Documents))
		for _, doc := range spec.Documents {
			docs = append(docs, doc)
		}
		res, err := collection.InsertMany(ctx, docs)
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb insertMany: %w", err)
		}
		return singleRowMongoResult(map[string]any{"insertedCount": len(res.InsertedIDs)}), nil
	case "updateone":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		res, err := collection.UpdateOne(ctx, defaultMongoMap(spec.Filter), defaultMongoMap(spec.Update))
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb updateOne: %w", err)
		}
		return singleRowMongoResult(map[string]any{
			"matchedCount":  res.MatchedCount,
			"modifiedCount": res.ModifiedCount,
			"upsertedCount": boolToInt(res.UpsertedID != nil),
			"upsertedId":    normalizeMongoValue(res.UpsertedID),
		}), nil
	case "updatemany":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		res, err := collection.UpdateMany(ctx, defaultMongoMap(spec.Filter), defaultMongoMap(spec.Update))
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb updateMany: %w", err)
		}
		return singleRowMongoResult(map[string]any{
			"matchedCount":  res.MatchedCount,
			"modifiedCount": res.ModifiedCount,
			"upsertedCount": boolToInt(res.UpsertedID != nil),
			"upsertedId":    normalizeMongoValue(res.UpsertedID),
		}), nil
	case "deleteone":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		res, err := collection.DeleteOne(ctx, defaultMongoMap(spec.Filter))
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb deleteOne: %w", err)
		}
		return singleRowMongoResult(map[string]any{"deletedCount": res.DeletedCount}), nil
	case "deletemany":
		collection, err := requireMongoCollection(database, spec.Collection)
		if err != nil {
			return contracts.QueryExecutionResponse{}, err
		}
		res, err := collection.DeleteMany(ctx, defaultMongoMap(spec.Filter))
		if err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb deleteMany: %w", err)
		}
		return singleRowMongoResult(map[string]any{"deletedCount": res.DeletedCount}), nil
	case "runcmd", "runcommand":
		if len(spec.Command) == 0 {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("mongodb runCommand requires command")
		}
		var doc bson.M
		if err := database.RunCommand(ctx, spec.Command).Decode(&doc); err != nil {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("execute mongodb runCommand: %w", err)
		}
		return singleRowMongoResult(normalizeMongoDocument(doc)), nil
	default:
		return contracts.QueryExecutionResponse{}, fmt.Errorf("unsupported mongodb operation %q", spec.Operation)
	}
}
