Language: [Русский](tutorial.md) · [English](tutorial.en.md)

# Туториалы

Работа с [`client.Ignicula`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/client#Ignicula). Подставьте свои `addr`, `Login` и `Password`.

## Гарантии

Что клиент обещает:

- **После успешного `Dial`** соединение активировано (Hello + Handshake), можно сразу вызывать `Ping` / `Read` / `Write` / `Delete`.
- **Каждая операция дожидается ответа** сервера (или ошибки / таймаута попытки). Успешный возврат из `Write` / `Delete` значит, что сервер прислал успешный Answer.
- **Transient-ошибки** (`RequestsConflict`, `InternalError`) клиент повторяет сам, до `RuntimeV1.MaxAttempts` (по умолчанию 3) с backoff.
- **Обрыв соединения** завершает висящие операции ошибкой; следующие вызовы тоже падают. Следить за `Done` не обязательно.
- **`Close` идемпотентен**.

Чего клиент не обещает:

- **Ровно-один раз на wire при ретраях** — при повторе запрос может уйти повторно; если нужна идемпотентность, её обычно обеспечивают на сервере или выборе ключа.
- **Глобальный порядок между горутинами** — параллельные вызовы с одного `Ignicula` допустимы, но порядок доставки и применения на сервере между ними не гарантирован.
- **Сохранность данных без ответа** — если `ctx` отменён или сокет оборвался до Answer, исход на сервере может быть неизвестен; при необходимости стоит перечитать значение.
- **Автоматический reconnect** — после `Done` соединение не восстанавливается само; достаточно снова вызвать `Dial`.

Таймауты по умолчанию: `ConnectTimeout` 30s (весь `Dial`), CRUD 30s на попытку, Ping 5s, socket read/write 100s.

## 1. Dial

`Dial` устанавливает TCP, делает Hello, активирует v1 и возвращает уже готовый клиент. Лимит ожидания — `ConnectTimeout` в конфиге (по умолчанию 30s).

```go
cfg := configs.DefaultConnectionConfig()
cfg.Auth = configs.AuthConfig{Login: "user", Password: "secret"}

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
if err := ignicula.Write(ctx, []byte("name"), values.StringValue("alice")); err != nil {
	log.Fatal(err)
}

val, err := ignicula.Read(ctx, []byte("name"))
if err != nil {
	log.Fatal(err)
}
fmt.Println(val)
```

## 4. Массивы

```go
tags, err := values.NewTypedArray(values.String, values.StringValue("a"), values.StringValue("b"))
if err != nil {
	log.Fatal(err)
}
if err := ignicula.Write(ctx, []byte("tags"), tags); err != nil {
	log.Fatal(err)
}

mix, err := values.NewUntypedArray(values.StringValue("hello"), values.Int64Value(-7))
if err != nil {
	log.Fatal(err)
}
if err := ignicula.Write(ctx, []byte("mix"), mix); err != nil {
	log.Fatal(err)
}
```

## 5. Delete

```go
if err := ignicula.Delete(ctx, []byte("name")); err != nil {
	log.Fatal(err)
}
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
