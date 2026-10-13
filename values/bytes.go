package values

// BytesValue is an opaque byte sequence with type code [Bytes].
type BytesValue []byte

// Type returns [Bytes].
func (b BytesValue) Type() Type {
	return Bytes
}
