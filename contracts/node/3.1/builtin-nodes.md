# Yotta 3.1 built-in nodes

Generated from sealed Data Type and Node Contract artifacts. Do not edit.

## `https://schemas.yotta.dev/nodes/text/concat/v1`

- Title key: `node.text.concat.title`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Channel | Direction | Port | Type | Required |
| --- | --- | --- | --- | --- |
| data | input | `a` | `https://schemas.yotta.dev/types/core/string/v1` | true |
| data | input | `b` | `https://schemas.yotta.dev/types/core/string/v1` | true |
| data | output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | — |

Exec, Error, and Status ports: none.
