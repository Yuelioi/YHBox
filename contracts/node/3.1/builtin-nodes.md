# Yotta 3.1 built-in nodes

Generated from the strict Node Authoring Projection `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`. Do not edit.

## `https://schemas.yotta.dev/nodes/text/concat/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.text.concat.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| input | `b` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/blob-to-stream/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.conversion.blobToStream.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
  - `stream`: `https://schemas.yotta.dev/capabilities/stream/session/v1`; target `stream-session`; risk `low`; consent `none`; operations `stream/cancel`, `stream/finish`, `stream/receive`, `stream/send`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `blob` | `https://schemas.yotta.dev/types/core/binary/v1` | `durable-or-runtime` | `durable` | `required` | — |
| output | `stream` | `https://schemas.yotta.dev/types/core/binary/v1` | `durable-or-runtime` | `runtime` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/stream-to-blob/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.conversion.streamToBlob.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-write`: `https://schemas.yotta.dev/capabilities/blob/write/v1`; target `blob-store`; risk `low`; consent `none`; operations `append`, `cancel`, `commit`
  - `stream`: `https://schemas.yotta.dev/capabilities/stream/session/v1`; target `stream-session`; risk `low`; consent `none`; operations `stream/cancel`, `stream/receive`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `stream` | `https://schemas.yotta.dev/types/core/binary/v1` | `durable-or-runtime` | `runtime` | `required` | — |
| output | `blob` | `https://schemas.yotta.dev/types/core/binary/v1` | `durable-or-runtime` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `mediaType` | `text` | yes | `minLength: 3, maxLength: 255, pattern: ^[a-z0-9][a-z0-9!#$&^_.+-]+/[a-z0-9][a-z0-9!#$&^_.+-]+$` |

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/add/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Add.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/subtract/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Sub.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/multiply/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Mul.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/less-than/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Lt.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/less-or-equal/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.LtEq.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/greater-than/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Gt.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/greater-or-equal/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.GtEq.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/logic/and/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.And.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `true` |
| input | `b` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `true` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/logic/or/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Or.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `false` |
| input | `b` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `false` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/logic/not/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Not.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `false` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/contains/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Contains.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `search` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/length/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Length.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/collection/split/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Split.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `separator` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `","` |
| output | `result` | `list<https://schemas.yotta.dev/types/core/string/v1>` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/collection/join/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Join.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `list` | `list<https://schemas.yotta.dev/types/core/string/v1>` | `durable` | `durable` | `required` | — |
| input | `separator` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `","` |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/collection/length/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ListLength.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `list` | `list<$T>` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/collection/get/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ListGet.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `list` | `list<$T>` | `resolved-at-compile` | `durable` | `required` | — |
| input | `index` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `$T` | `resolved-at-compile` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/collection/contains/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ListContains.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `list` | `list<$T>` | `resolved-at-compile` | `durable` | `required` | — |
| input | `value` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/collection/append/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ListAppend.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `list` | `list<$T>` | `resolved-at-compile` | `durable` | `required` | — |
| input | `item` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `list<$T>` | `resolved-at-compile` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/collection/slice/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ListSlice.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `list` | `list<$T>` | `resolved-at-compile` | `durable` | `required` | — |
| input | `start` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `count` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `-1` |
| output | `result` | `list<$T>` | `resolved-at-compile` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/divide/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Div.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/modulo/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Mod.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/negate/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Neg.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/absolute/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Abs.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/minimum/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Min.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/maximum/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Max.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/floor/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Floor.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/ceiling/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Ceil.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/round/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Round.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `digits` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/clamp/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Clamp.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `minimum` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `maximum` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `100` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/power/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Pow.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `base` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `exponent` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `1` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/square-root/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Sqrt.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/equal/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Eq.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| input | `b` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/not-equal/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.NotEq.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| input | `b` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/replace/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Replace.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `old` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `new` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `all` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `true` |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/substring/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Substring.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `start` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `length` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `-1` |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/trim/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Trim.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/uppercase/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ToUpper.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/lowercase/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ToLower.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/index-of/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.IndexOf.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `search` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/starts-with/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.StartsWith.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `prefix` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/ends-with/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.EndsWith.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `suffix` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/regex-match/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.RegexMatch.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `pattern` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/regex-extract/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.RegexExtract.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `pattern` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/to-string/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ToString.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/string-to-number/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ToNumber.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/string-to-boolean/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ToBool.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `"false"` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/json/parse/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ParseJSON.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `"null"` |
| output | `result` | `https://schemas.yotta.dev/types/core/json/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/json/stringify/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ToJSON.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/json/path/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.JsonPath.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `json` | `https://schemas.yotta.dev/types/core/json/v1` | `durable` | `durable` | `required` | — |
| input | `path` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `"$"` |
| output | `result` | `https://schemas.yotta.dev/types/core/json/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/logic/select/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.Select.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `condition` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `true` |
| input | `when_true` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| input | `when_false` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `$T` | `resolved-at-compile` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/geometry/make-point/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.MakePoint.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `x` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `y` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `unit` | `https://schemas.yotta.dev/types/geometry/point-unit/v1` | `durable` | `durable` | `default-available` | `"ratio"` |
| output | `result` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/geometry/offset-point/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.OffsetPoint.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `point` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `required` | — |
| input | `offset_x` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `offset_y` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/geometry/point-distance/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.PointDistance.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `begin` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `required` | — |
| input | `end` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/geometry/region-around-point/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ROIAroundPoint.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `center` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `required` | — |
| input | `width` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.2` |
| input | `height` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.2` |
| output | `result` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/random/integer/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.random.integer.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `minimum` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `maximum` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `100` |
| input | `distribution` | `https://schemas.yotta.dev/types/random/distribution/v1` | `durable` | `durable` | `default-available` | `"uniform"` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/random/number/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.random.number.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `minimum` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `maximum` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `1` |
| input | `distribution` | `https://schemas.yotta.dev/types/random/distribution/v1` | `durable` | `durable` | `default-available` | `"uniform"` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/random/boolean/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.random.boolean.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `probability` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.5` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/random/choice/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.random.choice.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `list` | `list<$T>` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `$T` | `resolved-at-compile` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/time/observe/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.time.observe.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/state/read/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.state.read.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access:
  - `state`: `read` slot selected by config `variable`; type `$T`

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| output | `result` | `$T` | `resolved-at-compile` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `variable` | `state-variable` | yes | `minLength: 1, maxLength: 128, pattern: ^[A-Za-z0-9_][A-Za-z0-9._-]*$` |

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/state/write/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.state.write.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access:
  - `state`: `write` slot selected by config `variable`; type `$T`

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `$T` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `$T` | `resolved-at-compile` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `variable` | `state-variable` | yes | `minLength: 1, maxLength: 128, pattern: ^[A-Za-z0-9_][A-Za-z0-9._-]*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `done` |
Status events: none.

## `https://schemas.yotta.dev/nodes/state/metadata/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.state.metadata.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access:
  - `state`: `read` slot selected by config `variable`; type `$T`

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| output | `revision` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |
| output | `changed-at` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `variable` | `state-variable` | yes | `minLength: 1, maxLength: 128, pattern: ^[A-Za-z0-9_][A-Za-z0-9._-]*$` |

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/event/run-started/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.event.runStarted.title`
- Availability: `portable`
- Execution: `event` / `deterministic` / cache `none`
- Program instruction: `run-root` `{"kind":"run-root","runRoot":{"output":"started"}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `output` | `started` |
Status events: none.

## `https://schemas.yotta.dev/nodes/control/branch/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.control.branch.title`
- Availability: `portable`
- Execution: `control` / `deterministic` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `condition` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `true` |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `true` |
| `exec` | `output` | `false` |
Status events: none.

## `https://schemas.yotta.dev/nodes/control/delay/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.control.delay.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `duration-milliseconds` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `1000` |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `done` |
| `error` | `output` | `failed` |

| Status event | Category |
| --- | --- |
| `control.delay.waiting` | `waiting` |

## `https://schemas.yotta.dev/nodes/control/end-branch/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.control.endBranch.title`
- Availability: `portable`
- Execution: `control` / `deterministic` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
Status events: none.

## `https://schemas.yotta.dev/nodes/control/repeat/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.control.repeat.title`
- Availability: `portable`
- Execution: `region` / `deterministic` / cache `none`
- Program instruction: `counted-loop` `{"kind":"counted-loop","countedLoop":{"entryInput":"in","breakInput":"break","continueInput":"continue","bodyOutput":"body","completedOutput":"completed","countInput":"count","indexOutput":"index","ordinalType":{"typeId":"https://schemas.yotta.dev/types/core/integer/v1","semanticDigest":"sha256:b838d72eeaa6c0afb5d71e86851210e1f0d7d536bd2b7424509272a4bb78620c"},"maxIterations":10000}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `count` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `10` |
| output | `index` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `input` | `break` |
| `exec` | `input` | `continue` |
| `exec` | `output` | `body` |
| `exec` | `output` | `completed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/control/for-each/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.control.forEach.title`
- Availability: `portable`
- Execution: `region` / `deterministic` / cache `none`
- Program instruction: `for-each` `{"kind":"for-each","forEach":{"entryInput":"in","breakInput":"break","continueInput":"continue","bodyOutput":"body","completedOutput":"completed","itemsInput":"items","indexOutput":"index","itemOutput":"item","ordinalType":{"typeId":"https://schemas.yotta.dev/types/core/integer/v1","semanticDigest":"sha256:b838d72eeaa6c0afb5d71e86851210e1f0d7d536bd2b7424509272a4bb78620c"},"maxItems":10000}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `items` | `list<$T>` | `resolved-at-compile` | `durable` | `required` | — |
| output | `index` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |
| output | `item` | `$T` | `resolved-at-compile` | `durable` | `output` | — |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `input` | `break` |
| `exec` | `input` | `continue` |
| `exec` | `output` | `body` |
| `exec` | `output` | `completed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/control/retry/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.control.retry.title`
- Availability: `portable`
- Execution: `region` / `deterministic` / cache `none`
- Program instruction: `retry` `{"kind":"retry","retry":{"entryInput":"in","retryInput":"retry","bodyOutput":"body","completedOutput":"completed","exhaustedOutput":"exhausted","attemptsInput":"attempts","attemptOutput":"attempt","ordinalType":{"typeId":"https://schemas.yotta.dev/types/core/integer/v1","semanticDigest":"sha256:b838d72eeaa6c0afb5d71e86851210e1f0d7d536bd2b7424509272a4bb78620c"},"maxAttempts":100}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `attempts` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `3` |
| output | `attempt` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `error` | `input` | `retry` |
| `exec` | `output` | `body` |
| `exec` | `output` | `completed` |
| `exec` | `output` | `exhausted` |
Status events: none.

## `https://schemas.yotta.dev/nodes/ai/generate/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ai.generate.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `model`: `https://schemas.yotta.dev/capabilities/ai/generation/v1`; target `model`; risk `sensitive`; consent `once`; operations `generate`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `prompt` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `instructions` | `text` | no | `maxLength: 65536` |
| `maxOutputTokens` | `integer` | no | `minimum: 1, maximum: 1000000` |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |
| `temperature` | `number` | no | `minimum: 0, maximum: 2` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/ai/extract/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.ai.extract.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `model`: `https://schemas.yotta.dev/capabilities/ai/generation/v1`; target `model`; risk `sensitive`; consent `once`; operations `generate-structured`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `prompt` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/json/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `instructions` | `text` | no | `maxLength: 65536` |
| `maxOutputTokens` | `integer` | no | `minimum: 1, maximum: 1000000` |
| `schema` | `text` | yes | `minLength: 2, maxLength: 65536` |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |
| `temperature` | `number` | no | `minimum: 0, maximum: 2` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/script/execute/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.script.execute.title`
- Availability: `host-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features:
  - `isolation`: `https://schemas.yotta.dev/host-features/script-isolation/lpac-appcontainer-job/v1`
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `input` | `https://schemas.yotta.dev/types/core/json/v1` | `durable` | `durable` | `default-available` | `{}` |
| output | `result` | `https://schemas.yotta.dev/types/core/json/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `source` | `code` | yes | `minLength: 1, maxLength: 262144` |
| `timeoutMilliseconds` | `integer` | yes | `minimum: 1, maximum: 30000` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/filesystem/read-text/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.filesystem.readText.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `workspace-files`: `https://schemas.yotta.dev/capabilities/filesystem/read/v1`; target `workspace-files`; risk `low`; consent `none`; operations `read`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `path` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| output | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |
| output | `metadata` | `https://schemas.yotta.dev/types/filesystem/metadata/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `encoding` | `select` | no | `enum: "auto", "utf-8", "gbk", default hint: "auto"` |
| `maxBytes` | `integer` | no | `minimum: 1, maximum: 1048576, default hint: 1048576` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/filesystem/read-json/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.filesystem.readJSON.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `workspace-files`: `https://schemas.yotta.dev/capabilities/filesystem/read/v1`; target `workspace-files`; risk `low`; consent `none`; operations `read`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `path` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| output | `value` | `https://schemas.yotta.dev/types/core/json/v1` | `durable` | `durable` | `output` | — |
| output | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |
| output | `metadata` | `https://schemas.yotta.dev/types/filesystem/metadata/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `maxBytes` | `integer` | no | `minimum: 1, maximum: 1048576, default hint: 1048576` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/filesystem/stat/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.filesystem.stat.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `workspace-files`: `https://schemas.yotta.dev/capabilities/filesystem/read/v1`; target `workspace-files`; risk `low`; consent `none`; operations `stat`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `path` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| output | `metadata` | `https://schemas.yotta.dev/types/filesystem/metadata/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/network/http-get/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.network.httpGet.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `origin`: `https://schemas.yotta.dev/capabilities/network/http-get/v1`; target `origin`; risk `sensitive`; consent `once`; operations `get`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `path` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `"/"` |
| input | `query` | `https://schemas.yotta.dev/types/core/json/v1` | `durable` | `durable` | `default-available` | `{}` |
| output | `status` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |
| output | `body` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |
| output | `content-type` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/application/launch/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.application.launch.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `application`: `https://schemas.yotta.dev/capabilities/application/lifecycle/v1`; target `application`; risk `dangerous`; consent `once`; operations `launch`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/application/terminate/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.application.terminate.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `application`: `https://schemas.yotta.dev/capabilities/application/lifecycle/v1`; target `application`; risk `dangerous`; consent `once`; operations `terminate`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| output | `terminated-count` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/click-pointer/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.automation.clickPointer.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/input/v1`; target `target`; risk `dangerous`; consent `once`; operations `click`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `point` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `default-available` | `{"unit":"ratio","x":0.5,"y":0.5}` |
| input | `button` | `https://schemas.yotta.dev/types/automation/pointer-button/v1` | `durable` | `durable` | `default-available` | `"left"` |
| input | `hold-duration` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `50` |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/move-pointer/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.automation.movePointer.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/input/v1`; target `target`; risk `dangerous`; consent `once`; operations `move`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `point` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `default-available` | `{"unit":"ratio","x":0.5,"y":0.5}` |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/scroll-pointer/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.automation.scrollPointer.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/input/v1`; target `target`; risk `dangerous`; consent `once`; operations `scroll`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `point` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `default-available` | `{"unit":"ratio","x":0.5,"y":0.5}` |
| input | `notches` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `1` |
| input | `horizontal` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `false` |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/drag-pointer/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.automation.dragPointer.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/input/v1`; target `target`; risk `dangerous`; consent `once`; operations `drag`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `from` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `default-available` | `{"unit":"ratio","x":0.5,"y":0.5}` |
| input | `to` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `default-available` | `{"unit":"ratio","x":0.5,"y":0.5}` |
| input | `button` | `https://schemas.yotta.dev/types/automation/pointer-button/v1` | `durable` | `durable` | `default-available` | `"left"` |
| input | `duration` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `300` |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/move-pointer-relative/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.automation.movePointerRelative.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/input/v1`; target `target`; risk `dangerous`; consent `once`; operations `move-relative`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `delta-x` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `delta-y` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `duration` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `0` |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/press-keys/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.automation.pressKeys.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/input/v1`; target `target`; risk `dangerous`; consent `once`; operations `press-keys`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `keys` | `list<https://schemas.yotta.dev/types/automation/key-code/v1>` | `durable` | `durable` | `required` | — |
| input | `hold-duration` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `50` |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/type-text/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.automation.typeText.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/input/v1`; target `target`; risk `dangerous`; consent `once`; operations `type-text`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/activate-window/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.automation.activateWindow.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/window/v1`; target `target`; risk `dangerous`; consent `once`; operations `activate`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/capture-window/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.automation.captureWindow.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-write`: `https://schemas.yotta.dev/capabilities/blob/write/v1`; target `blob-store`; risk `low`; consent `none`; operations `append`, `cancel`, `commit`
  - `target`: `https://schemas.yotta.dev/capabilities/automation/capture/v1`; target `target`; risk `sensitive`; consent `once`; operations `capture`, `read-capture`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| output | `image` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/observability/log/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.observability.log.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `message` | `https://schemas.yotta.dev/types/observability/message/v1` | `durable` | `durable` | `required` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `level` | `select` | no | `enum: "debug", "info", "warn", "error", default hint: "info"` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/control/throw/v1`

- Authoring projection: `sha256:1f57fa8e7dcbd38464f16a26627486ff66f6638be2030ed7a4fda7b420738f79`
- Title key: `node.control.throw.title`
- Availability: `portable`
- Execution: `control` / `deterministic` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `message` | `https://schemas.yotta.dev/types/observability/message/v1` | `durable` | `durable` | `default-available` | `""` |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
Status events: none.
