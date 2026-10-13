Language: [Русский](cli.md) · [English](cli.en.md)

# CLI

Interactive client in `cmd/`. Useful for manual checks against a local or remote server.

## Run

```bash
go run ./cmd \
  -addr 127.0.0.1:16000 \
  -login user \
  -password secret
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-addr` | `127.0.0.1:16000` | `host:port` |
| `-login` | `user` | Handshake login |
| `-password` | empty | Password; client hashes it with Argon2id |
| `-anon` | `false` | Anonymous handshake (empty login, zero hash) |

`Dial` already waits for activation (`ConnectTimeout` in the config). After a successful dial the CLI prints the command reference and shows a `>` prompt. If the server drops the connection, the process exits with `disconnected: ...`.

## Commands

Each command on its own line (same list as the welcome banner):

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

Short aliases: `p`, `r`, `w`, `wt`, `wu`, `d` / `del`, `h` / `?`, `q` / `exit`.

### Scalar write

```text
write greeting hello string
write blob raw-bytes
write n -42 int64
write j {"a":1} json
```

Type names (and short forms): `bytes`/`b`, `string`/`s`/`str`, `int64`/`i`/`int`, `uint64`/`u`/`uint`, `float64`/`f`/`float`, `json`/`j`.

### Typed array

All elements share one type:

```text
writetyped tags string red green blue
wt nums int64 1 2 3
```

### Untyped array

Each element is `type:value` (no spaces inside a token unless the value itself has none):

```text
writeuntyped mix string:hello int64:-7 json:{"a":1}
wu mix float64:3.5 bytes:abc
```

Empty untyped array:

```text
writeuntyped emptykey
```

## Tips

- Keys and scalar string values are taken as raw text from the line (no shell-style quoting).
- JSON for `write` / `writeuntyped` must be a single token (no spaces), e.g. `{"a":1}`.
- `read` prints a typed summary (`string "…"`, `typed_array[String] […]`, …).
