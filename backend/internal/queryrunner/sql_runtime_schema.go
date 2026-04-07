package queryrunner

import (
	"context"
	"fmt"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func fetchSQLSchema(ctx context.Context, target *contracts.DatabaseTarget) (contracts.SchemaInfo, error) {
	switch targetProtocol(target) {
	case protocolMySQL:
		return fetchMySQLSchema(ctx, target)
	case protocolMSSQL:
		return fetchMSSQLSchema(ctx, target)
	case protocolOracle:
		return fetchOracleSchema(ctx, target)
	default:
		return contracts.SchemaInfo{}, fmt.Errorf("unsupported database protocol %q", target.Protocol)
	}
}

func emptySchemaInfo() contracts.SchemaInfo {
	return contracts.SchemaInfo{
		Tables:     make([]contracts.SchemaTable, 0),
		Views:      make([]contracts.SchemaView, 0),
		Functions:  make([]contracts.SchemaRoutine, 0),
		Procedures: make([]contracts.SchemaRoutine, 0),
		Triggers:   make([]contracts.SchemaTrigger, 0),
		Sequences:  make([]contracts.SchemaSequence, 0),
		Packages:   make([]contracts.SchemaPackage, 0),
		Types:      make([]contracts.SchemaNamedType, 0),
	}
}
