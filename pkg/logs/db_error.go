package logs

// DBOpenError is logged when there is a failure to "Open" the database
type DBOpenError struct {
	Message string `logevent:"message,default=database-open-error"`
	Reason  string `logevent:"reason"`
}

// DBPingError is logged when there is a failure to "Ping" the database, which is required to establish Postgres connection
type DBPingError struct {
	Message string `logevent:"message,default=database-ping-error"`
	Reason  string `logevent:"reason"`
}
