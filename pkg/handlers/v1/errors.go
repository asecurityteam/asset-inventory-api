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

// NotFound is an error indicating no assets with the given ID were found
type NotFound struct {
	ID string
}

func (n NotFound) Error() string {
	return fmt.Sprintf("resource %s not found", n.ID)
}
