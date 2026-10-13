package values

// StringValue is a UTF-8 string with type code [String].
type StringValue string

// Type returns [String].
func (s StringValue) Type() Type {
	return String
}
