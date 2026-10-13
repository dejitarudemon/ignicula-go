package adapters_v1

import (
	"github.com/dejitarudemon/ignicula-go/values"
	protovalue "github.com/dejitarudemon/ignicula-wire/v1/value"
	protovalues "github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

const (
	opFrom = "from"
	opTo   = "to"
)

// Adapter converts between client [values.Value] and protocol v1 [protovalue.V].
// Only [values.Bytes] and [values.String] are supported for now.
type Adapter struct{}

// New returns an [Adapter].
func New() Adapter {
	return Adapter{}
}

// From converts a protocol v1 value into a client [values.Value].
func (Adapter) From(v protovalue.V) (values.Value, error) {
	if v == nil {
		return nil, newConversionError(opFrom, "nil value")
	}

	switch data := v.(type) {
	case protovalues.Bytes:
		return values.BytesValue(data), nil
	case protovalues.String:
		return values.StringValue(data), nil
	default:
		return nil, newConversionErrorf(opFrom, "unsupported wire type %T", v)
	}
}

// To converts a client [values.Value] into a protocol v1 value.
func (Adapter) To(v values.Value) (protovalue.V, error) {
	if v == nil {
		return nil, newConversionError(opTo, "nil value")
	}

	switch data := v.(type) {
	case values.BytesValue:
		return protovalues.Bytes(data), nil
	case values.StringValue:
		return protovalues.String(data), nil
	default:
		return nil, newConversionErrorf(opTo, "unsupported value type %T (%v)", v, v.Type())
	}
}
