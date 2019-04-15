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

// DBCommitError is logged when there is a failure to commit a transaction to the database
type DBCommitError struct {
	Message string `logevent:"message,default=database-transaction-commit-error"`
	Reason  string `logevent:"reason"`
}

// DBRollbackError is logged when there is a failure to rollback a failed transaction to the database
type DBRollbackError struct {
	Message string `logevent:"message,default=database-transaction-rollback-error"`
	Reason  string `logevent:"reason"`
}

// DBInsertError is logged when there is a failure to insert to the database
type DBInsertError struct {
	Message string `logevent:"message,default=database-insert-error"`
	Reason  string `logevent:"reason"`
}

// DBCreateTableError is logged when there is a failure to create a table in the database
type DBCreateTableError struct {
	Message string `logevent:"message,default=database-create-table-error"`
	Reason  string `logevent:"reason"`
}

// DBCreateIndexError is logged when there is a failure to create an index in the database
type DBCreateIndexError struct {
	Message string `logevent:"message,default=database-create-index-error"`
	Reason  string `logevent:"reason"`
}

// DBBeginTransactionError is logged when there is a failure to begin a transaction in the database
type DBBeginTransactionError struct {
	Message string `logevent:"message,default=database-begin-transaction-error"`
	Reason  string `logevent:"reason"`
}

// DBSelectError is logged when there is a failure to run a SELECT query to the database
type DBSelectError struct {
	Message string `logevent:"message,default=database-select-error"`
	Reason  string `logevent:"reason"`
}

// DBRowScanError is logged when there is a failure to scan a resultant row from a select query to the database
type DBRowScanError struct {
	Message string `logevent:"message,default=database-row-scan-error"`
	Reason  string `logevent:"reason"`
}
