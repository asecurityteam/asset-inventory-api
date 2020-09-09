// +build integration

package tests

import (
	"context"
	"fmt"
	"os"
	"testing"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
)

var assetInventoryAPI *openapi.APIClient //this contains pre-configured API client used by all fixtures

func TestMain(m *testing.M) {
	config := openapi.NewConfiguration()
	appURL := os.Getenv("AIA_APP_URL")
	config.BasePath = appURL
	_, config.Debug = os.LookupEnv("AIA_INTEGRATION_DEBUG")
	assetInventoryAPI = openapi.NewAPIClient(config)
	ctx := context.Background()
	schemaVersion = getSchemaVersion(ctx, assetInventoryAPI.DefaultApi)
	res := 0

	//run all know tests with all supported schema versions
	for v := int32(12); v <= maxSchema; v++ {
		err := setSchemaVersion(v)
		if err != nil {
			panic(fmt.Errorf("error migrating database schema %#v", err))
		}
		// until we fix return format for cases when account has no owners/champions set - this is required
		_, err = assetInventoryAPI.DefaultApi.V1AccountOwnerPost(ctx, SampleAccountOwner())
		if err != nil {
			panic(err)
		}
		// run all discovered tests
		res += m.Run()
	}
	os.Exit(res) //non-zero if any of the fixtures failed
}
