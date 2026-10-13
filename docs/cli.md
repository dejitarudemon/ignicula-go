Language: [Русский](cli.md) · [English](cli.en.md)

# CLI

Интерактивный клиент в `cmd/`. Удобен для ручных проверок против локального или удалённого сервера.

## Запуск

```bash
go run ./cmd \
  -addr 127.0.0.1:16000 \
  -login user \
  -hash 0000000000000000000000000000000000000000000000000000000000000000
```

| Флаг | По умолчанию | Смысл |
| --- | --- | --- |
| `-addr` | `127.0.0.1:16000` | `host:port` |
| `-login` | `user` | Login для Handshake |
| `-hash` | 32 нулевых байта | Auth hash, 64 hex-символа |

После connect CLI печатает справочник команд, ждёт активации (`Ping`), затем показывает `>`. Если сервер рвёт соединение, процесс выходит с `disconnected: ...`.

## Команды

Каждая команда на своей строке (тот же список, что в приветствии):

```text
ping                                 send a Ping and wait for answer
read <key>                           read a value by key
write <key> <value> [type]            scalar (bytes|string|int64|uint64|float64|json; default bytes)
writetyped <key> <elemType> <elem>... typed array (alias: wt)
writeuntyped <key> <type:value>...    untyped array (alias: wu)
delete <key>                         delete a value by key
help                                 show this command list
quit                                 close connection and exit
```

Короткие алиасы: `p`, `r`, `w`, `wt`, `wu`, `d` / `del`, `h` / `?`, `q` / `exit`.

### Скалярная запись

```text
write greeting hello string
write blob raw-bytes
write n -42 int64
write j {"a":1} json
```

Имена типов (и короткие формы): `bytes`/`b`, `string`/`s`/`str`, `int64`/`i`/`int`, `uint64`/`u`/`uint`, `float64`/`f`/`float`, `json`/`j`.

### Typed array

Все элементы одного типа:

```text
writetyped tags string red green blue
wt nums int64 1 2 3
```

### Untyped array

Каждый элемент — `type:value` (без пробелов внутри токена):

```text
writeuntyped mix string:hello int64:-7 json:{"a":1}
wu mix float64:3.5 bytes:abc
```

Пустой untyped array:

```text
writeuntyped emptykey
```

## Замечания

- Ключи и скалярные строки берутся из строки как есть (без shell-quoting).
- JSON для `write` / `writeuntyped` — один токен без пробелов, например `{"a":1}`.
- `read` печатает краткое представление типа (`string "…"`, `typed_array[String] […]`, …).
