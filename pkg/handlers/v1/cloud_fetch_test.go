package v1

import (
	"context"
	"encoding/base32"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/asecurityteam/asset-inventory-api/pkg/domain"
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

func newFetchByARNIDHandler(fetcher domain.CloudAssetByARNIDFetcher) *CloudFetchByARNIDHandler {
	return &CloudFetchByARNIDHandler{
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

func newCloudFetchAllAssetsByTimePageHandler(fetcher domain.CloudAllAssetsByTimeFetcher) *CloudFetchAllAssetsByTimePageHandler {
	return &CloudFetchAllAssetsByTimePageHandler{
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

func validFetchByARNIDInput() CloudAssetFetchByARNIDParameters {
	return CloudAssetFetchByARNIDParameters{
		ARN:       "arnid",
		Timestamp: time.Now().Format(time.RFC3339Nano),
	}
}

func validFetchAllByTimestampInput() CloudAssetFetchAllByTimestampParameters {
	var count uint = 100
	var offset uint
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

func TestCloudFetchAllAssetsByTimePageInvalidDate(t *testing.T) {
	input := validFetchAllByTimestampInput()
	input.Timestamp = "not a valid date"
	pageToken, _ := input.toNextPageToken()
	_, err := newCloudFetchAllAssetsByTimePageHandler(nil).Handle(
		context.Background(),
		CloudAssetFetchAllByTimeStampPageParameters{PageToken: pageToken},
	)
	require.NotNil(t, err)
}

func TestCloudFetchAllAssetsByTimePageInvalidCount(t *testing.T) {
	input := validFetchAllByTimestampInput()
	input.Count = 0
	pageToken, _ := input.toNextPageToken()
	_, err := newCloudFetchAllAssetsByTimePageHandler(nil).Handle(
		context.Background(),
		CloudAssetFetchAllByTimeStampPageParameters{PageToken: pageToken})
	require.NotNil(t, err)
}

func TestCloudFetchAllAssetsByTimePageInvalidType(t *testing.T) {
	input := validFetchAllByTimestampInput()
	input.Type = "Very wrong type"
	pageToken, _ := input.toNextPageToken()
	_, err := newCloudFetchAllAssetsByTimePageHandler(nil).Handle(
		context.Background(),
		CloudAssetFetchAllByTimeStampPageParameters{PageToken: pageToken})
	require.NotNil(t, err)
}

func TestCloudFetchAllAssetsByTimePageStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAllAssetsByTimeFetcher(ctrl)
	input := validFetchAllByTimestampInput()
	pageToken, _ := input.toNextPageToken()
	input.Offset += input.Count //emulate the paging
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchAll(gomock.Any(), ts, input.Count, input.Offset, input.Type).Return([]domain.CloudAssetDetails{}, errors.New(""))

	_, e := newCloudFetchAllAssetsByTimePageHandler(fetcher).Handle(context.Background(), CloudAssetFetchAllByTimeStampPageParameters{PageToken: pageToken})
	require.NotNil(t, e)
}
func TestCloudFetchAllAssetsByTimePageNoResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAllAssetsByTimeFetcher(ctrl)
	input := validFetchAllByTimestampInput()
	pageToken, _ := input.toNextPageToken()
	input.Offset += input.Count //emulate the paging
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchAll(gomock.Any(), ts, input.Count, input.Offset, input.Type).Return([]domain.CloudAssetDetails{}, nil)

	_, e := newCloudFetchAllAssetsByTimePageHandler(fetcher).Handle(context.Background(), CloudAssetFetchAllByTimeStampPageParameters{PageToken: pageToken})
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

func TestFetchByARNIDInvalidInput(t *testing.T) {
	tc := []struct {
		name  string
		input CloudAssetFetchByARNIDParameters
	}{
		{
			name:  "empty timestamp",
			input: CloudAssetFetchByARNIDParameters{Timestamp: ""},
		},
		{
			name:  "invalid timestamp",
			input: CloudAssetFetchByARNIDParameters{Timestamp: "foo"},
		},
		{
			name:  "no ARN ID",
			input: CloudAssetFetchByARNIDParameters{Timestamp: time.Now().Format(time.RFC3339Nano)},
		},
	}

	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(*testing.T) {
			_, e := newFetchByARNIDHandler(nil).Handle(context.Background(), tt.input)
			assert.NotNil(t, e)

			_, ok := e.(InvalidInput)
			assert.True(t, ok)
		})
	}
}

func TestFetchByARNIDNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAssetByARNIDFetcher(ctrl)
	input := validFetchByARNIDInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchByARNID(gomock.Any(), ts, input.ARN).Return([]domain.CloudAssetDetails{}, nil)

	_, e := newFetchByARNIDHandler(fetcher).Handle(context.Background(), input)
	assert.NotNil(t, e)
}

func TestFetchByARNIDStorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAssetByARNIDFetcher(ctrl)
	input := validFetchByARNIDInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	fetcher.EXPECT().FetchByARNID(gomock.Any(), ts, input.ARN).Return([]domain.CloudAssetDetails{}, errors.New(""))

	_, e := newFetchByARNIDHandler(fetcher).Handle(context.Background(), input)
	assert.NotNil(t, e)
}

func TestFetchByARNIDSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := NewMockCloudAssetByARNIDFetcher(ctrl)
	input := validFetchByARNIDInput()
	ts, _ := time.Parse(time.RFC3339Nano, input.Timestamp)
	output := []domain.CloudAssetDetails{
		{
			PrivateIPAddresses: []string{"10.2.3.4"},
			PublicIPAddresses:  []string{"1.2.3.4"},
			Hostnames:          []string{"foo"},
			ARN:                "arnid",
			AccountOwner: domain.AccountOwner{
				AccountID: "abc123",
				Owner: domain.Person{
					Name:  "fake name",
					Login: "fake",
					Email: "fake@atlassian.com",
					Valid: true,
				},
			},
		},
	}
	fetcher.EXPECT().FetchByARNID(gomock.Any(), ts, input.ARN).Return(output, nil)

	asset, e := newFetchByARNIDHandler(fetcher).Handle(context.Background(), input)
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
					AccountOwner: domain.AccountOwner{
						AccountID: "accountID",
						Owner: domain.Person{
							Name:  "name",
							Login: "login",
							Email: "email@atlassian.com",
							Valid: false,
						},
					},
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
						AccountOwner: domain.AccountOwner{
							AccountID: "accountID",
							Owner: domain.Person{
								Name:  "name",
								Login: "login",
								Email: "email@atlassian.com",
								Valid: false,
							},
							Champions: make([]domain.Person, 0),
						},
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
					AccountOwner: domain.AccountOwner{
						AccountID: "accountID",
						Owner: domain.Person{
							Name:  "name",
							Login: "login",
							Email: "email@atlassian.com",
							Valid: true,
						},
						Champions: []domain.Person{
							{
								Name:  "name",
								Login: "login",
								Email: "email@atlassian.com",
								Valid: true,
							},
							{
								Name:  "name2",
								Login: "login2",
								Email: "email2@atlassian.com",
								Valid: true,
							},
						},
					},
				},
				{
					Hostnames:          []string{"hostname"},
					PublicIPAddresses:  []string{"2.2.2.2"},
					PrivateIPAddresses: []string{"10.2.2.2"},
					AccountOwner: domain.AccountOwner{
						AccountID: "accountID2",
						Owner: domain.Person{
							Name:  "name",
							Login: "login",
							Email: "email@atlassian.com",
							Valid: true,
						},
						Champions: []domain.Person{
							{
								Name:  "name",
								Login: "login",
								Email: "email@atlassian.com",
								Valid: true,
							},
							{
								Name:  "name2",
								Login: "login2",
								Email: "email2@atlassian.com",
								Valid: true,
							},
						},
					},
				},
			},
			expected: CloudAssets{
				Assets: []CloudAssetDetails{
					{
						PrivateIPAddresses: []string{"10.1.1.1"},
						PublicIPAddresses:  []string{"1.1.1.1"},
						Hostnames:          []string{"hostname"},
						Tags:               make(map[string]string),
						AccountOwner: domain.AccountOwner{
							AccountID: "accountID",
							Owner: domain.Person{
								Name:  "name",
								Login: "login",
								Email: "email@atlassian.com",
								Valid: true,
							},
							Champions: []domain.Person{
								{
									Name:  "name",
									Login: "login",
									Email: "email@atlassian.com",
									Valid: true,
								},
								{
									Name:  "name2",
									Login: "login2",
									Email: "email2@atlassian.com",
									Valid: true,
								},
							},
						},
					},
					{
						PrivateIPAddresses: []string{"10.2.2.2"},
						PublicIPAddresses:  []string{"2.2.2.2"},
						Hostnames:          []string{"hostname"},
						Tags:               make(map[string]string),
						AccountOwner: domain.AccountOwner{
							AccountID: "accountID2",
							Owner: domain.Person{
								Name:  "name",
								Login: "login",
								Email: "email@atlassian.com",
								Valid: true,
							},
							Champions: []domain.Person{
								{
									Name:  "name",
									Login: "login",
									Email: "email@atlassian.com",
									Valid: true,
								},
								{
									Name:  "name2",
									Login: "login2",
									Email: "email2@atlassian.com",
									Valid: true,
								},
							},
						},
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

func Test_validateAssetType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"ValidEC2", awsEC2, awsEC2, false},
		{"ValidALB", awsALB, awsALB, false},
		{"ValidELB", awsELB, awsELB, false},
		{"Invalid", "not a valid asset type", "", true},
		{"Empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateAssetType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAssetType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateAssetType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fetchAllByTimeStampParametersForToken(t *testing.T) {
	validParameters := validFetchAllByTimestampInput()
	validToken, _ := validParameters.toNextPageToken()
	validParameters.Offset += validParameters.Count //next page
	brokenJS := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("this is not json"))
	tests := []struct {
		name    string
		token   string
		want    *CloudAssetFetchAllByTimestampParameters
		wantErr bool
	}{
		{"not base32", "this is not base32 $@&%#@&*^*&%(*", nil, true},
		{"not valid json", brokenJS, nil, true},
		{"valid", validToken, &validParameters, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchAllByTimeStampParametersForToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchAllByTimeStampParametersForToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchAllByTimeStampParametersForToken() got = %v, want %v", got, tt.want)
			}
		})
	}
}
