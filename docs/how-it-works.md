Language: [Русский](how-it-works.md) · [English](how-it-works.en.md)

# Как это работает

Жизненный цикл Go-клиента. Детали кадров — в [ignicula-wire](https://github.com/dejitarudemon/ignicula-wire).

## Жизненный цикл соединения

```text
Dial(addr, cfg, log)
  │
  ├─ TCP connect
  ├─ Hello (протокол v0): список MustSupport
  └─ return *Ignicula
       │
       └─ фоновый Start
            ├─ goroutine reader + writer
            ├─ Handshake (login + hash) на v1
            └─ Activate runner → операции разрешены
```

1. **`client.Dial`** устанавливает TCP, обменивается Hello, собирает соединение и асинхронно запускает активацию.
2. Пока Handshake не прошёл, `Ping` / `Read` / `Write` / `Delete` возвращают `transport.ErrNoActiveVersion` (или ошибку Start).
3. После активации runner v1 владеет request ID, ретраями и маршрутизацией ответов.
4. **`Close`** или обрыв со стороны сервера завершает соединение: закрывается `Done`, висящие await падают.

## Слои

| Слой | Ответственность |
| --- | --- |
| `client` | Публичный фасад, фоновый `Start`, `Done` / `Err` |
| `transport` | Сокет, Hello, worker’ы чтения/записи, disconnect |
| `runners/v1` | Handshake, Ping/Read/Write/Delete, ретраи |
| `adapters/v1` | Клиентский `values.Value` ↔ wire |
| `values` | Типизированные значения приложения |

## Операции

Каждый вызов CRUD/Ping:

1. Регистрирует request ID в таблице v1.
2. Кодирует кадр и ставит его в очередь writer.
3. Ждёт Answer (или ErrorAnswer) с таймаутом попытки.
4. При transient-ошибках сервера (`RequestsConflict`, `InternalError`) повторяет до `MaxAttempts`.

Таймауты: `configs.RunnerV1Config` (на операцию) и `ConnectionConfig` (deadline сокета).

## Значения

Коды типов совпадают с wire: Bytes, TypedArray, UntypedArray, Int64, Uint64, Float64, String, JSON.

- Скаляры: `values.New(typ, data)` или конкретные типы (`StringValue`, `Int64Value`, …).
- Однородные массивы: `values.NewTypedArray(elemType, elems...)`.
- Смешанные: `values.NewUntypedArray(elems...)`.
- `Write(ctx, key, data, typ...)` собирает скаляр через `New`.
- `WriteValue(ctx, key, value)` принимает любой `values.Value` (включая массивы).

## Живость соединения

| Сигнал | Смысл |
| --- | --- |
| `Done()` | Соединение закончено (локальный `Close`, EOF пира или сбой транспорта) |
| `Err()` | До `Done` — `nil`; после: `ErrClosed` (штатный Close) или ошибка с `ErrDisconnected` |

Локальный `Close` не логирует «use of closed network connection» как FATAL. EOF пира abort’ит runner, чтобы висящие операции сразу разблокировались.

## Дефолты конфигурации

`configs.DefaultConnectionConfig()`:

- Таймаут dial: 30s
- Deadline read/write сокета: 100s (под окно idle-ping сервера)
- MustSupport: версия 1
- Runner: кадры 10 KiB, batch 64, 3 попытки, backoff 100ms, CRUD 30s, ping 5s

Auth (`Login`, hash на 32 байта) задаёт вызывающий код до полезной активации.
