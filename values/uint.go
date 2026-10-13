package values

// Uint64Value is a unsigned 64-bit integer with type code [Uint64].
type Uint64Value uint64

// Type returns [Uint64].
func (u Uint64Value) Type() Type {
	return Uint64
}
