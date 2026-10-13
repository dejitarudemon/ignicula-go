// Package adapters_v1 converts between client [values.Value]s and protocol v1
// wire values ([github.com/dejitarudemon/ignicula-wire/v1/value.V]).
//
// Only Bytes and String are supported so far; other types return
// [ConversionError].
package adapters_v1
