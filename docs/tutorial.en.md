Language: [Русский](tutorial.md) · [English](tutorial.en.md)

# Tutorials

Using [`client.Ignicula`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/client#Ignicula). Replace `addr`, `Login`, and `Hash` for your server.

## Guarantees

What the client promises:

- **After a successful `Dial`**, the connection is activated (Hello + Handshake); you may call `Ping` / `Read` / `Write` / `Delete` immediately.
- **Each op waits for a server answer** (or an error / attempt timeout). A successful return from `Write` / `Delete` means the server sent a successful Answer.
- **Transient errors** (`RequestsConflict`, `InternalError`) are retried automatically, up to `RuntimeV1.MaxAttempts` (default 3) with backoff.
- **A dropped connection** fails in-flight ops and makes later calls fail too. Watching `Done` is optional.
- **`Close` is idempotent** and is not logged as a fatal reader error (unlike peer disconnect).

What it does not promise:

- **Exactly-once on the wire under retries** — a request may be resent; server/key idempotency is your concern if it matters.
- **Global ordering across goroutines** — concurrent calls on one `Ignicula` are fine, but delivery/apply order between them is not fixed.
- **Durability without an answer** — if `ctx` is canceled or the socket dies before Answer, the server outcome is unknown; re-read or design an idempotent write.
- **Automatic reconnect** — after `Done`, dial again.

Default timeouts: `ConnectTimeout` 30s (whole `Dial`), CRUD 30s per attempt, Ping 5s, socket read/write 100s.

## 1. Dial

`Dial` opens TCP, runs Hello, activates v1, and returns a ready client. The wait is bounded by `ConnectTimeout` in the config (default 30s).

```go
cfg := configs.DefaultConnectionConfig()
cfg.Auth = configs.AuthConfig{Login: "user", Hash: [32]byte{ /* … */ }}

ignicula, err := client.Dial("127.0.0.1:16000", cfg, nil)
if err != nil {
	log.Fatal(err)
}
defer ignicula.Close()
```

## 2. Ping

```go
if err := ignicula.Ping(ctx); err != nil {
	log.Fatal(err)
}
```

## 3. Write / Read

```go
_ = ignicula.Write(ctx, []byte("name"), values.StringValue("alice"))
_ = ignicula.Write(ctx, []byte("n"), values.Int64Value(-42))
_ = ignicula.Write(ctx, []byte("j"), values.JSONValue(`{"ok":true}`))

val, err := ignicula.Read(ctx, []byte("name"))
if err != nil {
	log.Fatal(err)
}
fmt.Println(val)
```

## 4. Arrays

```go
tags, _ := values.NewTypedArray(values.String, values.StringValue("a"), values.StringValue("b"))
_ = ignicula.Write(ctx, []byte("tags"), tags)

mix, _ := values.NewUntypedArray(values.StringValue("hello"), values.Int64Value(-7))
_ = ignicula.Write(ctx, []byte("mix"), mix)
```

## 5. Delete

```go
_ = ignicula.Delete(ctx, []byte("name"))
```

## 6. Done and Err (optional)

Ops already fail after disconnect. Use `Done` only if you want to notice the end without another request:

```go
go func() {
	<-ignicula.Done()
	log.Println(ignicula.Err())
}()
```

## 7. Config and logger

```go
cfg.ConnectTimeout = 5 * time.Second
cfg.RuntimeV1.MaxAttempts = 5

ignicula, err := client.Dial(addr, cfg, implemented.NewText(slog.LevelDebug))
```
