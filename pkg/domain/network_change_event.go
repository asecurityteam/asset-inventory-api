package domain

import (
	"time"
)

// NetworkChangeEvent represent a result row of database query
type NetworkChangeEvent struct {
	ResourceID string
	IPAddress  string
	Hostname   string
	IsPublic   bool
	IsJoin     bool
	Timestamp  time.Time
	AccountID  string
	Region     string
	Type       string
	Tags       map[string]string
}
