// +build integration

package tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
	"github.com/stretchr/testify/assert"
)

var assetInventoryAPI *openapi.APIClient
var schemaVersion int32 = 6 // our default data starts at v6 as of now
var maxSchema int32 = 14 // TODO: extrapolate this somewhere?

func WithSchemaVersion(input string) string{
	return fmt.Sprintf("Schema V%d %s", schemaVersion, input)
}

func TestMain(m *testing.M) {
	config := openapi.NewConfiguration()
	appURL := os.Getenv("AIA_APP_URL")
	config.BasePath = appURL
	assetInventoryAPI = openapi.NewAPIClient(config)
	res := 0
	for v:= int32(12); v<= maxSchema; v++ {
		err := setSchemaVersion(v)
		if err != nil {
			panic(fmt.Errorf("error migrating database schema %#v", err))
		}
		res += m.Run()
	}
	os.Exit(res) //non-zero if any of the fixtures failed
}

func TestHealthcheck(t *testing.T) {
	tt := []struct {
		Name string
	}{
		{
			Name: "Health Check",
		},
	}
	for _, test := range tt {
		fn := func(t *testing.T) {
			ctx := context.Background()
			resp, err := assetInventoryAPI.DefaultApi.HealthcheckGet(ctx)
			assert.NoError(t, err, "Health check should produce no errors")
			assert.Equal(t, resp.StatusCode, http.StatusOK)
		}
		t.Run(WithSchemaVersion(test.Name), fn)
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
