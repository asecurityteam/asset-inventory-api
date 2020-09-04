// +build integration

package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
	"github.com/stretchr/testify/assert"
)

func SampleCloudAssetChange() openapi.CloudAssetChange{
	return openapi.CloudAssetChange{
		PrivateIpAddresses: []string{"10.0.0.1", "10.0.0.2"},
		PublicIpAddresses:  []string{"8.8.8.8","8.8.4.4"},
		Hostnames:          []string{"a.amazonaws.com","b.amazonaws.com"},
		RelatedResources:   []string{},
		ChangeType:         "ADDED",
	}
}

func SampleAssetChanges() openapi.CloudAssetChanges{
	r := openapi.CloudAssetChanges{
		Changes:      []openapi.CloudAssetChange{SampleCloudAssetChange()},
		ChangeTime:   time.Time{},
		ResourceType: "AWS:EC2:Instance",
		AccountId:    "001234567891011",
		Region:       "us-west-1",
		Tags:         map[string]string{"Name":"ValidInstance", "resource_owner":"jsmith"},
	}
	r.Arn = fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", r.Region, r.AccountId, "i-0123456789abcdef0")
	return r
}

func TestCloudChanges(t *testing.T) {
	testCases := map[string]struct{
		changesAdapter func(*openapi.CloudAssetChanges)
		expectedResponse int
		mustError bool
	}{
		"Valid" : {
			func(changes *openapi.CloudAssetChanges) {
			},
			http.StatusCreated,
			false,
		},
		/* "MissingAccountId": { Disabled. Turns PSQL into a pumpkin because 2 resources exist for same ARNID
			func(changes *openapi.CloudAssetChanges){
				changes.AccountId = ""
			},
			http.StatusCreated,
			false,
		},*/
		"BadResourceType": {
			func(changes *openapi.CloudAssetChanges) {
				changes.ResourceType = "MS:Windows:2000"
			},
			http.StatusCreated, //TODO fix the code this should be 400
			false,
		},
		/* "BadPrivateIP": { Disabled. Permanently poisons persistent PSQL connection. Need validation of IPs.
			func(changes *openapi.CloudAssetChanges) {
				changes.Changes[0].PrivateIpAddresses[0]="I am not an ip address"
			},
			http.StatusBadRequest,
			false,
		},*/
		"PublicIPsWithoutHostnames": {
			func(changes *openapi.CloudAssetChanges) {
				changes.Changes[0].Hostnames=[]string{}
			},
			//TODO fix/check the code. We should not accept semantically incorrect event. This should be 400.
			http.StatusCreated,
			false,
		},
		"InvalidChangeType": {
			func(changes *openapi.CloudAssetChanges) {
				changes.Changes[0].ChangeType="INVALIDTYPE"
			},
			http.StatusBadRequest,
			true,
		},
	}
	for name, tc := range testCases {
		t.Run(WithSchemaVersion(name),
			func(t *testing.T) {
				ctx := context.Background()
				changes := SampleAssetChanges()
				tc.changesAdapter(&changes)
				resp, err := assetInventoryAPI.DefaultApi.V1CloudChangePost(ctx, changes)
				if err==nil {
					assert.Equal(t, tc.expectedResponse, resp.StatusCode)
				} else if !tc.mustError {
					t.Logf("Error calling asset-inventory-api %s", err.Error())
				}
				assert.Equal(t, tc.mustError, err!=nil)
			})
	}
}
