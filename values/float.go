package values

// Float64Value is a 64-bit floating-point number with type code [Float64].
type Float64Value float64

// Type returns [Float64].
func (f Float64Value) Type() Type {
	return Float64
}
