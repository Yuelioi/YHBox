# Yotta 3.1 built-in nodes

Generated from the strict Node Authoring Projection `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`. Do not edit.

## Type capability matrix

Generated closure view. A missing applicable capability fails Catalog construction.

| Type | Traits | Produced | Consumed | Structure break | Conversions | Waiver |
| --- | --- | --- | --- | --- | --- | --- |
| `https://schemas.yotta.dev/types/automation/held-input/v1` |  | yes | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/automation/input-clip/v1` |  | no | yes | no | 0 | created and selected through the recording asset library |
| `https://schemas.yotta.dev/types/automation/key-code/v1` | durable, equatable, observable | no | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/automation/macro/v1` |  | no | yes | no | 0 | created and selected through the macro asset library |
| `https://schemas.yotta.dev/types/automation/pointer-button/v1` | durable, equatable, observable | no | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/core/binary/v1` |  | yes | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/core/boolean/v1` | durable, equatable, observable | yes | yes | no | 1 |  |
| `https://schemas.yotta.dev/types/core/integer/v1` | durable, equatable, numeric, observable, ordered | yes | yes | no | 5 |  |
| `https://schemas.yotta.dev/types/core/json/v1` | durable, equatable, observable | yes | yes | no | 1 |  |
| `https://schemas.yotta.dev/types/core/number/v1` | durable, equatable, numeric, observable, ordered | yes | yes | no | 5 |  |
| `https://schemas.yotta.dev/types/core/string/v1` | durable, equatable, observable | yes | yes | no | 5 |  |
| `https://schemas.yotta.dev/types/filesystem/metadata/v1` | durable, equatable, observable | yes | yes | yes | 0 |  |
| `https://schemas.yotta.dev/types/geometry/point-unit/v1` | durable, equatable, observable | yes | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/geometry/point/v1` | durable, equatable, observable | yes | yes | yes | 0 |  |
| `https://schemas.yotta.dev/types/geometry/region/v1` | durable, equatable, observable | yes | yes | yes | 0 |  |
| `https://schemas.yotta.dev/types/media/image/v1` |  | yes | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/observability/message/v1` | durable, equatable, observable | no | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/random/distribution/v1` | durable, equatable, observable | no | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | durable, equatable, observable | no | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/vision/color-blob/v1` | durable, equatable, observable | yes | yes | yes | 0 |  |
| `https://schemas.yotta.dev/types/vision/color-range/v1` | durable, equatable, observable | no | yes | no | 0 |  |
| `https://schemas.yotta.dev/types/vision/qr-code/v1` | durable, equatable, observable | yes | yes | yes | 0 |  |
| `https://schemas.yotta.dev/types/vision/template-match/v1` | durable, equatable, observable | yes | yes | yes | 0 |  |

## `https://schemas.yotta.dev/nodes/text/concat`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/conversion/blob-to-stream`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/conversion/stream-to-blob`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/math/add`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-add.title`
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

## `https://schemas.yotta.dev/nodes/math/subtract`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-subtract.title`
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

## `https://schemas.yotta.dev/nodes/math/multiply`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-multiply.title`
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

## `https://schemas.yotta.dev/nodes/comparison/less-than`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.comparison-less-than.title`
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

## `https://schemas.yotta.dev/nodes/comparison/less-or-equal`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.comparison-less-or-equal.title`
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

## `https://schemas.yotta.dev/nodes/comparison/greater-than`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.comparison-greater-than.title`
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

## `https://schemas.yotta.dev/nodes/comparison/greater-or-equal`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.comparison-greater-or-equal.title`
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

## `https://schemas.yotta.dev/nodes/logic/and`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.logic-and.title`
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

## `https://schemas.yotta.dev/nodes/logic/or`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.logic-or.title`
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

## `https://schemas.yotta.dev/nodes/logic/not`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.logic-not.title`
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

## `https://schemas.yotta.dev/nodes/text/contains`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-contains.title`
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

## `https://schemas.yotta.dev/nodes/text/length`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-length.title`
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

## `https://schemas.yotta.dev/nodes/collection/split`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.collection-split.title`
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

## `https://schemas.yotta.dev/nodes/collection/join`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.collection-join.title`
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

## `https://schemas.yotta.dev/nodes/collection/length`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.collection-length.title`
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

## `https://schemas.yotta.dev/nodes/collection/get`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.collection-get.title`
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

## `https://schemas.yotta.dev/nodes/collection/contains`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.collection-contains.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `list` | `list<$E>` | `resolved-at-compile` | `durable` | `required` | — |
| input | `value` | `$E` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/collection/append`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.collection-append.title`
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

## `https://schemas.yotta.dev/nodes/collection/slice`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.collection-slice.title`
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

## `https://schemas.yotta.dev/nodes/math/divide`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-divide.title`
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

## `https://schemas.yotta.dev/nodes/math/modulo`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-modulo.title`
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

## `https://schemas.yotta.dev/nodes/math/negate`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-negate.title`
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

## `https://schemas.yotta.dev/nodes/math/absolute`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-absolute.title`
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

## `https://schemas.yotta.dev/nodes/math/minimum`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-minimum.title`
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

## `https://schemas.yotta.dev/nodes/math/maximum`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-maximum.title`
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

## `https://schemas.yotta.dev/nodes/math/floor`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-floor.title`
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

## `https://schemas.yotta.dev/nodes/math/ceiling`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-ceiling.title`
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

## `https://schemas.yotta.dev/nodes/math/round`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-round.title`
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

## `https://schemas.yotta.dev/nodes/math/clamp`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-clamp.title`
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

## `https://schemas.yotta.dev/nodes/math/power`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-power.title`
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

## `https://schemas.yotta.dev/nodes/math/square-root`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-square-root.title`
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

## `https://schemas.yotta.dev/nodes/math/integer-add`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-integer-add.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/integer-subtract`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-integer-subtract.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/integer-multiply`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-integer-multiply.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/integer-modulo`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-integer-modulo.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/integer-negate`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-integer-negate.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/integer-absolute`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-integer-absolute.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/integer-minimum`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-integer-minimum.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/integer-maximum`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-integer-maximum.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `b` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/math/integer-clamp`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.math-integer-clamp.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `minimum` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| input | `maximum` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `100` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/equal`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.comparison-equal.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `$E` | `resolved-at-compile` | `durable` | `required` | — |
| input | `b` | `$E` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/comparison/not-equal`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.comparison-not-equal.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `a` | `$E` | `resolved-at-compile` | `durable` | `required` | — |
| input | `b` | `$E` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/text/replace`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-replace.title`
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

## `https://schemas.yotta.dev/nodes/text/substring`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-substring.title`
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

## `https://schemas.yotta.dev/nodes/text/trim`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-trim.title`
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

## `https://schemas.yotta.dev/nodes/text/uppercase`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-uppercase.title`
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

## `https://schemas.yotta.dev/nodes/text/lowercase`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-lowercase.title`
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

## `https://schemas.yotta.dev/nodes/text/index-of`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-index-of.title`
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

## `https://schemas.yotta.dev/nodes/text/starts-with`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-starts-with.title`
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

## `https://schemas.yotta.dev/nodes/text/ends-with`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-ends-with.title`
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

## `https://schemas.yotta.dev/nodes/text/regex-match`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-regex-match.title`
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

## `https://schemas.yotta.dev/nodes/text/regex-extract`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.text-regex-extract.title`
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

## `https://schemas.yotta.dev/nodes/conversion/to-string`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.conversion-to-string.title`
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

## `https://schemas.yotta.dev/nodes/conversion/string-to-number`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.conversion-string-to-number.title`
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

## `https://schemas.yotta.dev/nodes/conversion/string-to-integer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.conversion-string-to-integer.title`
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

## `https://schemas.yotta.dev/nodes/conversion/truncate-to-integer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.conversion-truncate-to-integer.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/floor-to-integer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.conversion-floor-to-integer.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/ceiling-to-integer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.conversion-ceiling-to-integer.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/round-to-integer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.conversion-round-to-integer.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `result` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/conversion/string-to-boolean`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.conversion-string-to-boolean.title`
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

## `https://schemas.yotta.dev/nodes/json/parse`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.json-parse.title`
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

## `https://schemas.yotta.dev/nodes/json/stringify`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.json-stringify.title`
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

## `https://schemas.yotta.dev/nodes/json/path`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.json-path.title`
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

## `https://schemas.yotta.dev/nodes/logic/select`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.logic-select.title`
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

## `https://schemas.yotta.dev/nodes/geometry/make-point`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.geometry-make-point.title`
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

## `https://schemas.yotta.dev/nodes/geometry/offset-point`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.geometry-offset-point.title`
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

## `https://schemas.yotta.dev/nodes/geometry/point-distance`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.geometry-point-distance.title`
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

## `https://schemas.yotta.dev/nodes/geometry/region-around-point`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.builtin.geometry-region-around-point.title`
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

## `https://schemas.yotta.dev/nodes/random/integer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/random/number`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/random/boolean`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/random/choice`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/time/observe`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/state/read`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/state/write`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/state/metadata`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/state/last-change`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.state.lastChange.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access:
  - `state`: `read` slot selected by config `variable`; type `$T`

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| output | `changed-at` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `variable` | `state-variable` | yes | `minLength: 1, maxLength: 128, pattern: ^[A-Za-z0-9_][A-Za-z0-9._-]*$` |

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/state/increment`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.state.increment.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access:
  - `state`: `write` slot selected by config `variable`; type `$N`

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `delta` | `$N` | `resolved-at-compile` | `durable` | `required` | — |
| output | `result` | `$N` | `resolved-at-compile` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `variable` | `state-variable` | yes | `minLength: 1, maxLength: 128, pattern: ^[A-Za-z0-9_][A-Za-z0-9._-]*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `done` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/event/run-started`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/control/branch`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/control/delay`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/control/end-branch`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/control/repeat`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.control.repeat.title`
- Availability: `portable`
- Execution: `region` / `deterministic` / cache `none`
- Program instruction: `counted-loop` `{"kind":"counted-loop","countedLoop":{"entryInput":"in","breakInput":"break","continueInput":"continue","bodyOutput":"body","completedOutput":"completed","countInput":"count","indexOutput":"index","ordinalType":{"typeId":"https://schemas.yotta.dev/types/core/integer/v1","semanticDigest":"sha256:2cba7041419de83ae27630871f5c833ec42ba1f701bbc140a6b200a81a2f8c2d"},"maxIterations":10000}}`
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

## `https://schemas.yotta.dev/nodes/control/for-each`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.control.forEach.title`
- Availability: `portable`
- Execution: `region` / `deterministic` / cache `none`
- Program instruction: `for-each` `{"kind":"for-each","forEach":{"entryInput":"in","breakInput":"break","continueInput":"continue","bodyOutput":"body","completedOutput":"completed","itemsInput":"items","indexOutput":"index","itemOutput":"item","ordinalType":{"typeId":"https://schemas.yotta.dev/types/core/integer/v1","semanticDigest":"sha256:2cba7041419de83ae27630871f5c833ec42ba1f701bbc140a6b200a81a2f8c2d"},"maxItems":10000}}`
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

## `https://schemas.yotta.dev/nodes/control/retry`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.control.retry.title`
- Availability: `portable`
- Execution: `region` / `deterministic` / cache `none`
- Program instruction: `retry` `{"kind":"retry","retry":{"entryInput":"in","retryInput":"retry","bodyOutput":"body","completedOutput":"completed","exhaustedOutput":"exhausted","attemptsInput":"attempts","attemptOutput":"attempt","ordinalType":{"typeId":"https://schemas.yotta.dev/types/core/integer/v1","semanticDigest":"sha256:2cba7041419de83ae27630871f5c833ec42ba1f701bbc140a6b200a81a2f8c2d"},"maxAttempts":100}}`
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

## `https://schemas.yotta.dev/nodes/control/switch`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.control.switch.title`
- Availability: `portable`
- Execution: `control` / `deterministic` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `$T` | `resolved-at-compile` | `durable` | `required` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `caseCount` | `integer` | no | `minimum: 1, maximum: 32, default hint: 3` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `default` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/time/stopwatch-start`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.time.stopwatchStart.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| output | `started-at` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/time/stopwatch-read`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.time.stopwatchRead.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `started-at` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `required` | — |
| output | `elapsed` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/time/stopwatch-stop`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.time.stopwatchStop.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `started-at` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `required` | — |
| output | `elapsed` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/ai/generate`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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
| `maxOutputTokens` | `integer` | no | `minimum: 1, maximum: 1000000` |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |
| `temperature` | `number` | no | `minimum: 0, maximum: 2` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/ai/extract`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/ai/agent`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.ai.agent.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `model`: `https://schemas.yotta.dev/capabilities/ai/generation/v1`; target `model`; risk `sensitive`; consent `once`; operations `agent-continue`, `agent-start`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `prompt` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| input | `context` | `https://schemas.yotta.dev/types/core/json/v1` | `durable` | `durable` | `optional` | — |
| output | `result` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `maxCostMicrounits` | `integer` | yes | `minimum: 1, maximum: 1000000000000` |
| `maxInputTokens` | `integer` | yes | `minimum: 1, maximum: 100000000` |
| `maxIterations` | `integer` | yes | `minimum: 1, maximum: 64` |
| `maxOutputTokens` | `integer` | yes | `minimum: 1, maximum: 1000000` |
| `maxParallelism` | `integer` | yes | `minimum: 1, maximum: 32` |
| `maxToolCalls` | `integer` | yes | `minimum: 1, maximum: 256` |
| `maxTotalOutputTokens` | `integer` | yes | `minimum: 1, maximum: 100000000` |
| `maxWallTimeMillis` | `integer` | yes | `minimum: 1, maximum: 3600000` |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |

| Status event | Category |
| --- | --- |
| `ai.agent_tool_calls` | `progress` |
| `ai.agent_turn` | `progress` |

## `https://schemas.yotta.dev/nodes/script/execute`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/filesystem/read-text`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.filesystem.readText.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `workspace-files`: `https://schemas.yotta.dev/capabilities/filesystem/workspace/v1`; target `workspace-files`; risk `low`; consent `none`; operations `read`
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

## `https://schemas.yotta.dev/nodes/filesystem/read-json`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.filesystem.readJSON.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `workspace-files`: `https://schemas.yotta.dev/capabilities/filesystem/workspace/v1`; target `workspace-files`; risk `low`; consent `none`; operations `read`
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

## `https://schemas.yotta.dev/nodes/filesystem/stat`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.filesystem.stat.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `workspace-files`: `https://schemas.yotta.dev/capabilities/filesystem/workspace/v1`; target `workspace-files`; risk `low`; consent `none`; operations `stat`
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

## `https://schemas.yotta.dev/nodes/filesystem/load-image`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.filesystem.loadImage.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-write`: `https://schemas.yotta.dev/capabilities/blob/write/v1`; target `blob-store`; risk `low`; consent `none`; operations `append`, `cancel`, `commit`
  - `workspace-files`: `https://schemas.yotta.dev/capabilities/filesystem/workspace/v1`; target `workspace-files`; risk `low`; consent `none`; operations `read-range`, `stat`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `path` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| output | `image` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `output` | — |
| output | `metadata` | `https://schemas.yotta.dev/types/filesystem/metadata/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `maxBytes` | `integer` | no | `minimum: 1, maximum: 33554432, default hint: 33554432` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/filesystem/save-image`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.filesystem.saveImage.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
  - `workspace-files`: `https://schemas.yotta.dev/capabilities/filesystem/workspace/v1`; target `workspace-files`; risk `low`; consent `none`; operations `write-append`, `write-cancel`, `write-commit`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `image` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `path` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `required` | — |
| output | `metadata` | `https://schemas.yotta.dev/types/filesystem/metadata/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `overwrite` | `toggle` | no | `default hint: false` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/network/http-get`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/application/launch`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/application/terminate`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/automation/click-pointer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/automation/move-pointer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/automation/scroll-pointer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/automation/drag-pointer`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/automation/move-pointer-relative`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.movePointerRelative.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/desktop-input/v1`; target `target`; risk `dangerous`; consent `once`; operations `move-relative`
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

## `https://schemas.yotta.dev/nodes/automation/press-keys`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.pressKeys.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/key-input/v1`; target `target`; risk `dangerous`; consent `once`; operations `press-keys`
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

## `https://schemas.yotta.dev/nodes/automation/type-text`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/automation/hold-keys`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.holdKeys.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/held-input/v1`; target `target`; risk `dangerous`; consent `once`; operations `hold-button`, `hold-keys`, `release-held`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `keys` | `list<https://schemas.yotta.dev/types/automation/key-code/v1>` | `durable` | `durable` | `required` | — |
| output | `held` | `https://schemas.yotta.dev/types/automation/held-input/v1` | `runtime-only` | `runtime` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/hold-pointer-button`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.holdPointerButton.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/held-input/v1`; target `target`; risk `dangerous`; consent `once`; operations `hold-button`, `hold-keys`, `release-held`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `point` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `required` | — |
| input | `button` | `https://schemas.yotta.dev/types/automation/pointer-button/v1` | `durable` | `durable` | `default-available` | `"left"` |
| output | `held` | `https://schemas.yotta.dev/types/automation/held-input/v1` | `runtime-only` | `runtime` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/release-held-input`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.releaseHeldInput.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/held-input/v1`; target `target`; risk `dangerous`; consent `once`; operations `release-held`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `held` | `https://schemas.yotta.dev/types/automation/held-input/v1` | `runtime-only` | `runtime` | `required` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/close-window`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.closeWindow.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/window/v1`; target `target`; risk `dangerous`; consent `once`; operations `close-window`
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

## `https://schemas.yotta.dev/nodes/automation/move-resize-window`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.moveResizeWindow.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/window/v1`; target `target`; risk `dangerous`; consent `once`; operations `move-resize-window`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `x` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `required` | — |
| input | `y` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `required` | — |
| input | `width` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `required` | — |
| input | `height` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `required` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/maximize-window`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.maximizeWindow.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/window/v1`; target `target`; risk `dangerous`; consent `once`; operations `set-window-state`
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

## `https://schemas.yotta.dev/nodes/automation/minimize-window`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.minimizeWindow.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/window/v1`; target `target`; risk `dangerous`; consent `once`; operations `set-window-state`
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

## `https://schemas.yotta.dev/nodes/automation/restore-window`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.restoreWindow.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/window/v1`; target `target`; risk `dangerous`; consent `once`; operations `set-window-state`
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

## `https://schemas.yotta.dev/nodes/automation/get-window-state`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.getWindowState.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/window/v1`; target `target`; risk `dangerous`; consent `once`; operations `get-window-state`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| output | `state` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |
| output | `foreground` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |
| output | `x` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |
| output | `y` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |
| output | `width` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |
| output | `height` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/wait-window`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.waitWindow.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/window/v1`; target `target`; risk `dangerous`; consent `once`; operations `wait-window`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `timeout` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `10000` |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `found` |
| `exec` | `output` | `timeout` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/wait-window-gone`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.waitWindowGone.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/window/v1`; target `target`; risk `dangerous`; consent `once`; operations `wait-window-gone`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `timeout` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `10000` |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `gone` |
| `exec` | `output` | `timeout` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/wait-template`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.waitTemplate.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
  - `capture-target`: `https://schemas.yotta.dev/capabilities/automation/capture/v1`; target `target`; risk `sensitive`; consent `once`; operations `capture`, `read-capture`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `template` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| input | `threshold` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.85` |
| input | `timeout` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `5000` |
| input | `poll-interval` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `100` |
| input | `settle-duration` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `200` |
| output | `matched` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |
| output | `score` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `center` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `output` | — |
| output | `bounds` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `found` |
| `exec` | `output` | `timeout` |
| `error` | `output` | `failed` |

| Status event | Category |
| --- | --- |
| `automation.template.matched` | `progress` |
| `automation.template.timeout` | `progress` |
| `automation.template.waiting` | `waiting` |

## `https://schemas.yotta.dev/nodes/automation/click-template`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.clickTemplate.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
  - `capture-target`: `https://schemas.yotta.dev/capabilities/automation/capture/v1`; target `target`; risk `sensitive`; consent `once`; operations `capture`, `read-capture`
  - `input-target`: `https://schemas.yotta.dev/capabilities/automation/input/v1`; target `target`; risk `dangerous`; consent `once`; operations `click`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `template` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| input | `threshold` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.85` |
| input | `timeout` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `5000` |
| input | `poll-interval` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `100` |
| input | `settle-duration` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `200` |
| input | `button` | `https://schemas.yotta.dev/types/automation/pointer-button/v1` | `durable` | `durable` | `default-available` | `"left"` |
| input | `hold-duration` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `50` |
| output | `matched` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |
| output | `score` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `center` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `output` | — |
| output | `bounds` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `exec` | `output` | `timeout` |
| `error` | `output` | `failed` |

| Status event | Category |
| --- | --- |
| `automation.template.matched` | `progress` |
| `automation.template.timeout` | `progress` |
| `automation.template.waiting` | `waiting` |

## `https://schemas.yotta.dev/nodes/automation/wait-template-gone`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.waitTemplateGone.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
  - `capture-target`: `https://schemas.yotta.dev/capabilities/automation/capture/v1`; target `target`; risk `sensitive`; consent `once`; operations `capture`, `read-capture`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `template` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| input | `threshold` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.85` |
| input | `timeout` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `5000` |
| input | `poll-interval` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `100` |
| output | `matched` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |
| output | `score` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `center` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `output` | — |
| output | `bounds` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `gone` |
| `exec` | `output` | `timeout` |
| `error` | `output` | `failed` |

| Status event | Category |
| --- | --- |
| `automation.template.matched` | `progress` |
| `automation.template.timeout` | `progress` |
| `automation.template.waiting` | `waiting` |

## `https://schemas.yotta.dev/nodes/automation/wait-stable`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.waitStable.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `capture-target`: `https://schemas.yotta.dev/capabilities/automation/capture/v1`; target `target`; risk `sensitive`; consent `once`; operations `capture`, `read-capture`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| input | `threshold` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.02` |
| input | `timeout` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `5000` |
| input | `poll-interval` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `100` |
| input | `grid-size` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `32` |
| input | `cell-delta` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `12` |
| input | `stable-duration` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `500` |
| output | `changed-ratio` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `mean-difference` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `stable` |
| `exec` | `output` | `timeout` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/wait-change`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.waitChange.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `capture-target`: `https://schemas.yotta.dev/capabilities/automation/capture/v1`; target `target`; risk `sensitive`; consent `once`; operations `capture`, `read-capture`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| input | `threshold` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.02` |
| input | `timeout` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `5000` |
| input | `poll-interval` | `https://schemas.yotta.dev/types/time/duration-milliseconds/v1` | `durable` | `durable` | `default-available` | `100` |
| input | `grid-size` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `32` |
| input | `cell-delta` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `12` |
| output | `changed-ratio` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `mean-difference` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `changed` |
| `exec` | `output` | `timeout` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/activate-window`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/automation/stop-target-app`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.stopTargetApp.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `target`: `https://schemas.yotta.dev/capabilities/automation/app-lifecycle/v1`; target `target`; risk `dangerous`; consent `once`; operations `stop-app`
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

## `https://schemas.yotta.dev/nodes/automation/capture-window`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/automation/play-input-clip`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.playInputClip.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
  - `target`: `https://schemas.yotta.dev/capabilities/automation/playback/v1`; target `target`; risk `dangerous`; consent `once`; operations `play-event`, `release-held`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `clip` | `https://schemas.yotta.dev/types/automation/input-clip/v1` | `durable` | `durable` | `required` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/automation/play-macro`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.automation.playMacro.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
  - `target`: `https://schemas.yotta.dev/capabilities/automation/playback/v1`; target `target`; risk `dangerous`; consent `once`; operations `play-event`, `release-held`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `macro` | `https://schemas.yotta.dev/types/automation/macro/v1` | `durable` | `durable` | `required` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `slot` | `text` | yes | `minLength: 1, maxLength: 128, pattern: ^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/vision/match-template`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.vision.matchTemplate.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `image` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `template` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| input | `threshold` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.8` |
| output | `matched` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |
| output | `score` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `center` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `output` | — |
| output | `bounds` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/vision/find-template-matches`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.vision.findTemplateMatches.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `image` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `template` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| input | `threshold` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `default-available` | `0.8` |
| input | `minimum-distance` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `0` |
| output | `matches` | `list<https://schemas.yotta.dev/types/vision/template-match/v1>` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/vision/compare-images`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.vision.compareImages.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `before` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `after` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| input | `grid-size` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `32` |
| input | `cell-delta` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `12` |
| output | `changed-ratio` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `mean-difference` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/vision/decode-qr`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.vision.decodeQR.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `image` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| output | `codes` | `list<https://schemas.yotta.dev/types/vision/qr-code/v1>` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/vision/analyze-color`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.vision.analyzeColor.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `image` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `range` | `https://schemas.yotta.dev/types/vision/color-range/v1` | `durable` | `durable` | `required` | — |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| output | `pixel-count` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |
| output | `fraction` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `centroid` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/vision/find-color-blobs`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.vision.findColorBlobs.title`
- Availability: `target-required`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities:
  - `blob-read`: `https://schemas.yotta.dev/capabilities/blob/read/v1`; target `blob-store`; risk `low`; consent `none`; operations `read-range`
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `image` | `https://schemas.yotta.dev/types/media/image/v1` | `durable` | `durable` | `required` | — |
| input | `range` | `https://schemas.yotta.dev/types/vision/color-range/v1` | `durable` | `durable` | `required` | — |
| input | `region` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `default-available` | `{"height":1,"unit":"ratio","width":1,"x":0,"y":0}` |
| input | `minimum-area` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `default-available` | `1` |
| output | `blobs` | `list<https://schemas.yotta.dev/types/vision/color-blob/v1>` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/observability/log`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.observability.log.title`
- Availability: `portable`
- Execution: `effect` / `recorded` / cache `none`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `message` | `$T` | `resolved-at-compile` | `durable` | `optional` | — |

| Configuration field | Control | Required | Constraints |
| --- | --- | --- | --- |
| `level` | `select` | no | `enum: "debug", "info", "warn", "error", default hint: "info"` |
| `message` | `text` | no | `maxLength: 16384, default hint: ""` |

| Signal channel | Direction | Port |
| --- | --- | --- |
| `exec` | `input` | `in` |
| `exec` | `output` | `completed` |
| `error` | `output` | `failed` |
Status events: none.

## `https://schemas.yotta.dev/nodes/control/throw`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
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

## `https://schemas.yotta.dev/nodes/structure/break-point`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.structure.breakPoint.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `required` | — |
| output | `unit` | `https://schemas.yotta.dev/types/geometry/point-unit/v1` | `durable` | `durable` | `output` | — |
| output | `x` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `y` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/structure/break-region`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.structure.breakRegion.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `required` | — |
| output | `height` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `unit` | `https://schemas.yotta.dev/types/geometry/point-unit/v1` | `durable` | `durable` | `output` | — |
| output | `width` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `x` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |
| output | `y` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/structure/break-template-match`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.structure.breakTemplateMatch.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/vision/template-match/v1` | `durable` | `durable` | `required` | — |
| output | `bounds` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `output` | — |
| output | `center` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `output` | — |
| output | `score` | `https://schemas.yotta.dev/types/core/number/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/structure/break-qr-code`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.structure.breakQRCode.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/vision/qr-code/v1` | `durable` | `durable` | `required` | — |
| output | `points` | `list<https://schemas.yotta.dev/types/geometry/point/v1>` | `durable` | `durable` | `output` | — |
| output | `text` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/structure/break-color-blob`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.structure.breakColorBlob.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/vision/color-blob/v1` | `durable` | `durable` | `required` | — |
| output | `area` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |
| output | `bounds` | `https://schemas.yotta.dev/types/geometry/region/v1` | `durable` | `durable` | `output` | — |
| output | `center` | `https://schemas.yotta.dev/types/geometry/point/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.

## `https://schemas.yotta.dev/nodes/structure/break-file-metadata`

- Node version: `1.0.0`
- Authoring projection: `sha256:1f2fe64a8df1eaa4e19df9ba874435b67fecf902853cfe9e475d27102c105b35`
- Title key: `node.structure.breakFileMetadata.title`
- Availability: `portable`
- Execution: `pure-data` / `deterministic` / cache `per-run`
- Program instruction: `invoke` `{"kind":"invoke","invoke":{}}`
- Host features: none
- Capabilities: none
- Run state access: none

| Direction | Port | Type | Lifecycle | Carrier | Binding | Default hint |
| --- | --- | --- | --- | --- | --- | --- |
| input | `value` | `https://schemas.yotta.dev/types/filesystem/metadata/v1` | `durable` | `durable` | `required` | — |
| output | `extension` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |
| output | `is-directory` | `https://schemas.yotta.dev/types/core/boolean/v1` | `durable` | `durable` | `output` | — |
| output | `media-type` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |
| output | `modified-unix-millis` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |
| output | `name` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |
| output | `path` | `https://schemas.yotta.dev/types/core/string/v1` | `durable` | `durable` | `output` | — |
| output | `size` | `https://schemas.yotta.dev/types/core/integer/v1` | `durable` | `durable` | `output` | — |

Configuration fields: none.

Exec and Error ports: none.
Status events: none.
