// +build integration

package tests

import (
	"context"
	"fmt"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
)

var schemaVersion int32  //current schema version
var maxSchema int32 = 14 // TODO: extrapolate this somewhere?

// decorate a test name with current schema version
func addSchemaVersion(input string) string {
	return fmt.Sprintf("Schema v%d %s", schemaVersion, input)
}

// get current schema version via API
func getSchemaVersion(ctx context.Context, api *openapi.DefaultApiService) int32 {
	versionState, _, err := api.OpsPgsqlV1SchemaVersionGet(ctx)
	if err != nil {
		panic(err.Error())
	}
	if versionState.Dirty {
		panic("dirty schema version")
	}
	return versionState.Version
}

// migrate database to the desired schema
func setSchemaVersion(version int32) error {
	ctx := context.Background()

	if schemaVersion == version {
		return nil
	} else if schemaVersion < version {
		schema, _, err := assetInventoryAPI.DefaultApi.OpsPgsqlV1SchemaVersionStepUpGet(ctx)

		if err != nil {
			return err
		}

		schemaVersion = schema.Version
	} else {
		schema, _, err := assetInventoryAPI.DefaultApi.OpsPgsqlV1SchemaVersionStepDownGet(ctx)

		if err != nil {
			return err
		}

		schemaVersion = schema.Version
	}

	return setSchemaVersion(version)
}
