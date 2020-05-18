package domain

import (
	"time"
)

// CloudAssetChanges represent network changes to an asset and associated metadata
type CloudAssetChanges struct {
	Changes      []NetworkChanges
	ChangeTime   time.Time
	ResourceType string
	AccountID    string
	Region       string
	ARN          string
	Tags         map[string]string
}

// NetworkChanges represent changes to an asset's IP addresses or associated host names
type NetworkChanges struct {
	PrivateIPAddresses []string
	PublicIPAddresses  []string
	Hostnames          []string
	ChangeType         string
}

// CloudAssetDetails represent an asset and associated metadata
type CloudAssetDetails struct {
	PrivateIPAddresses []string
	PublicIPAddresses  []string
	Hostnames          []string
	ResourceType       string
	AccountID          string
	Region             string
	ARN                string
	Tags               map[string]string
}

// AccountOwner represents an AWS account with its owner and account champions
type AccountOwner struct {
	AccountID string
	Owner     Person
	Champions []Person
}

// Person represents details about an Atlassian member
type Person struct {
	Name  string
	Login string
	Email string
	Valid bool
}
