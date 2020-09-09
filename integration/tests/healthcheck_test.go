// +build integration

package tests

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthcheck(t *testing.T) {
	tt := []struct {
		Name string
	}{
		{
			Name: "Health Check",
		},
	}
	for _, test := range tt {
		fn := func(t *testing.T) {
			ctx := context.Background()
			resp, err := assetInventoryAPI.DefaultApi.HealthcheckGet(ctx)
			assert.NoError(t, err, "Health check should produce no errors")
			assert.Equal(t, resp.StatusCode, http.StatusOK)
		}
		t.Run(addSchemaVersion(test.Name), fn)
	}
}
