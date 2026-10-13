Language: [Русский](how-it-works.md) · [English](how-it-works.en.md)

# How it works

High-level lifecycle of the Go client. Wire details are in [ignicula-wire](https://github.com/dejitarudemon/ignicula-wire).

## Connection lifecycle

```text
Dial(addr, cfg, log)
  │
  ├─ TCP connect
  ├─ Hello (protocol v0): advertise MustSupport versions
  └─ return *Ignicula
       │
       └─ background Start
            ├─ reader + writer goroutines
            ├─ Handshake (login + hash) on v1
            └─ runner Activate → ops allowed
```

1. **`client.Dial`** dials TCP, exchanges Hello, builds the connection, and starts activation asynchronously.
2. Until Handshake succeeds, `Ping` / `Read` / `Write` / `Delete` return `transport.ErrNoActiveVersion` (or a start failure).
3. After activation, the v1 runner owns request IDs, retries, and answer routing.
4. **`Close`** or a peer disconnect finishes the connection: `Done` closes and pending awaits fail.

## Layers

| Layer | Responsibility |
| --- | --- |
| `client` | Public façade, background `Start`, `Done` / `Err` |
| `transport` | Socket, Hello, read/write workers, disconnect |
| `runners/v1` | Handshake, Ping/Read/Write/Delete, retries |
| `adapters/v1` | Client `values.Value` ↔ wire value |
| `values` | Typed payloads used by the application |

## Operations

Each CRUD/Ping call:

1. Registers a request ID in the v1 table.
2. Encodes a frame and queues it for the writer.
3. Waits for a matching Answer (or ErrorAnswer) with an attempt timeout.
4. Retries on transient server errors (`RequestsConflict`, `InternalError`) up to `MaxAttempts`.

Timeouts come from `configs.RunnerV1Config` (per-op) and `ConnectionConfig` (socket read/write deadlines).

## Values

Wire and client share the same type codes: Bytes, TypedArray, UntypedArray, Int64, Uint64, Float64, String, JSON.

- Scalars: `values.New(typ, data)` or concrete types (`StringValue`, `Int64Value`, …).
- Homogeneous arrays: `values.NewTypedArray(elemType, elems...)`.
- Mixed arrays: `values.NewUntypedArray(elems...)`.
- `Write(ctx, key, data, typ...)` builds a scalar via `New`.
- `WriteValue(ctx, key, value)` accepts any `values.Value` (including arrays).

## Liveness

| Signal | Meaning |
| --- | --- |
| `Done()` | Connection is finished (local `Close`, peer EOF, or transport failure) |
| `Err()` | `nil` before `Done`; after Done: `ErrClosed` (clean local close) or an error wrapping `ErrDisconnected` |

Local `Close` does not treat “use of closed network connection” as a fatal reader error. Peer EOF aborts the runner so pending ops unblock quickly.

## Configuration defaults

`configs.DefaultConnectionConfig()`:

- Connect timeout: 30s
- Socket read/write deadlines: 100s (aligned with a typical server idle-ping window)
- MustSupport: version 1
- Runner: 10 KiB frames, batch 64, 3 attempts, 100ms backoff, 30s CRUD timeouts, 5s ping timeout

Auth (`Login`, 32-byte `Hash`) must be set by the caller before useful activation.
