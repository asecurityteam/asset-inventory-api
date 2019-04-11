package domain

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// NetworkChangeEvent represent a result row of database query
type NetworkChangeEvent struct {
	ResourceID string
	IPAddress  string
	Hostname   sql.NullString
	IsPublic   bool
	IsJoin     bool
	Timestamp  time.Time
	AccountID  string
	Region     string
	Type       string
	Tags       ResourceTags
}

// ResourceTags represents the contents of the metadata (CloudAssetChanges.Tags) saved with the resource
type ResourceTags map[string]interface{}

// Value implements the driver.Valuer interface to marshal ResourceTags to []byte
func (p ResourceTags) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, err
}

// Scan implements the sql.Scanner interface to unmarshal the database data into a map, which is the same type as received and written into the database
func (p *ResourceTags) Scan(src interface{}) error {
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
