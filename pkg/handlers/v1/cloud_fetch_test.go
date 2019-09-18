package v1

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func newFetchByIPHandler(fetcher domain.CloudAssetByIPFetcher) *CloudFetchByIPHandler {
	return &CloudFetchByIPHandler{
		LogFn:   testLogFn,
		StatFn:  testStatFn,
		Fetcher: fetcher,
	}
}

func newFetchByHostnameHandler(fetcher domain.CloudAssetByHostnameFetcher) *CloudFetchByHostnameHandler {
	return &CloudFetchByHostnameHandler{
		LogFn:   testLogFn,
		StatFn:  testStatFn,
		Fetcher: fetcher,
	}
}

func newCloudFetchAllAssetsByTimeHandler(fetcher domain.CloudAllAssetsByTimeFetcher) *CloudFetchAllAssetsByTimeHandler {
	return &CloudFetchAllAssetsByTimeHandler{
		LogFn:   testLogFn,
		StatFn:  testStatFn,
		Fetcher: fetcher,
	}
}

func validFetchByIPInput() CloudAssetFetchByIPParameters {
	return CloudAssetFetchByIPParameters{
		IPAddress: "1.1.1.1",
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}
}

func validFetchByHostnameInput() CloudAssetFetchByHostnameParameters {
	return CloudAssetFetchByHostnameParameters{
		Hostname:  "hostname",
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}
}

func validFetchAllByTimestampInput() CloudAssetFetchAllByTimestampParameters {
	var count uint = 100
	var offset uint = 0
	return CloudAssetFetchAllByTimestampParameters{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Count:     count,
		Offset:    offset,
		Type:      awsEC2,
	}
}

func TestCloudFetchAllAssetsByTimeInvalidDate(t *testing.T) {
	input := validFetchAllByTimestampInput()
	input.Timestamp = "not a valid date"
	_, err := newCloudFetchAllAssetsByTimeHandler(nil).Handle(context.Background(), input)
	require.NotNil(t, err)
}

func TestCloudFetchAllAssetsByTimeInvalidCount(t *testing.T) {
	input := validFetchAllByTimestampInput()
	input.Count = 0
	_, err := newCloudFetchAllAssetsByTimeHandler(nil).Handle(context.Background(), input)
	require.NotNil(t, err)
}

func TestCloudFetchAllAssetsByTimeInvalidType(t *testing.T) {
	input := validFetchAllByTimestampInput()
	input.Type = "Very wrong type"
	_, err := newCloudFetchAllAssetsByTimeHandler(nil).Handle(context.Background(), input)
	require.NotNil(t, err)
}

func TestCloudFetchAllAssetsByTimeStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAllAssetsByTimeFetcher(ctrl)
	input := validFetchAllByTimestampInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchAll(gomock.Any(), ts, input.Count, input.Offset, input.Type).Return([]domain.CloudAssetDetails{}, errors.New(""))

	_, e := newCloudFetchAllAssetsByTimeHandler(fetcher).Handle(context.Background(), input)
	require.NotNil(t, e)
}
func TestCloudFetchAllAssetsByTimeNoResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAllAssetsByTimeFetcher(ctrl)
	input := validFetchAllByTimestampInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchAll(gomock.Any(), ts, input.Count, input.Offset, input.Type).Return([]domain.CloudAssetDetails{}, nil)

	_, e := newCloudFetchAllAssetsByTimeHandler(fetcher).Handle(context.Background(), input)
	require.NotNil(t, e)
}

func TestFetchByIPInvalidInput(t *testing.T) {
	tc := []struct {
		name  string
		input CloudAssetFetchByIPParameters
	}{
		{
			name:  "empty timestamp",
			input: CloudAssetFetchByIPParameters{Timestamp: ""},
		},
		{
			name:  "invalid timestamp",
			input: CloudAssetFetchByIPParameters{Timestamp: "foo"},
		},
		{
			name:  "no ipAddress",
			input: CloudAssetFetchByIPParameters{Timestamp: time.Now().Format(time.RFC3339Nano)},
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(*testing.T) {
			_, e := newFetchByIPHandler(nil).Handle(context.Background(), tt.input)
			assert.NotNil(t, e)

			_, ok := e.(InvalidInput)
			assert.True(t, ok)
		})
	}
}

func TestFetchByIPStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAssetByIPFetcher(ctrl)
	input := validFetchByIPInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchByIP(gomock.Any(), ts, input.IPAddress).Return([]domain.CloudAssetDetails{}, errors.New(""))

	_, e := newFetchByIPHandler(fetcher).Handle(context.Background(), input)
	assert.NotNil(t, e)
}

func TestFetchByIPNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAssetByIPFetcher(ctrl)
	input := validFetchByIPInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchByIP(gomock.Any(), ts, input.IPAddress).Return([]domain.CloudAssetDetails{}, nil)

	_, e := newFetchByIPHandler(fetcher).Handle(context.Background(), input)
	assert.NotNil(t, e)
}

func TestFetchByIPSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAssetByIPFetcher(ctrl)
	input := validFetchByIPInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	output := []domain.CloudAssetDetails{
		{
			PrivateIPAddresses: []string{input.IPAddress},
			PublicIPAddresses:  []string{input.IPAddress},
			Hostnames:          []string{"foo"},
		},
	}
	fetcher.EXPECT().FetchByIP(gomock.Any(), ts, input.IPAddress).Return(output, nil)

	asset, e := newFetchByIPHandler(fetcher).Handle(context.Background(), input)
	assert.Nil(t, e)
	assert.NotNil(t, asset)
}

func TestFetchByHostnameInvalidInput(t *testing.T) {
	tc := []struct {
		name  string
		input CloudAssetFetchByHostnameParameters
	}{
		{
			name:  "empty timestamp",
			input: CloudAssetFetchByHostnameParameters{Timestamp: ""},
		},
		{
			name:  "invalid timestamp",
			input: CloudAssetFetchByHostnameParameters{Timestamp: "foo"},
		},
		{
			name:  "no hostname",
			input: CloudAssetFetchByHostnameParameters{Timestamp: time.Now().Format(time.RFC3339Nano)},
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(*testing.T) {
			_, e := newFetchByHostnameHandler(nil).Handle(context.Background(), tt.input)
			assert.NotNil(t, e)

			_, ok := e.(InvalidInput)
			assert.True(t, ok)
		})
	}
}

func TestFetchByHostnameStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAssetByHostnameFetcher(ctrl)
	input := validFetchByHostnameInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchByHostname(gomock.Any(), ts, input.Hostname).Return([]domain.CloudAssetDetails{}, errors.New(""))

	_, e := newFetchByHostnameHandler(fetcher).Handle(context.Background(), input)
	assert.NotNil(t, e)
}

func TestFetchByHostnameNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAssetByHostnameFetcher(ctrl)
	input := validFetchByHostnameInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchByHostname(gomock.Any(), ts, input.Hostname).Return([]domain.CloudAssetDetails{}, nil)

	_, e := newFetchByHostnameHandler(fetcher).Handle(context.Background(), input)
	assert.NotNil(t, e)
}

func TestFetchByHostnameSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAssetByHostnameFetcher(ctrl)
	input := validFetchByHostnameInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	output := []domain.CloudAssetDetails{
		{
			PrivateIPAddresses: []string{"10.2.3.4"},
			PublicIPAddresses:  []string{"1.2.3.4"},
			Hostnames:          []string{"foo"},
		},
	}
	fetcher.EXPECT().FetchByHostname(gomock.Any(), ts, input.Hostname).Return(output, nil)

	asset, e := newFetchByHostnameHandler(fetcher).Handle(context.Background(), input)
	assert.Nil(t, e)
	assert.NotNil(t, asset)
}

func TestExtractOutput(t *testing.T) {
	tc := []struct {
		name     string
		input    []domain.CloudAssetDetails
		expected CloudAssets
	}{
		{
			name:     "no assets",
			input:    []domain.CloudAssetDetails{},
			expected: CloudAssets{Assets: make([]CloudAssetDetails, 0)},
		},
		{
			name: "empty arrays",
			input: []domain.CloudAssetDetails{
				{
					ResourceType: "resourceType",
					AccountID:    "accountId",
					Region:       "Region",
					ARN:          "arn",
				},
			},
			expected: CloudAssets{
				Assets: []CloudAssetDetails{
					{
						PrivateIPAddresses: make([]string, 0),
						PublicIPAddresses:  make([]string, 0),
						Hostnames:          make([]string, 0),
						ResourceType:       "resourceType",
						AccountID:          "accountId",
						Region:             "Region",
						ARN:                "arn",
						Tags:               make(map[string]string),
					},
				},
			},
		},
		{
			name: "multiple assets",
			input: []domain.CloudAssetDetails{
				{
					Hostnames:          []string{"hostname"},
					PublicIPAddresses:  []string{"1.1.1.1"},
					PrivateIPAddresses: []string{"10.1.1.1"},
				},
				{
					Hostnames:          []string{"hostname"},
					PublicIPAddresses:  []string{"2.2.2.2"},
					PrivateIPAddresses: []string{"10.2.2.2"},
				},
			},
			expected: CloudAssets{
				Assets: []CloudAssetDetails{
					{
						PrivateIPAddresses: []string{"10.1.1.1"},
						PublicIPAddresses:  []string{"1.1.1.1"},
						Hostnames:          []string{"hostname"},
						Tags:               make(map[string]string),
					},
					{
						PrivateIPAddresses: []string{"10.2.2.2"},
						PublicIPAddresses:  []string{"2.2.2.2"},
						Hostnames:          []string{"hostname"},
						Tags:               make(map[string]string),
					},
				},
			},
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(*testing.T) {
			actual := extractOutput(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
