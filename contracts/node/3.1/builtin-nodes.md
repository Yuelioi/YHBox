# Yotta 3.1 built-in nodes

Generated from sealed Data Type and Node Contract artifacts. Do not edit.

## `https://schemas.yotta.dev/nodes/text/concat/v1`

- Title key: `node.text.concat.title`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Channel | Direction | Port | Type | Required | Resource lease |
| --- | --- | --- | --- | --- | --- |
| data | input | `a` | `https://schemas.yotta.dev/types/core/string/v1` | true | — |
| data | input | `b` | `https://schemas.yotta.dev/types/core/string/v1` | true | — |
| data | output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | — | — |

Exec, Error, and Status ports: none.

## `https://schemas.yotta.dev/nodes/conversion/blob-to-stream/v1`

- Title key: `node.conversion.blobToStream.title`
- Execution: `effect` / `recorded` / cache `none`
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1` operations `read-range`
  - `stream`: `https://schemas.yotta.dev/capabilities/stream/session/v1` operations `stream/cancel`, `stream/finish`, `stream/receive`, `stream/send`

| Channel | Direction | Port | Type | Required | Resource lease |
| --- | --- | --- | --- | --- | --- |
| data | input | `blob` | `https://schemas.yotta.dev/types/core/binary/v1` | true | — |
| data | output | `stream` | `https://schemas.yotta.dev/types/core/binary/v1` | — | `stream` (`stream/cancel`, `stream/receive`) |

Exec, Error, and Status ports: none.

## `https://schemas.yotta.dev/nodes/conversion/stream-to-blob/v1`

- Title key: `node.conversion.streamToBlob.title`
- Execution: `effect` / `recorded` / cache `none`
- Capabilities:
  - `blob-write`: `https://schemas.yotta.dev/capabilities/blob/write/v1` operations `append`, `cancel`, `commit`
  - `stream`: `https://schemas.yotta.dev/capabilities/stream/session/v1` operations `stream/cancel`, `stream/receive`

| Channel | Direction | Port | Type | Required | Resource lease |
| --- | --- | --- | --- | --- | --- |
| data | input | `stream` | `https://schemas.yotta.dev/types/core/binary/v1` | true | `stream` (`stream/cancel`, `stream/receive`) |
| data | output | `blob` | `https://schemas.yotta.dev/types/core/binary/v1` | — | — |

Exec, Error, and Status ports: none.
