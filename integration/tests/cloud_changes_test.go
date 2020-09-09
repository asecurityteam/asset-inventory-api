// +build integration

package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
	"github.com/stretchr/testify/assert"
)

func SampleCloudAssetChange() openapi.CloudAssetChange {
	return openapi.CloudAssetChange{
		PrivateIpAddresses: []string{"10.0.0.1", "10.0.0.2"},
		PublicIpAddresses:  []string{"8.8.8.8", "8.8.4.4"},
		Hostnames:          []string{"a.amazonaws.com", "b.amazonaws.com"},
		RelatedResources:   []string{},
		ChangeType:         "ADDED",
	}
}

func SampleAssetChanges() openapi.CloudAssetChanges {
	r := openapi.CloudAssetChanges{
		Changes:      []openapi.CloudAssetChange{SampleCloudAssetChange()},
		ChangeTime:   time.Date(2018, 01, 12, 22, 51, 48, 324359102, time.UTC),
		ResourceType: "AWS:EC2:Instance",
		AccountId:    "001234567891011",
		Region:       "us-west-1",
		Tags:         map[string]string{"Name": "ValidInstance", "resource_owner": "jsmith"},
	}
	r.Arn = fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", r.Region, r.AccountId, "i-0123456789abcdef0")
	return r
}

func ChangesInResponse(needle openapi.CloudAssetChanges, haystack []openapi.CloudAssetDetails) bool {
	for _, asset := range haystack {
		if strings.HasSuffix(needle.Arn, asset.Arn) &&
			asset.ResourceType == needle.ResourceType &&
			asset.Region == needle.Region &&
			asset.AccountId == asset.AccountId {
			// nb we are not checking tags as these might not match in some cases
			return true
		}
	}
	return false
}

func LookupError(t *testing.T, needle string, when time.Time, err error) {
	t.Errorf("error looking up %s at %s : %s", needle, when, err.Error())
}

func LookupWrongCode(t *testing.T, needle string, when time.Time, response http.Response) {
	t.Errorf("unexpected response looking up %s at %s : %d", needle, when, response.StatusCode)
}

func LookupNoAssetInResponse(t *testing.T, needle string, when time.Time) {
	t.Errorf("response does not contain resource looking up %s at %s", needle, when)
}


func CheckChangesPresent(t *testing.T, changes openapi.CloudAssetChanges) {
	ctx := context.Background()
	ts := changes.ChangeTime.Add(time.Second)
	spl := strings.Split(changes.Arn, "/")
	resId := spl[len(spl)-1]
	// TODO remove copy/paste in validation
	t.Run("Test Lookup by ID:" + resId, func(t *testing.T) {
		assets, res, err := assetInventoryAPI.DefaultApi.V1CloudResourceidResourceidGet(ctx, resId, ts)
		if err != nil {
			LookupError(t, resId, ts, err)
		} else if res.StatusCode != 200 {
			LookupWrongCode(t, resId, ts, *res)
		} else if !ChangesInResponse(changes, assets.Assets) {
			LookupNoAssetInResponse(t, resId, ts)
		}
	})
	for _, change := range changes.Changes {
		for  _, publicIP := range change.PublicIpAddresses {
			t.Run("Test Lookup by public IP:" + publicIP, func(t *testing.T) {
				assets, res, err := assetInventoryAPI.DefaultApi.V1CloudIpIpAddressGet(ctx, publicIP, ts)
				if err != nil {
					LookupError(t, publicIP, ts, err)
				} else if res.StatusCode != 200 {
					LookupWrongCode(t, publicIP, ts, *res)
				} else if !ChangesInResponse(changes, assets.Assets) {
					LookupNoAssetInResponse(t, publicIP, ts)
				}
			})
		}
		for _, privateIP := range change.PrivateIpAddresses {
			t.Run("Test Lookup by private IP:" + privateIP, func(t *testing.T) {
				assets, res, err := assetInventoryAPI.DefaultApi.V1CloudIpIpAddressGet(ctx, privateIP, ts)
				if err != nil {
					LookupError(t, privateIP, ts, err)
				} else if res.StatusCode != 200 {
					LookupWrongCode(t, privateIP, ts, *res)
				} else if !ChangesInResponse(changes, assets.Assets) {
					LookupNoAssetInResponse(t, privateIP, ts)
				}
			})
		}
	}

}

func TestCloudChanges(t *testing.T) {
	testCases := map[string]struct {
		changesAdapter   func(*openapi.CloudAssetChanges)
		expectedResponse int
		mustError        bool
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
			http.StatusCreated, //TODO fix the code this should be 400
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
		},*/
		"PublicIPsWithoutHostnames": {
			func(changes *openapi.CloudAssetChanges) {
				changes.Changes[0].Hostnames = []string{}
			},
			//TODO fix/check the code. We should not accept semantically incorrect event. This should be 400.
			http.StatusCreated,
			false,
			nil,
		},
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
		t.Run(WithSchemaVersion(name),
			func(t *testing.T) {
				ctx := context.Background()
				changes := SampleAssetChanges()
				tc.changesAdapter(&changes)
				resp, err := assetInventoryAPI.DefaultApi.V1CloudChangePost(ctx, changes)
				if err == nil {
					assert.Equal(t, tc.expectedResponse, resp.StatusCode)
				} else if !tc.mustError {
					t.Logf("Error calling asset-inventory-api %s", err.Error())
				}
				assert.Equal(t, tc.mustError, err != nil)
				if tc.responseValidator!=nil && err==nil {
					tc.responseValidator(t, changes)
				}
			})
	}
}
