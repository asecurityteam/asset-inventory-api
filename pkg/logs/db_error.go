package logs

// DBError is logged when there is a failure with the backend dependency
type DBError struct {
	Message string `logevent:"message,default=database-error"`
	Reason  string `logevent:"reason"`
}
