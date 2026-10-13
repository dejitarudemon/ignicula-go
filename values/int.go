package values

// Int64Value is a signed 64-bit integer with type code [Int64].
type Int64Value int64

// Type returns [Int64].
func (i Int64Value) Type() Type {
	return Int64
}
