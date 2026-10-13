Language: [Русский](tutorial.md) · [English](tutorial.en.md)

# Туториалы

Пошаговая работа с типом [`client.Ignicula`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/client#Ignicula) против живого сервера.

Подставьте свои `addr`, `Login` и `Hash`.

## 1. Dial и ожидание готовности

`Dial` возвращает `*Ignicula` и в фоне запускает Hello + Handshake. Операции до активации падают — обычно ждут успешного `Ping`.

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
		Hash:  [32]byte{ /* 32 байта */ },
	}

	ignicula, err := client.Dial("127.0.0.1:16000", cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ignicula.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for {
		if err := ignicula.Ping(ctx); err == nil {
			log.Println("ready")
			break
		}
		select {
		case <-ctx.Done():
			log.Fatal("activate:", ctx.Err())
		case <-ignicula.Done():
			log.Fatal("disconnected:", ignicula.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}

	// дальше: Ping / Read / Write / Delete на этом же ignicula
}
```

`nil` вместо логгера — текстовый логгер в stderr на уровне info.

## 2. Ping

```go
ctx := context.Background()
if err := ignicula.Ping(ctx); err != nil {
	log.Fatal(err)
}
```

## 3. Запись и чтение скаляров

`Ignicula.Write` собирает значение через `values.New`. Без типа пишутся **bytes**.

```go
import "github.com/dejitarudemon/ignicula-go/values"

ctx := context.Background()
key := []byte("user:1:name")

if err := ignicula.Write(ctx, key, []byte("alice"), values.String); err != nil {
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

Числа и JSON:

```go
_ = ignicula.Write(ctx, []byte("n"), []byte("-42"), values.Int64)
_ = ignicula.Write(ctx, []byte("u"), []byte("42"), values.Uint64)
_ = ignicula.Write(ctx, []byte("f"), []byte("3.14"), values.Float64)
_ = ignicula.Write(ctx, []byte("j"), []byte(`{"ok":true}`), values.JSON)
```

## 4. Массивы через WriteValue

Для массивов (и любого готового `values.Value`) используйте `Ignicula.WriteValue`:

```go
typed, err := values.NewTypedArray(
	values.String,
	values.StringValue("a"),
	values.StringValue("b"),
)
if err != nil {
	log.Fatal(err)
}
if err := ignicula.WriteValue(ctx, []byte("tags"), typed); err != nil {
	log.Fatal(err)
}

mixed, err := values.NewUntypedArray(
	values.StringValue("hello"),
	values.Int64Value(-7),
)
if err != nil {
	log.Fatal(err)
}
if err := ignicula.WriteValue(ctx, []byte("mix"), mixed); err != nil {
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

## 6. Следить за обрывом

У `Ignicula` есть `Done` / `Err` — то же, что у соединения под капотом:

```go
go func() {
	<-ignicula.Done()
	log.Println("connection ended:", ignicula.Err())
}()

// ... работа с ignicula ...

select {
case <-ignicula.Done():
	if err := ignicula.Err(); err != nil {
		log.Println("disconnected:", err)
	}
default:
}
```

После `Done` дальнейшие вызовы `Ping` / `Read` / `Write` / `Delete` возвращают ту же ошибку, что `Err()` (обычно с `ErrDisconnected`, либо `ErrClosed` после штатного `Close`).

## 7. Свой конфиг и логгер

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

Подойдёт любая реализация `logger.Logger`, либо `nil` для дефолта.
