package transport

import (
	"context"
	"testing"
)

func TestRequestStubs(t *testing.T) {
	c := &Connection{}
	ctx := context.Background()

	if v, err := c.Read(ctx, []byte("k")); v != nil || err != nil {
		t.Fatalf("Read = (%v, %v), want (nil, nil)", v, err)
	}
	if err := c.Write(ctx, []byte("k"), nil); err != nil {
		t.Fatalf("Write = %v", err)
	}
	if err := c.Delete(ctx, []byte("k")); err != nil {
		t.Fatalf("Delete = %v", err)
	}
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("Ping = %v", err)
	}
}
