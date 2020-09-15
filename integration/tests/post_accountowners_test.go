// +build integration

package tests

import (
	"context"
	openapi "github.com/asecurityteam/asset-inventory-api/client"
	"github.com/stretchr/testify/assert"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"
)

func LookupNoAccountOwnerInResponse(t *testing.T, needle string, when time.Time) {
	t.Errorf("response does not contain account owner for resource %s at %s", needle, when)
}

func CheckAccountOwnersPresent(t *testing.T, accountOwnersExpected openapi.AccountOwner, changes openapi.CloudAssetChanges) {

	ctx := context.Background()
	ts := SampleAssetChanges().ChangeTime.Add(time.Minute) // best way I could think of to get time to query accounts by
	spl := strings.Split(changes.Arn, "/")
	resId := spl[len(spl)-1]
	type check struct {
		lookup   func(context.Context, string, time.Time) (openapi.CloudAssets, *http.Response, error)
		haystack string
	}
	tests := []check{ // add the check for lookup-by-resource-id
		{assetInventoryAPI.DefaultApi.V1CloudResourceidResourceidGet, resId},
	}
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
	// time.Sleep(1000 * time.Millisecond)
	for _, test := range tests { // run every created check as separate sub-test
		t.Run("Test lookup by: "+test.haystack, func(t *testing.T) {
			assets, httpRes, err := test.lookup(ctx, test.haystack, ts)
			if err != nil {
				LookupError(t, test.haystack, ts, err)
			} else if httpRes.StatusCode != 200 {
				LookupWrongCode(t, test.haystack, ts, *httpRes)
			} else if !AccountsInResponse(t, accountOwnersExpected, assets.Assets) {
				LookupNoAccountOwnerInResponse(t, test.haystack, ts)
			}
		})
	}
}

func AccountsInResponse(t *testing.T, needle openapi.AccountOwner, haystack []openapi.CloudAssetDetails) bool {
	for _, asset := range haystack {
		// TODO: handle errors if AccountOwner is nonexistent (is that possible?)
		if asset.AccountOwner.AccountId != needle.AccountId ||
			asset.AccountOwner.Owner != needle.Owner ||
			!accountChampionsSlicesEqual(t, needle.Champions, asset.AccountOwner.Champions) {
			return false
		}
	}
	return true
}

func accountChampionsSlicesEqual(t *testing.T, expected, actual []openapi.Person) bool {
	if len(expected) != len(actual) {
		// log error here
		t.Errorf("length of slices is not the same: expected %d, actual %d, ", len(expected), len(actual))
		return false
	}
	// sort both slices of champions by login so the comparison can be done per struct
	orderAccountChampionsByLogin(&expected)
	orderAccountChampionsByLogin(&actual)
	for i, champion := range expected {
		t.Logf("person login: "+champion.Login)
		if champion != actual[i] {
			return false
		}
	}
	return true
}

func orderAccountChampionsByLogin(champions *[]openapi.Person) {
	// Sort by login, preserving original order
	sort.SliceStable(*champions, func(i, j int) bool { return (*champions)[i].Login < (*champions)[j].Login })
}

func TestPostAccountOwners(t *testing.T) {
	testCases := map[string]struct{
		accountOwnersAdapter func(*openapi.SetAccountOwner)
		expectedResponse int
		mustError bool
		responseValidator func(*testing.T, openapi.AccountOwner, openapi.CloudAssetChanges)
	}{
		"Valid": {
			func(owner *openapi.SetAccountOwner) {},
			http.StatusCreated,
			false,
			CheckAccountOwnersPresent,
		},
	}
	for name, tc := range testCases{
		t.Run(addSchemaVersion(name), func(t *testing.T) {
			ctx := context.Background()
			setAccountOwners := SampleAccountOwner()
			tc.accountOwnersAdapter(&setAccountOwners)
			getAccountOwners := SampleGetAccountOwner() // note: this does not have changes from above adapter, it is a different data type
			resp, err := assetInventoryAPI.DefaultApi.V1AccountOwnerPost(ctx, setAccountOwners)
			if err == nil {
				assert.Equal(t, tc.expectedResponse, resp.StatusCode)
			} else if !tc.mustError {
				t.Errorf("Error calling asset-inventory-api %s", err.Error())
			}
			assert.Equal(t, tc.mustError, err != nil)
			if tc.responseValidator != nil {
				// post changes to ensure there is data for account validation
				changes := SampleAssetChanges() // get sample valid AssetChanges
				cloudChangeResp, cloudChangeErr := assetInventoryAPI.DefaultApi.V1CloudChangePost(ctx, changes)
				if cloudChangeErr != nil {
					t.Errorf("Error calling asset-inventory-api %s", cloudChangeErr.Error())
				} else if cloudChangeResp.StatusCode != http.StatusCreated {
					t.Errorf("Error posting changes to asset-inventory-api: %d", cloudChangeResp.StatusCode)
				}
				tc.responseValidator(t, getAccountOwners, changes)
			}
		})
	}
}