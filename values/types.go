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
	// type here
	}

	return fmt.Sprintf("Unknown (%d)", t)
}

func (t Type) IsValid() bool {
	return t <= JSON
}
