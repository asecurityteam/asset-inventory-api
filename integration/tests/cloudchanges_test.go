// +build integration

package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
)

func LookupError(t *testing.T, needle string, when time.Time, err error) {
	t.Errorf("error looking up %s at %s : %s", needle, when, err.Error())
}

func LookupWrongCode(t *testing.T, needle string, when time.Time, response http.Response) {
	t.Errorf("unexpected response looking up %s at %s : %d", needle, when, response.StatusCode)
}

func LookupNoAssetInResponse(t *testing.T, needle string, when time.Time) {
	t.Errorf("response does not contain resource looking up %s at %s", needle, when)
}

// check (in a sub-test per item) if all changes in "changes" were properly applied to A-I server by running lookups
func CheckChangesPresent(t *testing.T, changes openapi.CloudAssetChanges) {
	ctx := context.Background()
	ts := changes.ChangeTime.Add(time.Second) //TODO we might neeed different timestamp handling once we need REMOVED
	spl := strings.Split(changes.Arn, "/")
	resId := spl[len(spl)-1]
	type check struct {
		lookup   func(context.Context, string, time.Time) (openapi.CloudAssets, *http.Response, error)
		haystack string
	}
	tests := []check{ // add the check for lookup-by-resource-id
		{assetInventoryAPI.DefaultApi.V1CloudResourceidResourceidGet, resId},
	}
	// this looks very verbose compared to list comprehensions or map(), but Rob Pike tells this is fine
	// https://github.com/robpike/filter , so :shrug:
	for _, change := range changes.Changes { // for every change
		if change.ChangeType != "ADDED" { //no tests for REMOVED yet
			continue
		}
		for _, publicIP := range change.PublicIpAddresses { // add check for every public IP
			tests = append(tests, check{assetInventoryAPI.DefaultApi.V1CloudIpIpAddressGet, publicIP})
		}
		for _, privateIP := range change.PrivateIpAddresses { // add check for every private IP
			tests = append(tests, check{assetInventoryAPI.DefaultApi.V1CloudIpIpAddressGet, privateIP})
		}
		for _, hostName := range change.Hostnames { // hostnames
			tests = append(tests, check{assetInventoryAPI.DefaultApi.V1CloudHostnameHostnameGet, hostName})
		}
	}
	for _, test := range tests { // run every created check as separate sub-test
		t.Run("Test lookup by:"+test.haystack, func(t *testing.T) {
			assets, httpRes, err := test.lookup(ctx, test.haystack, ts)
			if err != nil {
				LookupError(t, test.haystack, ts, err)
			} else if httpRes.StatusCode != 200 {
				LookupWrongCode(t, test.haystack, ts, *httpRes)
			} else if !ChangesInResponse(changes, assets.Assets) { // check if the resource from changes is present in response
				LookupNoAssetInResponse(t, test.haystack, ts)
			}
		})
	}
}

func TestCloudChanges(t *testing.T) {
	testCases := map[string]struct {
		changesAdapter    func(*openapi.CloudAssetChanges)
		expectedResponse  int
		mustError         bool
		responseValidator func(*testing.T, openapi.CloudAssetChanges)
	}{
		"Valid": {
			func(changes *openapi.CloudAssetChanges) {
			},
			http.StatusCreated,
			false,
			CheckChangesPresent,
		},
		/* "MissingAccountId": { Disabled. Turns PSQL into a pumpkin because 2 resources exist for same ARNID
			func(changes *openapi.CloudAssetChanges){
				changes.AccountId = ""
			},
			http.StatusCreated,
			false,
			nil,
		},*/
		"BadResourceType": {
			func(changes *openapi.CloudAssetChanges) {
				changes.ResourceType = "MS:Windows:2000"
			},
			http.StatusCreated, //TODO fix the code this should be 400. Currently A-I-API accepts this.
			false,
			nil,
		},
		/* "BadPrivateIP": { Disabled. Permanently poisons persistent PSQL connection. Need validation of IPs.
			func(changes *openapi.CloudAssetChanges) {
				changes.Changes[0].PrivateIpAddresses[0]="I am not an ip address"
			},
			http.StatusBadRequest,
			false,
			nil,
		},
		"PublicIPsWithoutHostnames": { Disabled. Is not handled as it should. Need 400, getting 201
			func(changes *openapi.CloudAssetChanges) {
				changes.Changes[0].Hostnames = []string{}
			},
			//TODO fix/check the code. We should not accept semantically incorrect event. This should be 400.
			http.StatusBadRequest,
			false,
			nil,
		},*/
		"InvalidChangeType": {
			func(changes *openapi.CloudAssetChanges) {
				changes.Changes[0].ChangeType = "INVALIDTYPE"
			},
			http.StatusBadRequest,
			true,
			nil,
		},
	}
	for name, tc := range testCases {
		t.Run(addSchemaVersion(name),
			func(t *testing.T) {
				ctx := context.Background()
				changes := SampleAssetChanges() // get sample valid AssetChanges
				tc.changesAdapter(&changes)     // call the function to modify AssetChanges to match test goal
				resp, err := assetInventoryAPI.DefaultApi.V1CloudChangePost(ctx, changes)
				if err == nil {
					assert.Equal(t, tc.expectedResponse, resp.StatusCode)
				} else if !tc.mustError {
					t.Logf("Error calling asset-inventory-api %s", err.Error())
				}
				assert.Equal(t, tc.mustError, err != nil)
				if tc.responseValidator != nil && err == nil {
					tc.responseValidator(t, changes) // call additional validator function if present
				}
			})
	}
}