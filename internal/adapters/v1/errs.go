package adapters_v1

import "fmt"

// ConversionError is returned when a value cannot be converted between the
// client [values.Value] and protocol v1 [value.V].
type ConversionError struct {
	op  string
	msg string
	err error
}

func newConversionError(op, msg string) ConversionError {
	return ConversionError{op: op, msg: msg}
}

func newConversionErrorf(op, format string, args ...any) ConversionError {
	return ConversionError{op: op, msg: fmt.Sprintf(format, args...)}
}

// Op returns the conversion direction, such as "from" or "to".
func (e ConversionError) Op() string { return e.op }

// Error implements the error interface.
func (e ConversionError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("adapters/v1: %s: %s: %v", e.op, e.msg, e.err)
	}
	return fmt.Sprintf("adapters/v1: %s: %s", e.op, e.msg)
}

// Unwrap returns the underlying cause, if any.
func (e ConversionError) Unwrap() error { return e.err }
