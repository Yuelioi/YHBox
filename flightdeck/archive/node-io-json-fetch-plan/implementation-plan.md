# IO / JSON / Fetch Nodes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. User instruction overrides the default Superpowers commit cadence: do not commit unless the user explicitly asks.

**Goal:** Add the first practical external-data node set: read local text/JSON files, parse/query JSON, and perform HTTP fetches whose inputs can come from previous nodes.

**Architecture:** Keep side-effecting operations as `IO` runnable nodes and data transforms as `PureFunc` evaluator nodes. Treat `JSON` as any valid JSON value (`object`, `array`, scalar, or `null`) while preserving the existing object helper for old nodes. Use ordinary node specs so the existing backend catalog and frontend node renderer pick them up automatically.

**Tech Stack:** Go node framework (`internal/node`, `internal/nodes/io`, `internal/nodes/purefunc`), Go stdlib `encoding/json`, `net/http`, `os`, `golang.org/x/text` for GBK fallback, existing Vue i18n extraction (`frontend/src/i18n/*.ts`, `pnpm gen:node-i18n`).

---

## Scope

Implement now:

- `ReadTextFile` (`IO`): read a local text file into `Done.Text`.
- `ReadJsonFile` (`IO`): read and decode a local JSON file into `Done.JSON`.
- `ParseJSON` (`PureFunc`): parse a string into a JSON value.
- `ToJSON` (`PureFunc`): serialize any value to JSON text.
- `JsonPath` (`PureFunc`): extract from a JSON value using a deliberately small path grammar: `$`, `.field`, `[index]`, `[*]`.
- `Fetch` (`IO`): perform HTTP/HTTPS requests with method, URL, headers, cookies, body, timeout, redirect, and status-fail controls.

Defer:

- `WriteTextFile`, `WriteJsonFile`, `WatchFile`, cURL import, HTML extraction, advanced JSON merge/map/filter.
- A new `Object/Map` type. `JSON` carries external structured data for now.

## File Map

- Modify `internal/node/types.go`: change `JSON` type metadata from `map[string]any` to `any`.
- Modify `internal/node/interfaces.go` and `internal/node/inputs.go`: add `JSONValue(name string) any`; keep `JSON(name) map[string]any`.
- Modify `internal/node/types_test.go`: cover JSON type metadata and helper behavior.
- Create `internal/nodes/io/file_read.go`: `ReadTextFile`, `ReadJsonFile`, path resolution, text decoding.
- Create `internal/nodes/io/file_read_test.go`: temp-file tests for text, JSON arrays, JSON parse errors, missing file.
- Create `internal/nodes/io/fetch.go`: HTTP request node, response parsing, fail-output helper.
- Create `internal/nodes/io/fetch_test.go`: `httptest.Server` coverage for JSON response, headers/cookies/body, timeout/status fail.
- Create `internal/nodes/purefunc/json.go`: `ParseJSON`, `ToJSON`, `JsonPath`, path parser/evaluator.
- Create `internal/nodes/purefunc/json_test.go`: JSON parsing, serialization, object/array/scalar path extraction, wildcard extraction, invalid path behavior.
- Modify `internal/nodes/purefunc/purefunc.go`: register the three JSON pure functions.
- Modify `frontend/src/i18n/zh.ts` and `frontend/src/i18n/en.ts`: add node labels, pin labels, output labels, dropdown option labels.
- Regenerate `internal/catalog/node-i18n.json` with `cd frontend && pnpm gen:node-i18n`.
- Update `flightdeck/cockpit.md` after implementation to point at this work package.

## Task 1: JSON Type Semantics

**Files:**
- Modify: `internal/node/types.go`
- Modify: `internal/node/interfaces.go`
- Modify: `internal/node/inputs.go`
- Modify: `internal/node/types_test.go`

- [ ] **Step 1: Write failing tests**

Add tests proving `JSON` is metadata-described as `any`, `JSONValue` can return arrays/scalars, and `JSON` still returns only object maps for existing call sites.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/node -run "TestJSONTypeRegisteredAsAny|TestInputsJSONValue" -count=1`

Expected: fail because `JSONValue` does not exist and/or `GoType` is still `map[string]any`.

- [ ] **Step 3: Implement**

Change built-in type registration:

```go
{Tag: "JSON", GoType: "any", WidgetKind: "json", Color: "#9ca3af"},
```

Add to `Inputs`:

```go
JSONValue(name string) any
```

Add to `inputsImpl`:

```go
func (i *inputsImpl) JSONValue(name string) any { return i.merged[name] }
```

- [ ] **Step 4: Verify GREEN**

Run the same `go test` command and confirm pass.

## Task 2: File Read Nodes

**Files:**
- Create: `internal/nodes/io/file_read.go`
- Create: `internal/nodes/io/file_read_test.go`

- [ ] **Step 1: Write failing tests**

Cover:

- `ReadTextFile` reads UTF-8 text from absolute temp path and outputs `Text`, `Size`, `ModTimeMs`.
- `ReadTextFile` decodes GBK when `Encoding=gbk`.
- `ReadJsonFile` decodes a JSON array and proves `Done.JSON` can be `[]any`.
- Missing file and invalid JSON produce an error result that can route through `Fail`.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/nodes/io -run "TestReadTextFile|TestReadJsonFile" -count=1`

Expected: fail because node types do not exist.

- [ ] **Step 3: Implement**

Register both nodes in `init()`. Use absolute paths as-is; resolve relative paths against `YOTTA_DATA_DIR` or `bin/data`. Limit reads to `MaxBytes` default `1048576`, with `0` meaning default. Support `Encoding` values `utf-8`, `gbk`, `auto`; `auto` uses UTF-8 if valid, otherwise GBK.

Specs:

- Inputs: `In Exec`, `Path String`, `Encoding String`, `MaxBytes Integer`.
- `ReadTextFile` outputs `Done` data `Text String`, `Size Integer`, `ModTimeMs Integer`; `Fail` with `Error`, `Code`.
- `ReadJsonFile` outputs `Done` data `JSON JSON`, `Text String`, `Size Integer`, `ModTimeMs Integer`; `Fail` with `Error`, `Code`.

- [ ] **Step 4: Verify GREEN**

Run the same `go test` command and confirm pass.

## Task 3: JSON Pure Functions

**Files:**
- Create: `internal/nodes/purefunc/json.go`
- Create: `internal/nodes/purefunc/json_test.go`
- Modify: `internal/nodes/purefunc/purefunc.go`

- [ ] **Step 1: Write failing tests**

Cover:

- `ParseJSON` returns `map[string]any` for objects and `[]any` for arrays.
- `ParseJSON` returns `nil` on invalid JSON and logs a warning.
- `ToJSON` serializes maps/lists/scalars and returns `""` for marshal errors.
- `JsonPath` supports `$`, `$.user.name`, `$.items[0].id`, and `$.items[*].id`.
- `JsonPath` returns `nil` for missing fields, bad indexes, non-container traversal, and invalid path syntax.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/nodes/purefunc -run "TestParseJSON|TestToJSON|TestJsonPath" -count=1`

Expected: fail because node types do not exist.

- [ ] **Step 3: Implement**

Register `&ParseJSON{}`, `&ToJSON{}`, `&JsonPath{}` in `purefunc.go`.

Specs:

- `ParseJSON`: input `Text String`, output `Result JSON`.
- `ToJSON`: input `Value *`, output `Result String`.
- `JsonPath`: inputs `JSON JSON`, `Path String`, output `Result *`.

Path grammar is intentionally small and deterministic:

```text
$                     whole value
$.name                object field
$.items[0]            array index
$.items[*].name       map each array item through the following path
```

Do not import a full JSONPath dependency in this first slice.

- [ ] **Step 4: Verify GREEN**

Run the same `go test` command and confirm pass.

## Task 4: Fetch Node

**Files:**
- Create: `internal/nodes/io/fetch.go`
- Create: `internal/nodes/io/fetch_test.go`

- [ ] **Step 1: Write failing tests**

Cover:

- `Fetch` GET decodes JSON response into `Done.JSON`, preserves `Done.Body`, `Done.StatusCode`, `Done.Headers`.
- Headers and cookies can be passed from JSON/string inputs and reach the server.
- POST `BodyMode=json` marshals any JSON/body value and sets `Content-Type: application/json` if missing.
- `FailOnStatus=true` routes HTTP status >= 400 to `Fail` with `StatusCode` and `Body`.
- Timeout routes to `Fail` with code `timeout`.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/nodes/io -run TestFetch -count=1`

Expected: fail because `Fetch` does not exist.

- [ ] **Step 3: Implement**

Spec inputs:

- `In Exec`
- `Method String` dropdown: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`
- `URL String`
- `Headers JSON`
- `Cookies String`
- `Body *`
- `BodyMode String` dropdown: `text`, `json`, `none`
- `TimeoutMs Integer` default `10000`
- `FollowRedirects Bool` default `true`
- `FailOnStatus Bool` default `true`
- `MaxBytes Integer` default `1048576`

Spec outputs:

- `Done Exec` data: `StatusCode Integer`, `Body String`, `JSON JSON` optional, `Headers JSON`, `DurationMs Integer`.
- `Fail Exec` semantic `error` data: `Error String`, `Code String`, `StatusCode Integer` optional, `Body String` optional.

Implementation notes:

- Allow only `http://` and `https://`.
- Build headers from `map[string]any`; values stringify with `node.FormatValue`.
- Copy `Cookies` string into the `Cookie` header if non-empty and no explicit `Cookie` header was set.
- Read at most `MaxBytes + 1`; if larger, fail with code `error`.
- Parse response JSON only when body is non-empty and content type contains `json`; parse failure leaves `JSON=nil` and still routes `Done`.

- [ ] **Step 4: Verify GREEN**

Run the same `go test` command and confirm pass.

## Task 5: i18n and Catalog

**Files:**
- Modify: `frontend/src/i18n/zh.ts`
- Modify: `frontend/src/i18n/en.ts`
- Generate: `internal/catalog/node-i18n.json`

- [ ] **Step 1: Add i18n**

Add `node.ReadTextFile`, `node.ReadJsonFile`, `node.ParseJSON`, `node.ToJSON`, `node.JsonPath`, `node.Fetch` in both languages. Include every input label, dropdown option label, exec output label, and data field hint.

- [ ] **Step 2: Generate catalog**

Run: `cd frontend && pnpm gen:node-i18n`

- [ ] **Step 3: Verify catalog/i18n**

Run:

```powershell
go test ./internal/catalog -count=1
cd frontend
pnpm i18n:check
```

## Task 6: Full Verification and Flightdeck Update

**Files:**
- Modify: `flightdeck/cockpit.md`

- [ ] **Step 1: Run focused backend verification**

Run:

```powershell
go build ./...
go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... -count=1
```

- [ ] **Step 2: Run frontend verification**

Run:

```powershell
cd frontend
pnpm typecheck
pnpm i18n:check
```

- [ ] **Step 3: Run project build if focused checks pass**

Run: `task build`

- [ ] **Step 4: Update Flightdeck**

Record completion and any verification gaps in `flightdeck/cockpit.md`. Do not commit.
