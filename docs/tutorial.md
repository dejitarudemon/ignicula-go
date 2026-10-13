Language: [Русский](tutorial.md) · [English](tutorial.en.md)

# Туториалы

Пошаговая работа клиента с живым сервером Ignicula. Архитектура — в [how-it-works](how-it-works.md). Интерактивная оболочка: [CLI](cli.md).

Подставьте свои `addr`, `Login` и `Hash`.

## 1. Dial и ожидание готовности

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

Пока Handshake не завершён, `Ping` падает (активация ещё идёт). Повторяйте до успеха, дедлайна или `Done`. `nil` вместо логгера — текстовый логгер в stderr на уровне info.

## 2. Ping

```go
ctx := context.Background()
if err := cli.Ping(ctx); err != nil {
	log.Fatal(err)
}
```

## 3. Запись и чтение скаляров

`Write` собирает значение через `values.New`. Без типа пишутся **bytes**.

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

Числа и JSON:

```go
_ = cli.Write(ctx, []byte("n"), []byte("-42"), values.Int64)
_ = cli.Write(ctx, []byte("u"), []byte("42"), values.Uint64)
_ = cli.Write(ctx, []byte("f"), []byte("3.14"), values.Float64)
_ = cli.Write(ctx, []byte("j"), []byte(`{"ok":true}`), values.JSON)
```

## 4. Массивы через WriteValue

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

`Read` возвращает те же клиентские типы (включая массивы) после конвертации адаптером.

## 5. Delete

```go
if err := cli.Delete(ctx, []byte("user:1:name")); err != nil {
	log.Fatal(err)
}
```

## 6. Следить за обрывом

```go
go func() {
	<-cli.Done()
	log.Println("connection ended:", cli.Err())
}()

select {
case <-cli.Done():
	if err := cli.Err(); err != nil {
		log.Println("disconnected:", err)
	}
default:
}
```

После `Done` дальнейшие операции возвращают ту же ошибку, что `Err()` (обычно с `transport.ErrDisconnected`, либо `transport.ErrClosed` после штатного `Close`).

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
cli, err := client.Dial(addr, cfg, log)
```

Подойдёт любая реализация `logger.Logger`, либо `nil` для дефолта.
