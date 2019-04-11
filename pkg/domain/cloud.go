package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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
	ARN          string
	Tags         InboundResourceTags
}

// InboundResourceTags represents the contents of the metadata (CloudAssetChanges.Tags) saved with the resource
type InboundResourceTags map[string]interface{}

// Value implements the driver.Valuer interface to marshal InboundResourceTags to []byte
func (p InboundResourceTags) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, err
}

// Scan implements the sql.Scanner interface to unmarshal the database data into a map, which is the same type as received and written into the database
func (p *InboundResourceTags) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("Type assertion .([]byte) failed")
	}

	var i interface{}
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}

	if i != nil {
		*p, ok = i.(map[string]interface{})
		if !ok {
			return errors.New("Type assertion .(map[string]interface{}) failed")
		}
	}

	return nil
}

// NetworkChanges represent changes to an asset's IP addresses or associated host names
type NetworkChanges struct {
	PrivateIPAddresses []string
	PublicIPAddresses  []string
	Hostnames          []string
	ChangeType         string
}
