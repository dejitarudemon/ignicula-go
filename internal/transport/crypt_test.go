package transport

import (
	"bytes"
	"testing"
)

func TestHash_Deterministic(t *testing.T) {
	a := hash("secret")
	b := hash("secret")
	if a != b {
		t.Fatalf("same password produced different hashes")
	}
}

func TestHash_DifferentPasswords(t *testing.T) {
	a := hash("secret")
	b := hash("other")
	if a == b {
		t.Fatalf("different passwords produced the same hash")
	}
}

func TestHash_EmptyPassword(t *testing.T) {
	got := hash("")
	if bytes.Equal(got[:], make([]byte, 32)) {
		t.Fatalf("empty password must not produce a zero hash")
	}
}
