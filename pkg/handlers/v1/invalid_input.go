package v1

import (
	"fmt"
)

// InvalidInput is an error indicating the request was malformed
type InvalidInput struct {
	Field string
	Cause error
}

func (i InvalidInput) Error() string {
	return fmt.Sprintf("the value for field %s was invalid: %s", i.Field, i.Cause.Error())
}
