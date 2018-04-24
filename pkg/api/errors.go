package api

import (
	"fmt"
)

// API-specific errors
// The underlying error is not safe to return via the api
// The safe error is safe to return via the api
type apiSafeError struct {
	underlying error
	safe       error
}

func (e *apiSafeError) Error() string {
	return e.safe.Error()
}

// newAPISafeError is a factory to create a new apiSafeError
func newAPISafeError(underlying error, safeMessage string, args ...interface{}) error {
	safeError := fmt.Errorf(safeMessage, args...)
	return &apiSafeError{
		underlying: underlying,
		safe:       safeError,
	}
}

// safeError returns a safe error that can be returned and an underlying error to log
func safeError(err error, defaultMessage string, args ...interface{}) (safe error, underlying error) {
	// Asserts err is of type *apiSafeError
	// isSafeErr is whether the assert was successful
	fullErr, isSafeErr := err.(*apiSafeError)
	if isSafeErr {
		return fullErr, fullErr.underlying
	}

	// If the assert failed, return the default (safe) message
	//return errors.New(defaultMessage), err
	return fmt.Errorf(defaultMessage, args...), err
}
