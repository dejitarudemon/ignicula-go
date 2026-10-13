Language: [Русский](tutorial.md) · [English](tutorial.en.md)

# Туториалы

Работа с [`client.Ignicula`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/client#Ignicula). Подставьте свои `addr`, `Login` и `Hash`.

## 1. Dial

`Dial` устанавливает TCP, делает Hello, активирует v1 и возвращает уже готовый клиент. Лимит ожидания — `ConnectTimeout` в конфиге (по умолчанию 30s).

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

## 4. Массивы

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

## 6. Done и Err (по желанию)

Операции после обрыва и так вернут ошибку. `Done` — если нужно узнать о конце соединения без следующего запроса:

```go
go func() {
	<-ignicula.Done()
	log.Println(ignicula.Err())
}()
```

## 7. Конфиг и логгер

```go
cfg.ConnectTimeout = 5 * time.Second
cfg.RuntimeV1.MaxAttempts = 5

ignicula, err := client.Dial(addr, cfg, implemented.NewText(slog.LevelDebug))
```
