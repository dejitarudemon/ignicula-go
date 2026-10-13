package adapters_v1_test

import (
	"errors"
	"testing"

	adapters_v1 "github.com/dejitarudemon/ignicula-go/internal/adapters/v1"
	"github.com/dejitarudemon/ignicula-go/values"
	"github.com/dejitarudemon/ignicula-wire/v1/buffer"
	"github.com/dejitarudemon/ignicula-wire/v1/fields"
	protovalue "github.com/dejitarudemon/ignicula-wire/v1/value"
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

func TestFromTo_Int64(t *testing.T) {
	a := adapters_v1.New()

	got, err := a.From(protovalues.Int(-7))
	if err != nil {
		t.Fatalf("From = %v", err)
	}
	iv, ok := got.(values.Int64Value)
	if !ok || iv != -7 {
		t.Fatalf("From = %#v", got)
	}

	wire, err := a.To(iv)
	if err != nil {
		t.Fatalf("To = %v", err)
	}
	wi, ok := wire.(protovalues.Int)
	if !ok || wi != -7 {
		t.Fatalf("To = %#v", wire)
	}
}

func TestFromTo_Uint64(t *testing.T) {
	a := adapters_v1.New()

	got, err := a.From(protovalues.Uint(9))
	if err != nil {
		t.Fatalf("From = %v", err)
	}
	uv, ok := got.(values.Uint64Value)
	if !ok || uv != 9 {
		t.Fatalf("From = %#v", got)
	}

	wire, err := a.To(uv)
	if err != nil {
		t.Fatalf("To = %v", err)
	}
	wu, ok := wire.(protovalues.Uint)
	if !ok || wu != 9 {
		t.Fatalf("To = %#v", wire)
	}
}

func TestFromTo_Float64(t *testing.T) {
	a := adapters_v1.New()

	got, err := a.From(protovalues.Float(1.25))
	if err != nil {
		t.Fatalf("From = %v", err)
	}
	fv, ok := got.(values.Float64Value)
	if !ok || fv != 1.25 {
		t.Fatalf("From = %#v", got)
	}

	wire, err := a.To(fv)
	if err != nil {
		t.Fatalf("To = %v", err)
	}
	wf, ok := wire.(protovalues.Float)
	if !ok || wf != 1.25 {
		t.Fatalf("To = %#v", wire)
	}
}

func TestFromTo_JSON(t *testing.T) {
	a := adapters_v1.New()

	got, err := a.From(protovalues.JSON([]byte(`{"k":1}`)))
	if err != nil {
		t.Fatalf("From = %v", err)
	}
	jv, ok := got.(values.JSONValue)
	if !ok || string(jv) != `{"k":1}` {
		t.Fatalf("From = %#v", got)
	}

	wire, err := a.To(jv)
	if err != nil {
		t.Fatalf("To = %v", err)
	}
	wj, ok := wire.(protovalues.JSON)
	if !ok || string(wj) != `{"k":1}` {
		t.Fatalf("To = %#v", wire)
	}
}

func TestFromTo_TypedArray(t *testing.T) {
	a := adapters_v1.New()
	wire := protovalues.TypedArray{
		ElemType: fields.String,
		Elems:    []protovalue.V{protovalues.String("a"), protovalues.String("b")},
	}

	got, err := a.From(wire)
	if err != nil {
		t.Fatalf("From = %v", err)
	}
	ta, ok := got.(values.TypedArrayValue)
	if !ok || ta.ElemType != values.String || len(ta.Elems) != 2 {
		t.Fatalf("From = %#v", got)
	}

	back, err := a.To(ta)
	if err != nil {
		t.Fatalf("To = %v", err)
	}
	wta, ok := back.(protovalues.TypedArray)
	if !ok || wta.ElemType != fields.String || len(wta.Elems) != 2 {
		t.Fatalf("To = %#v", back)
	}
}

func TestFromTo_UntypedArray(t *testing.T) {
	a := adapters_v1.New()
	wire := protovalues.UntypedArray{protovalues.String("a"), protovalues.Int(1)}

	got, err := a.From(wire)
	if err != nil {
		t.Fatalf("From = %v", err)
	}
	ua, ok := got.(values.UntypedArrayValue)
	if !ok || len(ua.Elems) != 2 {
		t.Fatalf("From = %#v", got)
	}

	back, err := a.To(ua)
	if err != nil {
		t.Fatalf("To = %v", err)
	}
	wua, ok := back.(protovalues.UntypedArray)
	if !ok || len(wua) != 2 {
		t.Fatalf("To = %#v", back)
	}
}

func TestFrom_Unsupported(t *testing.T) {
	_, err := adapters_v1.New().From(fakeWire{})
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

func (unsupportedValue) Type() values.Type { return 255 }

type fakeWire struct{}

func (fakeWire) Size() int                          { return 0 }
func (fakeWire) IsValid() error                     { return nil }
func (fakeWire) Encode(buf buffer.Appender)         {}
func (fakeWire) Type() fields.Type                  { return 255 }
