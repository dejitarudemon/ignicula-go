package polices

import "fmt"

type Policy int8

const (
	UseOnlyOneVersion Policy = iota
	UseMinVersion
	UseMaxVersion
)

func (p Policy) String() string {
	switch p {
	case UseOnlyOneVersion:
		return "UseOnlyOneVersion"
	case UseMaxVersion:
		return "UseMaxVersion"
	case UseMinVersion:
		return "UseMinVersion"
	}

	return fmt.Sprintf("Unknown (%d)", p)
}

func (p Policy) IsValid() bool {
	return p <= UseMinVersion
}
