// +build integration

package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
)

var assetInventoryAPI *openapi.APIClient //this contains pre-configured API client used by all fixtures

func TestMain(m *testing.M) {
	config := openapi.NewConfiguration()
	appURL := os.Getenv("AIA_APP_URL")
	config.BasePath = appURL
	_, config.Debug = os.LookupEnv("AIA_INTEGRATION_DEBUG")
	config.Debug = true
	assetInventoryAPI = openapi.NewAPIClient(config)
	ctx := context.Background()
	schemaVersion = getSchemaVersion(ctx, assetInventoryAPI.DefaultApi)
	res := 0

	//run all know tests with all supported schema versions
	for v := minSchemaVersion; v <= maxSchemaVersion; v++ {
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

func Setup(t *testing.T, ctx context.Context) (openapi.CloudAssetChanges, openapi.CloudAssetChanges, *openapi.DefaultApiService) {
	// ensure the Asset has something assigned so that we can find it
	chgAssign := SampleAssetChanges()
	// create the matching delete changes so that we can check AFTER the assignment time
	chgRemove := SampleAssetChanges()
	chgRemove.Changes[0].ChangeType = "DELETED"
	chgRemove.ChangeTime = chgRemove.ChangeTime.Add(24 * time.Hour)
	// shortcut to defaultApi
	api := assetInventoryAPI.DefaultApi
	// add change events
	for _, chg := range []openapi.CloudAssetChanges{chgAssign, chgRemove} {
		_, err := api.V1CloudChangePost(ctx, chg)
		if err != nil {
			t.Errorf("error publishing sample change: %#v", err)
		}
	}
	return chgAssign, chgRemove, api
}

func RawUrlFollowupTests(t *testing.T, suffixPath string, tsDuring time.Time) {
	// subsequent tests can not be driven by the same table as we must call http directly
	// because swagger bindings do not allow to mess with invalid/missing time values or arbitrary arguments
	endpoint, err := url.Parse(assetInventoryAPI.GetConfig().BasePath)
	if err != nil {
		t.Fatalf("unable to parse base path for direct calls #%v", err)
	}
	// construct a complete valid URL for raw tests to start off with
	endpoint.Path = path.Join(endpoint.Path, suffixPath)
	q := endpoint.Query()
	q.Add("time", tsDuring.Format(time.RFC3339Nano))
	endpoint.RawQuery = q.Encode()

	rawHttp := http.Client{
		Timeout: 500 * time.Millisecond,
	}

	rawTestCases := map[string]struct {
		urlProcessor func(url.URL) url.URL //pass by value!
		httpCode     int
		mustError    bool // unlike openapi, http client does not set err for 404 or 400
	}{
		"RawRequestValid": { // this one is to test the raw test, to make sure we can get 200 over raw connection
			func(u url.URL) url.URL {
				return u // do nothing, send valid request
			},
			http.StatusOK,
			false,
		},
		"TimestampMissing": {
			func(u url.URL) url.URL {
				q := u.Query()
				q.Del("time")
				u.RawQuery = q.Encode()
				return u
			},
			http.StatusBadRequest,
			false,
		},
		"TimestampMalformed": {
			func(u url.URL) url.URL {
				q := u.Query()
				q.Del("time")
				q.Add("time", "somewhere in time and not valid at all")
				u.RawQuery = q.Encode()
				return u
			},
			http.StatusBadRequest,
			false,
		},
		"TimestampDuplicatedMalformed": {
			func(u url.URL) url.URL { //this results in two different timestamp= values, one being malformed
				q := u.Query()
				q.Add("time", "somewhere in time and not valid at all")
				u.RawQuery = q.Encode()
				return u
			},
			http.StatusOK, //FIXME this should be StatusBadRequest, as we should not shop around for valid values
			false,
		},
		"UnknownQueryArgument": {
			func(u url.URL) url.URL {
				q := u.Query()
				q.Add("rogue", "totally useless and not part of api definition")
				u.RawQuery = q.Encode()
				return u
			},
			http.StatusOK, //FIXME this should be StatusBadRequest, the unexpected args == typo/error
			false,
		},
	}

	for name, tc := range rawTestCases {
		t.Run(addSchemaVersion(name), func(t *testing.T) {
			rawUrl := tc.urlProcessor(*endpoint)
			req, err := http.NewRequest("GET", rawUrl.String(), nil)
			if err != nil {
				t.Fatalf("unable to initialize request for direct call #%v", err)
			}
			res, err := rawHttp.Do(req)
			if tc.mustError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.httpCode, res.StatusCode)
		})
	}
}
