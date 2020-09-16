// +build integration

package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//TODO add tests for resource IDs containing slashes, tracked in separate ticket

func TestLookupByResourceID(t *testing.T) {
	ctx := context.Background()
	chgAssign, chgRemove, api := Setup(t, ctx)
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

	RawUrlFollowupTests(t, "/v1/cloud/resourceid/"+resId, tsDuring)
}
