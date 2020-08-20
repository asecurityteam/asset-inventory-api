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

func TestMain(m *testing.M) {
	config := openapi.NewConfiguration()
	appURL := os.Getenv("AIA_APP_URL")
	config.BasePath = appURL
	assetInventoryAPI = openapi.NewAPIClient(config)
	ctx := context.Background()

	// Test on latest db schema
	for {
		_, _, err := assetInventoryAPI.DefaultApi.OpsPgsqlV1SchemaVersionStepUpGet(ctx)

		// An error hopefully means that he hit the last schema version
		if err != nil {
			break
		}
	}

	code := m.Run()
	os.Exit(code)
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
		t.Run(test.Name, fn)
	}
}
