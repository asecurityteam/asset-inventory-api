// +build integration

package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLookupByPrivateIP(t *testing.T) {
	ctx := context.Background()
	chgAssign, chgRemove, api := Setup(t, ctx)
	// extract IP address
	ip := chgAssign.Changes[0].PrivateIpAddresses[0]

	tsDuring := chgAssign.ChangeTime.Add(1 * time.Second)
	tsBefore := chgAssign.ChangeTime.Add(-1 * time.Second) // nb .Sub does something very different :confused:
	tsAfter := chgRemove.ChangeTime.Add(1 * time.Second)

	testCases := map[string]struct {
		ip        string
		ts        time.Time
		httpCode  int
		mustError bool
	}{
		"Valid": { // ideally we'd have response validation for happy path here, but the test for cloudchanges does it
			ip,
			tsDuring,
			http.StatusOK,
			false,
		},
		"TSBefore": {
			ip,
			tsBefore,
			http.StatusNotFound,
			true, // openapi bindings treat 404 as error
		},
		"TSAfter": {
			ip,
			tsAfter,
			http.StatusNotFound,
			true, // openapi bindings treat 404 as error
		},
		"IPEmpty": {
			"",
			tsDuring,
			http.StatusBadRequest,
			true,
		},
	}
	for name, tc := range testCases {
		t.Run(addSchemaVersion(name),
			func(t *testing.T) {
				_, httpRes, err := api.V1CloudIpIpAddressGet(ctx, tc.ip, tc.ts)
				if tc.mustError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
				assert.Equal(t, tc.httpCode, httpRes.StatusCode)
			})
	}

	RawUrlFollowupTests(t, "/v1/cloud/ip/"+ip, tsDuring)
}

func TestLookupByPublicIP(t *testing.T) {
	ctx := context.Background()
	chgAssign, chgRemove, api := Setup(t, ctx)
	// extract IP address
	ip := chgAssign.Changes[0].PublicIpAddresses[0]

	tsDuring := chgAssign.ChangeTime.Add(1 * time.Second)
	tsBefore := chgAssign.ChangeTime.Add(-1 * time.Second) // nb .Sub does something very different :confused:
	tsAfter := chgRemove.ChangeTime.Add(1 * time.Second)

	testCases := map[string]struct {
		ip        string
		ts        time.Time
		httpCode  int
		mustError bool
	}{
		"Valid": { // ideally we'd have response validation for happy path here, but the test for cloudchanges does it
			ip,
			tsDuring,
			http.StatusOK,
			false,
		},
		"TSBefore": {
			ip,
			tsBefore,
			http.StatusNotFound,
			true, // openapi bindings treat 404 as error
		},
		"TSAfter": {
			ip,
			tsAfter,
			http.StatusNotFound,
			true, // openapi bindings treat 404 as error
		},
		"IPEmpty": {
			"",
			tsDuring,
			http.StatusBadRequest,
			true,
		},
	}
	for name, tc := range testCases {
		t.Run(addSchemaVersion(name),
			func(t *testing.T) {
				_, httpRes, err := api.V1CloudIpIpAddressGet(ctx, tc.ip, tc.ts)
				if tc.mustError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
				assert.Equal(t, tc.httpCode, httpRes.StatusCode)
			})
	}

	RawUrlFollowupTests(t, "/v1/cloud/ip/"+ip, tsDuring)
}
