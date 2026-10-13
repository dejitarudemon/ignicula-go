package values

// JSONValue is an opaque byte sequence with type code [JSON].
type JSONValue []byte

// Type returns [JSON].
func (j JSONValue) Type() Type {
	return JSON
}
