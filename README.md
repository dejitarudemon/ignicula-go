[![CI](https://github.com/dejitarudemon/ignicula-go/actions/workflows/ci.yml/badge.svg)](https://github.com/dejitarudemon/ignicula-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/dejitarudemon/ignicula-go.svg)](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go)
[![Release](https://img.shields.io/github/v/tag/dejitarudemon/ignicula-go?label=release)](https://github.com/dejitarudemon/ignicula-go/releases)
[![License: MIT](https://img.shields.io/github/license/dejitarudemon/ignicula-go)](LICENSE)

# ignicula-go

Go client for an [Ignicula](https://github.com/dejitarudemon) server: dial, activate protocol v1, then Ping / Read / Write / Delete.

[English](#english) · [Русский](#русский)

Wire framing lives in [`ignicula-wire`](https://github.com/dejitarudemon/ignicula-wire). This module is the TCP client on top of it.

---

## English

### Docs

- [Tutorials](docs/tutorial.en.md)
- [CLI](docs/cli.en.md)

### Install

```bash
go get github.com/dejitarudemon/ignicula-go@latest
```

Requires Go 1.27+.

### Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dejitarudemon/ignicula-go/client"
	"github.com/dejitarudemon/ignicula-go/configs"
	"github.com/dejitarudemon/ignicula-go/values"
)

func main() {
	cfg := configs.DefaultConnectionConfig()
	cfg.Auth = configs.AuthConfig{
		Login: "user",
		Hash:  [32]byte{}, // 32-byte auth hash from your server
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
			break
		}
		select {
		case <-ctx.Done():
			log.Fatal("activate:", ctx.Err())
		case <-cli.Done():
			log.Fatal("disconnected:", cli.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}

	if err := cli.Write(ctx, []byte("greeting"), []byte("hello"), values.String); err != nil {
		log.Fatal(err)
	}
	val, err := cli.Read(ctx, []byte("greeting"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%T = %v\n", val, val)
}
```

`Dial` runs Hello and v1 activation in the background. Wait until `Ping` succeeds (or watch `Done`) before issuing other ops.

### Packages

| Package | Role |
| --- | --- |
| [`client`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/client) | Public API: `Dial`, CRUD, `Done` / `Err` |
| [`configs`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/configs) | Timeouts, auth, runner limits |
| [`values`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/values) | Typed values and array builders |
| [`logger`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/logger) | Optional logging interface |

### CLI

```bash
go run ./cmd -addr 127.0.0.1:16000 -login user -hash <64 hex chars>
```

See [CLI](docs/cli.en.md) for commands.

### License

[MIT](LICENSE)

---

## Русский

Go-клиент сервера [Ignicula](https://github.com/dejitarudemon): dial, активация протокола v1, затем Ping / Read / Write / Delete.

Кадры протокола — в [`ignicula-wire`](https://github.com/dejitarudemon/ignicula-wire). Этот модуль — TCP-клиент поверх него.

### Документация

- [Туториалы](docs/tutorial.md)
- [CLI](docs/cli.md)

### Установка

```bash
go get github.com/dejitarudemon/ignicula-go@latest
```

Нужен Go 1.27+.

### Быстрый старт

Тот же пример, что в [English](#quick-start): `Dial` → дождаться успешного `Ping` → `Write` / `Read`. Активация идёт в фоне; операции до готовности вернут ошибку (часто «no active protocol version»).

### Пакеты

| Пакет | Назначение |
| --- | --- |
| [`client`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/client) | Публичный API: `Dial`, CRUD, `Done` / `Err` |
| [`configs`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/configs) | Таймауты, auth, лимиты runner |
| [`values`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/values) | Типизированные значения и массивы |
| [`logger`](https://pkg.go.dev/github.com/dejitarudemon/ignicula-go/logger) | Опциональный логгер |

### CLI

```bash
go run ./cmd -addr 127.0.0.1:16000 -login user -hash <64 hex символа>
```

Команды — в [CLI](docs/cli.md).

### Лицензия

[MIT](LICENSE)
