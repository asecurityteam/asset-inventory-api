// +build integration

package tests

import (
	"context"
	"net/http"
	"os"
	"testing"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
	"github.com/stretchr/testify/assert"
)

var assetInventoryAPI *openapi.APIClient
var schemaVersion int32 = 1
var maxSchema int32 = 13 // TODO: extrapolate this somewhere?

func TestMain(m *testing.M) {
	config := openapi.NewConfiguration()
	appURL := os.Getenv("AIA_APP_URL")
	config.BasePath = appURL
	assetInventoryAPI = openapi.NewAPIClient(config)
	code := m.Run()
	os.Exit(code)
}

func TestHealthcheck(t *testing.T) {
	// Example: check every schema version against health check.
	for schema := schemaVersion; schema <= maxSchema; schema++ {
		tt := []struct {
			Name string
		}{
			{
				Name: "Health Check -- Testing with test change",
			},
		}
		for _, test := range tt {
			fn := func(t *testing.T) {
				ctx := context.Background()
				resp, err := assetInventoryAPI.DefaultApi.HealthcheckGet(ctx)
				assert.NoError(t, err, "Health check should produce no errors")
				assert.Equal(t, resp.StatusCode, http.StatusOK)
			}
			t.Run(test.Name, fn)
		}

		if schema < maxSchema {
			err := setSchemaVersion(schema + 1)
			assert.NoError(t, err, "The database migration should not return an error")
		}
	}
}

// Migrate database to the desired schema
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
