# Yotta 3.1 built-in nodes

Generated from the strict Node Authoring Projection `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`. Do not edit.

## `https://schemas.yotta.dev/nodes/text/concat/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.text.concat.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| input | `b` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/blob-to-stream/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.conversion.blobToStream.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
  - `stream`: `https://schemas.yotta.dev/capabilities/stream/session/v1`; target `stream-session`; risk `low`; consent `none`; operations `stream/cancel`, `stream/finish`, `stream/receive`, `stream/send`

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `blob` | `https://schemas.yotta.dev/types/core/binary/v1` | `durable-or-runtime` | `durable` | `required` | — |
| output | `stream` | `https://schemas.yotta.dev/types/core/binary/v1` | `durable-or-runtime` | `runtime` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/stream-to-blob/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.conversion.streamToBlob.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Capabilities:
  - `blob-write`: `https://schemas.yotta.dev/capabilities/blob/write/v1`; target `blob-store`; risk `low`; consent `none`; operations `append`, `cancel`, `commit`
  - `stream`: `https://schemas.yotta.dev/capabilities/stream/session/v1`; target `stream-session`; risk `low`; consent `none`; operations `stream/cancel`, `stream/receive`

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

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.Add.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/subtract/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.Sub.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/multiply/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.Mul.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/less-than/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.Lt.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/less-or-equal/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.LtEq.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/greater-than/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.Gt.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/greater-or-equal/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.GtEq.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/logic/and/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.And.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `true` |
| input | `b` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `true` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/logic/or/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.Or.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `false` |
| input | `b` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `false` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/logic/not/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.Not.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `default-available` | `false` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/contains/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.Contains.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| input | `search` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/length/v1`

- Authoring projection: `sha256:d29ac11df7a0465c640f4883c550d28490e31ebb955777f02b70dcd9b96b408e`
- Title key: `node.Length.label`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Capabilities: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `default-available` | `""` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.
