package errors

import (
	"fmt"
)

// configurationError errors are raised when the configuration is incorrect e.g. the
// broker administrator has made a mistake.  An example could be a binding missing for
// a particular service plan.
type configurationError struct {
	message string
}

// NewConfigurationError returns a new configuration error formatted like fmt.Errorf.
func NewConfigurationError(message string, arguments ...interface{}) error {
	return &configurationError{message: fmt.Sprintf(message, arguments...)}
}

// IsConfigurationError returns whether an error is a configuration error.
func IsConfigurationError(err error) bool {
	if _, ok := err.(*configurationError); !ok {
		return false
	}
	return true
}

// Error returns the configuration error string.
func (e *configurationError) Error() string {
	return e.message
}

// queryError errors are raised when the query is incorrect e.g. the broker client
// has made a mistake.  An example could be a malformed URL query.
type queryError struct {
	message string
}

// NewQueryError returns a new query error formatted like fmt.Errorf.
func NewQueryError(message string, arguments ...interface{}) error {
	return &queryError{message: fmt.Sprintf(message, arguments...)}
}

// IsQueryError returns whether an error is a query error.
func IsQueryError(err error) bool {
	if _, ok := err.(*queryError); !ok {
		return false
	}
	return true
}

// Error returns the query error string.
func (e *queryError) Error() string {
	return e.message
}

// parameterError errors are raised when the request parameters are incorrect e.g.
// the broker client has made a mistake.  An example could be a missing service plan.
type parameterError struct {
	message string
}

// NewParameterError returns a new parameter error formatted like fmt.Errorf.
func NewParameterError(message string, arguments ...interface{}) error {
	return &parameterError{message: fmt.Sprintf(message, arguments...)}
}

// IsParameterError returns whether an error is a parameter error.
func IsParameterError(err error) bool {
	if _, ok := err.(*parameterError); !ok {
		return false
	}
	return true
}

// Error returns the parameter error string.
func (e *parameterError) Error() string {
	return e.message
}

// validationError errors are raised when the parameter validation fails e.g.
// the broker client has made a mistake.
type validationError struct {
	message string
}

// NewValidationError returns a new validation error formatted like fmt.Errorf.
func NewValidationError(message string, arguments ...interface{}) error {
	return &validationError{message: fmt.Sprintf(message, arguments...)}
}

// IsValidationError returns whether an error is a validation error.
func IsValidationError(err error) bool {
	if _, ok := err.(*validationError); !ok {
		return false
	}
	return true
}

// Error returns the validation error string.
func (e *validationError) Error() string {
	return e.message
}

// asyncRequiredError errors are raised when the API only supports asynchronous
// requests.
type asyncRequiredError struct {
	message string
}

// NewAsyncRequiredError returns a new async required error formatted like fmt.Errorf.
func NewAsyncRequiredError(message string, arguments ...interface{}) error {
	return &asyncRequiredError{message: fmt.Sprintf(message, arguments...)}
}

// IsAsyncRequiredError returns whether an error is an async required error.
func IsAsyncRequiredError(err error) bool {
	if _, ok := err.(*asyncRequiredError); !ok {
		return false
	}
	return true
}

// Error returns the async required error string.
func (e *asyncRequiredError) Error() string {
	return e.message
}

// resourceConflictError errors are raised when a resource already exists.
type resourceConflictError struct {
	message string
}

// NewResourceConflictError returns a new resource conflict error formatted like fmt.Errorf.
func NewResourceConflictError(message string, arguments ...interface{}) error {
	return &resourceConflictError{message: fmt.Sprintf(message, arguments...)}
}

// IsResourceConflictError returns whether an error is a resource conflict error.
func IsResourceConflictError(err error) bool {
	if _, ok := err.(*resourceConflictError); !ok {
		return false
	}
	return true
}

// Error returns the resource conflict error string.
func (e *resourceConflictError) Error() string {
	return e.message
}

// resourceNotFoundError errors are raised when a resource is not found.
type resourceNotFoundError struct {
	message string
}

// NewResourceNotFoundError returns a new resource not found error formatted like fmt.Errorf.
func NewResourceNotFoundError(message string, arguments ...interface{}) error {
	return &resourceNotFoundError{message: fmt.Sprintf(message, arguments...)}
}

// IsResourceNotFoundError returns whether an error is a resource not found error.
func IsResourceNotFoundError(err error) bool {
	if _, ok := err.(*resourceNotFoundError); !ok {
		return false
	}
	return true
}

// Error returns the resource not found error string.
func (e *resourceNotFoundError) Error() string {
	return e.message
}

// resourceGoneError errors are raised when a resource has already been deleted.
type resourceGoneError struct {
	message string
}

// NewResourceGoneError returns a new resource gone error formatted like fmt.Errorf.
func NewResourceGoneError(message string, arguments ...interface{}) error {
	return &resourceGoneError{message: fmt.Sprintf(message, arguments...)}
}

// IsResourceGoneError returns whether an error is a resource gone error.
func IsResourceGoneError(err error) bool {
	if _, ok := err.(*resourceGoneError); !ok {
		return false
	}
	return true
}

// Error returns the resource gone error string.
func (e *resourceGoneError) Error() string {
	return e.message
}
