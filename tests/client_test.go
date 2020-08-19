// +build integration

package tests

import (
	"context"
	openapi "github.com/asecurityteam/asset-inventory-api/client"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"testing"
)

var assetInventoryAPI *openapi.APIClient

func TestMain(m *testing.M) {
	config := openapi.NewConfiguration()
	appURL := os.Getenv("AIA_APP_URL")
	config.BasePath = appURL
	assetInventoryAPI = openapi.NewAPIClient(config)
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
