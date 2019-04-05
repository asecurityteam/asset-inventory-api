package domain

import "time"

// QueryResult represent a result row of database query
type QueryResult struct {
	Hostname  string
	IPAddress string
	IsPublic  bool
	IsJoin    bool
	Timestamp time.Time
}
