package values

import "testing"

func TestNew_BytesString(t *testing.T) {
	bv, err := New(Bytes, []byte("hi"))
	if err != nil {
		t.Fatalf("New(Bytes) = %v", err)
	}
	if got, ok := bv.(BytesValue); !ok || string(got) != "hi" {
		t.Fatalf("Bytes = %#v", bv)
	}

	sv, err := New(String, []byte("hi"))
	if err != nil {
		t.Fatalf("New(String) = %v", err)
	}
	if got, ok := sv.(StringValue); !ok || string(got) != "hi" {
		t.Fatalf("String = %#v", sv)
	}
}

func TestNew_Int64(t *testing.T) {
	iv, err := New(Int64, []byte("-42"))
	if err != nil {
		t.Fatalf("New(Int64) = %v", err)
	}
	if got, ok := iv.(Int64Value); !ok || got != -42 {
		t.Fatalf("Int64 = %#v", iv)
	}

	if _, err := New(Int64, []byte("nope")); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestNew_Uint64(t *testing.T) {
	uv, err := New(Uint64, []byte("42"))
	if err != nil {
		t.Fatalf("New(Uint64) = %v", err)
	}
	if got, ok := uv.(Uint64Value); !ok || got != 42 {
		t.Fatalf("Uint64 = %#v", uv)
	}

	if _, err := New(Uint64, []byte("-1")); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestNew_Float64(t *testing.T) {
	fv, err := New(Float64, []byte("3.5"))
	if err != nil {
		t.Fatalf("New(Float64) = %v", err)
	}
	if got, ok := fv.(Float64Value); !ok || got != 3.5 {
		t.Fatalf("Float64 = %#v", fv)
	}

	if _, err := New(Float64, []byte("nope")); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestNew_JSON(t *testing.T) {
	jv, err := New(JSON, []byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("New(JSON) = %v", err)
	}
	if got, ok := jv.(JSONValue); !ok || string(got) != `{"a":1}` {
		t.Fatalf("JSON = %#v", jv)
	}

	if _, err := New(JSON, []byte("{")); err == nil {
		t.Fatal("expected invalid json")
	}
}

func TestNew_Unsupported(t *testing.T) {
	if _, err := New(TypedArray, nil); err == nil {
		t.Fatal("expected error")
	}
}
