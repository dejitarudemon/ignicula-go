package values

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// New builds a [Value] of typ from raw data.
// Numeric types parse decimal text; [JSON] requires valid JSON bytes.
func New(typ Type, data []byte) (Value, error) {
	switch typ {
	case Bytes:
		return BytesValue(append([]byte(nil), data...)), nil
	case String:
		return StringValue(data), nil
	case Int64:
		n, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid int64 %q: %w", data, err)
		}
		return Int64Value(n), nil
	case Uint64:
		n, err := strconv.ParseUint(string(data), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid uint64 %q: %w", data, err)
		}
		return Uint64Value(n), nil
	case Float64:
		n, err := strconv.ParseFloat(string(data), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float64 %q: %w", data, err)
		}
		return Float64Value(n), nil
	case JSON:
		if !json.Valid(data) {
			return nil, fmt.Errorf("invalid json %q", data)
		}
		return JSONValue(append([]byte(nil), data...)), nil
	default:
		return nil, fmt.Errorf("unsupported value type %s", typ)
	}
}
