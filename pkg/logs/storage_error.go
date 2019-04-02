package logs

// StorageError is logged when there is a failure with the backend dependency
type StorageError struct {
	Message string `logevent:"message,default=storage-error"`
	Reason  string `logevent:"reason"`
}
