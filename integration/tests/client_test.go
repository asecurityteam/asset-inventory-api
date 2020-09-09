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
var schemaVersion int32
var maxSchema int32 = 14 // TODO: extrapolate this somewhere?

func WithSchemaVersion(input string) string{
	return fmt.Sprintf("Schema V%d %s", schemaVersion, input)
}

func getSchemaVersion(ctx context.Context, api *openapi.DefaultApiService) int32{
	versionState, _, err := api.OpsPgsqlV1SchemaVersionGet(ctx)
	if err!=nil{
		panic(err)
	}
	if versionState.Dirty{
		panic("dirty schema version")
	}
	return versionState.Version
}

func TestMain(m *testing.M) {
	config := openapi.NewConfiguration()
	appURL := os.Getenv("AIA_APP_URL")
	config.BasePath = appURL
	// config.Debug = true
	assetInventoryAPI = openapi.NewAPIClient(config)
	ctx := context.Background()
	schemaVersion = getSchemaVersion(ctx, assetInventoryAPI.DefaultApi)
	res := 0
	john := openapi.Person{
		Name:  "John Smith",
		Login: "jsmith",
		Email: "jsmith@atlassian.com",
		Valid: false,
	}
	accountOwner := openapi.AccountOwner{
		AccountId: "001234567891011",
		Owner:    john,
		Champions: []openapi.Person{john},
	}
	_, err := assetInventoryAPI.DefaultApi.V1AccountOwnerPost(ctx, accountOwner)
	if err!=nil{
		panic(err)
	}
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
