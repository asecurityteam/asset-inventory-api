// +build integration

package tests

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
)

//TODO add tests for resource IDs containing slashes, tracked in separate ticket

func TestLookupByResourceID(t *testing.T) {
	ctx := context.Background()
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
	// extract resource ID
	spl := strings.Split(chgAssign.Arn, "/") // this will need separate handling (helper func? for things like ELB)
	resId := spl[len(spl)-1]

	tsDuring := chgAssign.ChangeTime.Add(1 * time.Second)
	tsBefore := chgAssign.ChangeTime.Add(-1 * time.Second) // nb .Sub does something very different :confused:
	tsAfter := chgRemove.ChangeTime.Add(1 * time.Second)

	testCases := map[string]struct {
		resourceID string
		ts         time.Time
		httpCode   int
		mustError  bool
	}{
		"Valid": { // ideally we'd have response validation for happy path here, but the test for cloudchanges does it
			resId,
			tsDuring,
			http.StatusOK,
			false,
		},
		"TSBefore": {
			resId,
			tsBefore,
			http.StatusNotFound,
			true, // openapi bindings treat 404 as error
		},
		"TSAfter": {
			resId,
			tsAfter,
			http.StatusNotFound,
			true, // openapi bindings treat 404 as error
		},
		"ResIDEmpty": {
			"",
			tsDuring,
			http.StatusBadRequest,
			true,
		},
		//TODO find a way to inject invalid timestamp
	}
	for name, tc := range testCases {
		t.Run(addSchemaVersion(name),
			func(t *testing.T) {
				_, httpRes, err := api.V1CloudResourceidResourceidGet(ctx, tc.resourceID, tc.ts)
				if tc.mustError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
				assert.Equal(t, tc.httpCode, httpRes.StatusCode)
			})
	}

	// subsequent tests can not be driven by the same table as we must call http directly
	// because swagger bindings do not allow to mess with invalid/missing time values or arbitrary arguments
	endpoint, err := url.Parse(assetInventoryAPI.GetConfig().BasePath)
	if err != nil {
		t.Fatalf("unable to parse base path for direct calls #%v", err)
	}
	// construct a complete valid URL for raw tests to start off with
	endpoint.Path = path.Join(endpoint.Path, "/v1/cloud/resourceid/", resId)
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
