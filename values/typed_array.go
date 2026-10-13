package values

import (
	"errors"
	"fmt"
)

// ErrNilArrayElem is returned when an array constructor or Append sees a nil element.
var ErrNilArrayElem = errors.New("nil array element")

// TypedArrayValue is a homogeneous sequence of [Value]s with type code [TypedArray].
type TypedArrayValue struct {
	ElemType Type
	Elems    []Value
}

// Type returns [TypedArray].
func (TypedArrayValue) Type() Type {
	return TypedArray
}

// NewTypedArray builds a typed array whose elements all have elemType.
func NewTypedArray(elemType Type, values ...Value) (TypedArrayValue, error) {
	array := TypedArrayValue{
		ElemType: elemType,
		Elems:    make([]Value, 0, len(values)),
	}

	for i, value := range values {
		if value == nil {
			return TypedArrayValue{}, fmt.Errorf("%w at index %d", ErrNilArrayElem, i)
		}
		if value.Type() != elemType {
			return TypedArrayValue{}, fmt.Errorf(
				"typed array element type mismatch at index %d: want %s, got %s",
				i, elemType, value.Type(),
			)
		}
		array.Elems = append(array.Elems, value)
	}

	return array, nil
}

// Append returns a copy of the array with values appended.
func (array TypedArrayValue) Append(values ...Value) (TypedArrayValue, error) {
	out := TypedArrayValue{
		ElemType: array.ElemType,
		Elems:    append([]Value(nil), array.Elems...),
	}
	for i, value := range values {
		if value == nil {
			return TypedArrayValue{}, fmt.Errorf("%w at index %d", ErrNilArrayElem, i)
		}
		if value.Type() != out.ElemType {
			return TypedArrayValue{}, fmt.Errorf(
				"typed array element type mismatch at index %d: want %s, got %s",
				i, out.ElemType, value.Type(),
			)
		}
		out.Elems = append(out.Elems, value)
	}
	return out, nil
}
