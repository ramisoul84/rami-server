package handler

import "fmt"

// InternalError is a 5xx whose cause is logged but never shown to the client.
type InternalError struct {
	Message string // safe for the response body
	Err     error  // the cause, for the logs
}

func (e *InternalError) Error() string { return fmt.Sprintf("%s: %v", e.Message, e.Err) }
func (e *InternalError) Unwrap() error { return e.Err }

func internal(message string, err error) error { return &InternalError{Message: message, Err: err} }
