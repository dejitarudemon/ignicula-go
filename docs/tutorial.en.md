Language: [Русский](tutorial.md) · [English](tutorial.en.md)

# Tutorials

Using [`client.Ignicula`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/client#Ignicula). Replace `addr`, `Login`, and `Hash` for your server.

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
