package values

import (
	"errors"
	"testing"
)

func TestNewTypedArray(t *testing.T) {
	got, err := NewTypedArray(String, StringValue("a"), StringValue("b"))
	if err != nil {
		t.Fatalf("NewTypedArray = %v", err)
	}
	if got.Type() != TypedArray || got.ElemType != String || len(got.Elems) != 2 {
		t.Fatalf("got %#v", got)
	}

	if _, err := NewTypedArray(String, StringValue("a"), BytesValue("b")); err == nil {
		t.Fatal("expected type mismatch")
	}
	if _, err := NewTypedArray(Int64, nil); !errors.Is(err, ErrNilArrayElem) {
		t.Fatalf("nil = %v", err)
	}
}

func TestTypedArrayAppend(t *testing.T) {
	base, err := NewTypedArray(Int64, Int64Value(1))
	if err != nil {
		t.Fatal(err)
	}
	got, err := base.Append(Int64Value(2))
	if err != nil {
		t.Fatalf("Append = %v", err)
	}
	if len(base.Elems) != 1 || len(got.Elems) != 2 {
		t.Fatalf("base=%d got=%d", len(base.Elems), len(got.Elems))
	}
}

func TestNewUntypedArray(t *testing.T) {
	got, err := NewUntypedArray(StringValue("a"), Int64Value(1))
	if err != nil {
		t.Fatalf("NewUntypedArray = %v", err)
	}
	if got.Type() != UntypedArray || len(got.Elems) != 2 {
		t.Fatalf("got %#v", got)
	}
	if _, err := NewUntypedArray(nil); !errors.Is(err, ErrNilArrayElem) {
		t.Fatalf("nil = %v", err)
	}
}
