package values

import "fmt"

// UntypedArrayValue is a heterogeneous sequence of [Value]s with type code [UntypedArray].
type UntypedArrayValue struct {
	Elems []Value
}

// Type returns [UntypedArray].
func (UntypedArrayValue) Type() Type {
	return UntypedArray
}

// NewUntypedArray builds an untyped array from values.
func NewUntypedArray(values ...Value) (UntypedArrayValue, error) {
	array := UntypedArrayValue{
		Elems: make([]Value, 0, len(values)),
	}

	for i, value := range values {
		if value == nil {
			return UntypedArrayValue{}, fmt.Errorf("%w at index %d", ErrNilArrayElem, i)
		}
		array.Elems = append(array.Elems, value)
	}

	return array, nil
}

// Append returns a copy of the array with values appended.
func (array UntypedArrayValue) Append(values ...Value) (UntypedArrayValue, error) {
	out := UntypedArrayValue{
		Elems: append([]Value(nil), array.Elems...),
	}
	for i, value := range values {
		if value == nil {
			return UntypedArrayValue{}, fmt.Errorf("%w at index %d", ErrNilArrayElem, i)
		}
		out.Elems = append(out.Elems, value)
	}
	return out, nil
}
