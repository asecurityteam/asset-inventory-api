package domain

import (
	"database/sql"
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
	Tags       map[string]string
}
