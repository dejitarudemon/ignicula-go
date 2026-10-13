package adapters_v1

import (
	"github.com/dejitarudemon/ignicula-go/values"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	protovalue "github.com/dejitarudemon/ignicula-wire/v1/value"
	protovalues "github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

const (
	opFrom = "from"
	opTo   = "to"
)

// Adapter converts between client [values.Value] and protocol v1 [protovalue.V].
// All scalar types and both array kinds are supported (arrays recursively).
type Adapter struct{}

// New returns an [Adapter].
func New() Adapter {
	return Adapter{}
}

// From converts a protocol v1 value into a client [values.Value].
func (a Adapter) From(v protovalue.V) (values.Value, error) {
	if v == nil {
		return nil, newConversionError(opFrom, "nil value")
	}

	switch data := v.(type) {
	case protovalues.Bytes:
		return values.BytesValue(data), nil
	case protovalues.String:
		return values.StringValue(data), nil
	case protovalues.Int:
		return values.Int64Value(data), nil
	case protovalues.Uint:
		return values.Uint64Value(data), nil
	case protovalues.Float:
		return values.Float64Value(data), nil
	case protovalues.JSON:
		return values.JSONValue(data), nil
	case protovalues.TypedArray:
		elems := make([]values.Value, len(data.Elems))
		for i, elem := range data.Elems {
			cv, err := a.From(elem)
			if err != nil {
				return nil, newConversionErrorf(opFrom, "typed array elem %d: %v", i, err)
			}
			elems[i] = cv
		}
		return values.TypedArrayValue{
			ElemType: fromWireType(data.ElemType),
			Elems:    elems,
		}, nil
	case protovalues.UntypedArray:
		elems := make([]values.Value, len(data))
		for i, elem := range data {
			cv, err := a.From(elem)
			if err != nil {
				return nil, newConversionErrorf(opFrom, "untyped array elem %d: %v", i, err)
			}
			elems[i] = cv
		}
		return values.UntypedArrayValue{Elems: elems}, nil
	default:
		return nil, newConversionErrorf(opFrom, "unsupported wire type %T", v)
	}
}

// To converts a client [values.Value] into a protocol v1 value.
func (a Adapter) To(v values.Value) (protovalue.V, error) {
	if v == nil {
		return nil, newConversionError(opTo, "nil value")
	}

	switch data := v.(type) {
	case values.BytesValue:
		return protovalues.Bytes(data), nil
	case values.StringValue:
		return protovalues.String(data), nil
	case values.Int64Value:
		return protovalues.Int(data), nil
	case values.Uint64Value:
		return protovalues.Uint(data), nil
	case values.Float64Value:
		return protovalues.Float(data), nil
	case values.JSONValue:
		return protovalues.JSON(data), nil
	case values.TypedArrayValue:
		wireType, err := toWireType(data.ElemType)
		if err != nil {
			return nil, newConversionErrorf(opTo, "typed array elem type: %v", err)
		}
		elems := make([]protovalue.V, len(data.Elems))
		for i, elem := range data.Elems {
			wv, err := a.To(elem)
			if err != nil {
				return nil, newConversionErrorf(opTo, "typed array elem %d: %v", i, err)
			}
			elems[i] = wv
		}
		return protovalues.TypedArray{ElemType: wireType, Elems: elems}, nil
	case values.UntypedArrayValue:
		elems := make(protovalues.UntypedArray, len(data.Elems))
		for i, elem := range data.Elems {
			wv, err := a.To(elem)
			if err != nil {
				return nil, newConversionErrorf(opTo, "untyped array elem %d: %v", i, err)
			}
			elems[i] = wv
		}
		return elems, nil
	default:
		return nil, newConversionErrorf(opTo, "unsupported value type %T (%v)", v, v.Type())
	}
}

func fromWireType(t fields.Type) values.Type {
	return values.Type(t)
}

func toWireType(t values.Type) (fields.Type, error) {
	if !t.IsValid() {
		return 0, newConversionErrorf(opTo, "invalid type %s", t)
	}
	return fields.Type(t), nil
}
