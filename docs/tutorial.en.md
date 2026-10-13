Language: [Русский](tutorial.md) · [English](tutorial.en.md)

# Tutorials

Step-by-step client usage against a running Ignicula server. Interactive shell: [CLI](cli.en.md).

Replace `addr`, `Login`, and `Hash` with values your server expects.

## 1. Dial and wait until ready

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

	cli, err := client.Dial("127.0.0.1:16000", cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for {
		if err := cli.Ping(ctx); err == nil {
			log.Println("ready")
			return
		}
		select {
		case <-ctx.Done():
			log.Fatal("activate:", ctx.Err())
		case <-cli.Done():
			log.Fatal("disconnected:", cli.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
}
```

Until Handshake finishes, `Ping` fails (activation still in progress). Retry until success, deadline, or `Done`. A nil logger uses a stderr text logger at info level.

## 2. Ping

```go
ctx := context.Background()
if err := cli.Ping(ctx); err != nil {
	log.Fatal(err)
}
```

## 3. Write and read scalars

`Write` builds a value with `values.New`. Omit the type to store **bytes**.

```go
import "github.com/dejitarudemon/ignicula-go/values"

ctx := context.Background()
key := []byte("user:1:name")

if err := cli.Write(ctx, key, []byte("alice"), values.String); err != nil {
	log.Fatal(err)
}

val, err := cli.Read(ctx, key)
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

Numeric and JSON examples:

```go
_ = cli.Write(ctx, []byte("n"), []byte("-42"), values.Int64)
_ = cli.Write(ctx, []byte("u"), []byte("42"), values.Uint64)
_ = cli.Write(ctx, []byte("f"), []byte("3.14"), values.Float64)
_ = cli.Write(ctx, []byte("j"), []byte(`{"ok":true}`), values.JSON)
```

## 4. Write arrays with WriteValue

```go
typed, err := values.NewTypedArray(
	values.String,
	values.StringValue("a"),
	values.StringValue("b"),
)
if err != nil {
	log.Fatal(err)
}
if err := cli.WriteValue(ctx, []byte("tags"), typed); err != nil {
	log.Fatal(err)
}

mixed, err := values.NewUntypedArray(
	values.StringValue("hello"),
	values.Int64Value(-7),
)
if err != nil {
	log.Fatal(err)
}
if err := cli.WriteValue(ctx, []byte("mix"), mixed); err != nil {
	log.Fatal(err)
}
```

`Read` returns the same client types (including arrays) after adapter conversion.

## 5. Delete

```go
if err := cli.Delete(ctx, []byte("user:1:name")); err != nil {
	log.Fatal(err)
}
```

## 6. Watch for disconnect

```go
go func() {
	<-cli.Done()
	log.Println("connection ended:", cli.Err())
}()

// ... use cli ...

select {
case <-cli.Done():
	if err := cli.Err(); err != nil {
		log.Println("disconnected:", err)
	}
default:
}
```

After `Done`, further ops return the same error from `Err()` (typically wrapping `transport.ErrDisconnected`, or `transport.ErrClosed` after a clean `Close`).

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
cli, err := client.Dial(addr, cfg, log)
```

Pass any `logger.Logger` implementation, or `nil` for the default.
