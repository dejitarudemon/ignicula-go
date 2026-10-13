package adapters_v1_test

import (
	"errors"
	"testing"

	adapters_v1 "github.com/dejitarudemon/ignicula-go/internal/adapters/v1"
	"github.com/dejitarudemon/ignicula-go/values"
	protovalues "github.com/dejitarudemon/ignicula-wire/v1/value/values"
)

func TestFromTo_Bytes(t *testing.T) {
	a := adapters_v1.New()

	got, err := a.From(protovalues.Bytes([]byte{1, 2, 3}))
	if err != nil {
		t.Fatalf("From = %v", err)
	}
	bv, ok := got.(values.BytesValue)
	if !ok || string(bv) != "\x01\x02\x03" {
		t.Fatalf("From = %#v", got)
	}

	wire, err := a.To(bv)
	if err != nil {
		t.Fatalf("To = %v", err)
	}
	wb, ok := wire.(protovalues.Bytes)
	if !ok || string(wb) != "\x01\x02\x03" {
		t.Fatalf("To = %#v", wire)
	}
}

func TestFromTo_String(t *testing.T) {
	a := adapters_v1.New()

	got, err := a.From(protovalues.String("hello"))
	if err != nil {
		t.Fatalf("From = %v", err)
	}
	sv, ok := got.(values.StringValue)
	if !ok || string(sv) != "hello" {
		t.Fatalf("From = %#v", got)
	}

	wire, err := a.To(sv)
	if err != nil {
		t.Fatalf("To = %v", err)
	}
	ws, ok := wire.(protovalues.String)
	if !ok || string(ws) != "hello" {
		t.Fatalf("To = %#v", wire)
	}
}

func TestFrom_Unsupported(t *testing.T) {
	_, err := adapters_v1.New().From(protovalues.Int(1))
	var conv adapters_v1.ConversionError
	if !errors.As(err, &conv) || conv.Op() != "from" {
		t.Fatalf("From = %v, want ConversionError(from)", err)
	}
}

func TestTo_Unsupported(t *testing.T) {
	_, err := adapters_v1.New().To(unsupportedValue{})
	var conv adapters_v1.ConversionError
	if !errors.As(err, &conv) || conv.Op() != "to" {
		t.Fatalf("To = %v, want ConversionError(to)", err)
	}
}

func TestNil(t *testing.T) {
	a := adapters_v1.New()
	if _, err := a.From(nil); err == nil {
		t.Fatal("From(nil) = nil")
	}
	if _, err := a.To(nil); err == nil {
		t.Fatal("To(nil) = nil")
	}
}

type unsupportedValue struct{}

func (unsupportedValue) Type() values.Type { return values.Int64 }
