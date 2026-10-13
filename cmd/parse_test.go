package main

import (
	"testing"

	"github.com/dejitarudemon/ignicula-go/values"
)

func TestParseWriteArgs(t *testing.T) {
	raw, name, typ := parseWriteArgs("hello string")
	if raw != "hello" || name != "string" || typ != values.String {
		t.Fatalf("got %q %q %v", raw, name, typ)
	}

	raw, name, typ = parseWriteArgs("typed string")
	if raw != "typed" || name != "string" || typ != values.String {
		t.Fatalf("got %q %q %v", raw, name, typ)
	}
}

func TestParseTypedArrayArgs(t *testing.T) {
	v, name, err := parseTypedArrayArgs("int64 1 2 3")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	ta, ok := v.(values.TypedArrayValue)
	if !ok || ta.ElemType != values.Int64 || len(ta.Elems) != 3 {
		t.Fatalf("got %#v name=%q", v, name)
	}
	if name != "typed[Int64]" {
		t.Fatalf("name = %q", name)
	}

	v, _, err = parseTypedArrayArgs("string")
	if err != nil {
		t.Fatalf("empty = %v", err)
	}
	ta, ok = v.(values.TypedArrayValue)
	if !ok || len(ta.Elems) != 0 {
		t.Fatalf("got %#v", v)
	}
}

func TestParseUntypedArrayArgs(t *testing.T) {
	v, name, err := parseUntypedArrayArgs(`string:hello int64:-7 json:{"a":1}`)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	ua, ok := v.(values.UntypedArrayValue)
	if !ok || len(ua.Elems) != 3 || name != "untyped" {
		t.Fatalf("got %#v name=%q", v, name)
	}

	v, _, err = parseUntypedArrayArgs("")
	if err != nil {
		t.Fatalf("empty = %v", err)
	}
	ua, ok = v.(values.UntypedArrayValue)
	if !ok || len(ua.Elems) != 0 {
		t.Fatalf("got %#v", v)
	}

	if _, _, err := parseUntypedArrayArgs("hello"); err == nil {
		t.Fatal("expected error")
	}
}
