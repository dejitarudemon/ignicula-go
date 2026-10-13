package values

import (
	"errors"
	"testing"
)

func TestConvert_SameType(t *testing.T) {
	v := StringValue("hi")
	got, err := Convert(v, String)
	if err != nil || got != v {
		t.Fatalf("Convert = (%v, %v)", got, err)
	}
}

func TestConvert_StringBytes(t *testing.T) {
	got, err := Convert(StringValue("hi"), Bytes)
	if err != nil {
		t.Fatalf("Convert = %v", err)
	}
	bv, ok := got.(BytesValue)
	if !ok || string(bv) != "hi" {
		t.Fatalf("got %#v", got)
	}

	got, err = Convert(BytesValue("hi"), String)
	if err != nil {
		t.Fatalf("Convert = %v", err)
	}
	sv, ok := got.(StringValue)
	if !ok || string(sv) != "hi" {
		t.Fatalf("got %#v", got)
	}
}

func TestConvert_Unsupported(t *testing.T) {
	_, err := Convert(StringValue("x"), Int64)
	var u ErrUnsupportedConversion
	if !errors.As(err, &u) {
		t.Fatalf("err = %v", err)
	}
}
