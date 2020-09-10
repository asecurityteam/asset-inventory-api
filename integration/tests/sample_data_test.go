// +build integration

package tests

import (
	"fmt"
	"strings"
	"time"

	openapi "github.com/asecurityteam/asset-inventory-api/client"
)

// ensure we use the same value for account owner and sample resources as there is
// a known issue with return format of accounts w/o owner/champions set
const accountID = "012345678901"

func SampleAssetChange() openapi.CloudAssetChange {
	return openapi.CloudAssetChange{
		PrivateIpAddresses: []string{"10.0.0.1", "10.0.0.2"},
		PublicIpAddresses:  []string{"8.8.8.8", "8.8.4.4"},
		//even though Hostnames is a list, there can be only one of them.
		//TODO - reflect in schema in api.yaml that we do not accept more than one hostname
		Hostnames:        []string{"myhostname.us-west-1.amazonaws.com"},
		RelatedResources: []string{},
		ChangeType:       "ADDED",
	}
}

func SampleAssetChanges() openapi.CloudAssetChanges {
	r := openapi.CloudAssetChanges{
		Changes:      []openapi.CloudAssetChange{SampleAssetChange()},
		ChangeTime:   time.Date(2018, 01, 12, 22, 51, 48, 324359102, time.UTC),
		ResourceType: "AWS:EC2:Instance",
		AccountId:    accountID,
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
			//we are not checking tags or account owner data as these might not match in some cases
			return true
		}
	}
	return false
}

func SampleAccountOwner() openapi.SetAccountOwner {
	alice := openapi.SetPerson{
		Name:  "Alice User",
		Login: "auser",
		Email: "auser@atlassian.com",
		Valid: true,
	}
	john := openapi.SetPerson{
		Name:  "John Smith",
		Login: "jsmith",
		Email: "jsmith@atlassian.com",
		Valid: true,
	}
	accountOwner := openapi.SetAccountOwner{
		AccountId: accountID,
		Owner:     alice,
		Champions: []openapi.SetPerson{alice, john},
	}
	return accountOwner
}
