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

	"github.com/dejitarudemon/ignicula-go/client"
	"github.com/dejitarudemon/ignicula-go/configs"
	"github.com/dejitarudemon/ignicula-go/values"
)

func main() {
	cfg := configs.DefaultConnectionConfig()
	cfg.Auth = configs.AuthConfig{
		Login:    "user",
		Password: "secret",
	}

	ignicula, err := client.Dial("127.0.0.1:16000", cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ignicula.Close()

	ctx := context.Background()
	_ = ignicula.Write(ctx, []byte("greeting"), values.StringValue("hello"))
	val, _ := ignicula.Read(ctx, []byte("greeting"))
	fmt.Println(val)
}
```

`Dial` blocks until v1 is ready (`ConnectTimeout`, default 30s).

### CLI

```bash
go run ./cmd -addr 127.0.0.1:16000 -login user -password secret
```

See [CLI](docs/cli.en.md) for commands.

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

Тот же пример, что в [English](#quick-start): `Dial` сразу возвращает готовый клиент.

### CLI

```bash
go run ./cmd -addr 127.0.0.1:16000 -login user -password secret
```

Команды — в [CLI](docs/cli.md).
