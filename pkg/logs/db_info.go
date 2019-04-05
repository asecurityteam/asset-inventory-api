package logs

// DBInfo is logged when there is some info to log with the backend dependency
type DBInfo struct {
	Message string `logevent:"message,default=database-info"`
	Reason  string `logevent:"reason"`
}
