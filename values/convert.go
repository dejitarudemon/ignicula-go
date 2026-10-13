package values

import "fmt"

// ErrUnsupportedConversion is returned when [Convert] cannot produce the
// requested type from the source value.
type ErrUnsupportedConversion struct {
	From Type
	To   Type
}

func (e ErrUnsupportedConversion) Error() string {
	return fmt.Sprintf("cannot convert %s to %s", e.From, e.To)
}

// Convert returns v as type to.
// Bytes and String convert to each other by interpreting the payload as UTF-8
// text. Identical types are returned unchanged. Other conversions fail with
// [ErrUnsupportedConversion].
func Convert(v Value, to Type) (Value, error) {
	if v == nil {
		return nil, fmt.Errorf("nil value")
	}
	if v.Type() == to {
		return v, nil
	}

	switch to {
	case Bytes:
		switch src := v.(type) {
		case StringValue:
			return BytesValue([]byte(src)), nil
		}
	case String:
		switch src := v.(type) {
		case BytesValue:
			return StringValue(src), nil
		}
	}

	return nil, ErrUnsupportedConversion{From: v.Type(), To: to}
}
