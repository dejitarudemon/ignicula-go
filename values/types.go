package values

import "fmt"

type Type uint8

const (
	Bytes Type = iota
	TypedArray
	UntypedArray
	Int64
	Uint64
	Float64
	String
	JSON
)

func (t Type) String() string {
	switch t {
	case Bytes:
		return "Bytes"
	case TypedArray:
		return "TypedArray"
	case UntypedArray:
		return "UntypedArray"
	case Int64:
		return "Int64"
	case Uint64:
		return "Uint64"
	case Float64:
		return "Float64"
	case String:
		return "String"
	case JSON:
		return "JSON"
	}

	return fmt.Sprintf("Unknown (%d)", t)
}

func (t Type) IsValid() bool {
	return t <= JSON
}
