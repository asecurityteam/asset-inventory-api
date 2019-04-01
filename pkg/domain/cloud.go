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
	ResourceID   string
	Tags         map[string]string
}

// NetworkChanges represent changes to an asset's IP addresses or associated host names
type NetworkChanges struct {
	PrivateIPAddresses []string
	PublicIPAddresses  []string
	Hostnames          []string
	ChangeType         string
}
