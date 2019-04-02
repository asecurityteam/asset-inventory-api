package logs

// InvalidInput is logged when the provided input is malformed
type InvalidInput struct {
	Message string `logevent:"message,default=invalid-input"`
	Reason  string `logevent:"reason"`
}
