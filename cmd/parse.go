package main

import (
	"fmt"
	"strings"

	"github.com/dejitarudemon/ignicula-go/values"
)

// parseTypeName maps a CLI type token to a [values.Type].
func parseTypeName(s string) (values.Type, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "bytes", "b":
		return values.Bytes, true
	case "string", "s", "str":
		return values.String, true
	case "int64", "i", "int":
		return values.Int64, true
	case "uint64", "u", "uint":
		return values.Uint64, true
	case "float64", "f", "float":
		return values.Float64, true
	case "json", "j":
		return values.JSON, true
	default:
		return 0, false
	}
}

// parseWriteArgs splits "<value>" or "<value> <type>" where type is optional
// and defaults to bytes. If the last token is a known scalar type name, it is
// treated as the type and the rest as the value.
func parseWriteArgs(rest string) (value, typName string, typ values.Type) {
	typ = values.Bytes
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", "", typ
	}
	lastSpace := strings.LastIndex(rest, " ")
	if lastSpace < 0 {
		return rest, "", typ
	}
	maybeType := strings.TrimSpace(rest[lastSpace+1:])
	if t, ok := parseTypeName(maybeType); ok {
		return strings.TrimSpace(rest[:lastSpace]), strings.ToLower(maybeType), t
	}
	return rest, "", typ
}

// parseTypedArrayArgs parses "<elemType> <elem>..." for writetyped.
func parseTypedArrayArgs(rest string) (values.Value, string, error) {
	fields := strings.Fields(strings.TrimSpace(rest))
	if len(fields) == 0 {
		return nil, "", fmt.Errorf("usage: writetyped <key> <elemType> <elem>...")
	}
	elemType, ok := parseTypeName(fields[0])
	if !ok {
		return nil, "", fmt.Errorf("unknown element type %q", fields[0])
	}

	elems := make([]values.Value, 0, len(fields)-1)
	for i, raw := range fields[1:] {
		v, err := values.New(elemType, []byte(raw))
		if err != nil {
			return nil, "", fmt.Errorf("elem %d: %w", i, err)
		}
		elems = append(elems, v)
	}

	arr, err := values.NewTypedArray(elemType, elems...)
	if err != nil {
		return nil, "", err
	}
	return arr, fmt.Sprintf("typed[%s]", elemType), nil
}

// parseUntypedArrayArgs parses "<type:value>..." for writeuntyped.
func parseUntypedArrayArgs(rest string) (values.Value, string, error) {
	fields := strings.Fields(strings.TrimSpace(rest))
	elems := make([]values.Value, 0, len(fields))
	for i, part := range fields {
		typName, raw, ok := strings.Cut(part, ":")
		if !ok || typName == "" || raw == "" {
			return nil, "", fmt.Errorf("elem %d: want <type>:<value>, got %q", i, part)
		}
		elemType, ok := parseTypeName(typName)
		if !ok {
			return nil, "", fmt.Errorf("elem %d: unknown type %q", i, typName)
		}
		v, err := values.New(elemType, []byte(raw))
		if err != nil {
			return nil, "", fmt.Errorf("elem %d: %w", i, err)
		}
		elems = append(elems, v)
	}

	arr, err := values.NewUntypedArray(elems...)
	if err != nil {
		return nil, "", err
	}
	return arr, "untyped", nil
}
