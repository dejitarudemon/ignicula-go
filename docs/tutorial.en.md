Language: [Русский](tutorial.md) · [English](tutorial.en.md)

# Tutorials

Step-by-step use of [`client.Ignicula`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/client#Ignicula) against a running server.

Replace `addr`, `Login`, and `Hash` with values your server expects.

## 1. Dial and WaitReady

`Dial` returns `*Ignicula` and starts Hello + Handshake in the background. Call `WaitReady` before other ops — it blocks until activation finishes (or until `ctx` / disconnect).

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/dejitarudemon/ignicula-go/client"
	"github.com/dejitarudemon/ignicula-go/configs"
)

func main() {
	cfg := configs.DefaultConnectionConfig()
	cfg.Auth = configs.AuthConfig{
		Login: "user",
		Hash:  [32]byte{ /* 32 bytes */ },
	}

	ignicula, err := client.Dial("127.0.0.1:16000", cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ignicula.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := ignicula.WaitReady(ctx); err != nil {
		log.Fatal("activate:", err)
	}

	// continue with Ping / Read / Write / Delete on this ignicula
}
```

A nil logger uses a stderr text logger at info level.

## 2. Ping

```go
ctx := context.Background()
if err := ignicula.Ping(ctx); err != nil {
	log.Fatal(err)
}
```

## 3. Write and read scalars

`Ignicula.Write` takes any `values.Value`.

```go
import "github.com/dejitarudemon/ignicula-go/values"

ctx := context.Background()
key := []byte("user:1:name")

if err := ignicula.Write(ctx, key, values.StringValue("alice")); err != nil {
	log.Fatal(err)
}

val, err := ignicula.Read(ctx, key)
if err != nil {
	log.Fatal(err)
}
switch v := val.(type) {
case values.StringValue:
	log.Printf("name=%q", string(v))
default:
	log.Printf("got %T", val)
}
```

Numeric and JSON examples (constructors or `values.New`):

```go
_ = ignicula.Write(ctx, []byte("n"), values.Int64Value(-42))
_ = ignicula.Write(ctx, []byte("u"), values.Uint64Value(42))
_ = ignicula.Write(ctx, []byte("f"), values.Float64Value(3.14))
j, _ := values.New(values.JSON, []byte(`{"ok":true}`))
_ = ignicula.Write(ctx, []byte("j"), j)
```

## 4. Arrays

Same `Write` — pass a typed or untyped array:

```go
typed, err := values.NewTypedArray(
	values.String,
	values.StringValue("a"),
	values.StringValue("b"),
)
if err != nil {
	log.Fatal(err)
}
if err := ignicula.Write(ctx, []byte("tags"), typed); err != nil {
	log.Fatal(err)
}

mixed, err := values.NewUntypedArray(
	values.StringValue("hello"),
	values.Int64Value(-7),
)
if err != nil {
	log.Fatal(err)
}
if err := ignicula.Write(ctx, []byte("mix"), mixed); err != nil {
	log.Fatal(err)
}

val, err := ignicula.Read(ctx, []byte("tags"))
if err != nil {
	log.Fatal(err)
}
if arr, ok := val.(values.TypedArrayValue); ok {
	log.Printf("elemType=%s len=%d", arr.ElemType, len(arr.Elems))
}
```

## 5. Delete

```go
if err := ignicula.Delete(ctx, []byte("user:1:name")); err != nil {
	log.Fatal(err)
}
```

## 6. Watch for disconnect

`Ignicula` exposes `Done` / `Err` for connection liveness:

```go
go func() {
	<-ignicula.Done()
	log.Println("connection ended:", ignicula.Err())
}()

// ... use ignicula ...

select {
case <-ignicula.Done():
	if err := ignicula.Err(); err != nil {
		log.Println("disconnected:", err)
	}
default:
}
```

After `Done`, further `Ping` / `Read` / `Write` / `Delete` calls return the same error as `Err()` (typically wrapping `ErrDisconnected`, or `ErrClosed` after a clean `Close`).

## 7. Custom config and logger

```go
import (
	"log/slog"
	"time"

	"github.com/dejitarudemon/ignicula-go/logger/implemented"
)

cfg := configs.DefaultConnectionConfig()
cfg.ConnectTimeout = 5 * time.Second
cfg.RuntimeV1.ReadTimeout = 10 * time.Second
cfg.RuntimeV1.MaxAttempts = 5
cfg.Auth = configs.AuthConfig{Login: "user", Hash: hash}

log := implemented.NewText(slog.LevelDebug)
ignicula, err := client.Dial(addr, cfg, log)
```

Pass any `logger.Logger` implementation, or `nil` for the default.
