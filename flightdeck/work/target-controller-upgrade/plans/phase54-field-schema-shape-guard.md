# Phase 54 — Field Schema Shape Guard

## Problem

Structured input schemas are recursive and are passed to the frontend renderer. Existing tests covered widget props, but not malformed `FieldSchema` trees such as arrays without item schemas, nested fields without keys, or unknown schema types.

## Scope

- Add a recursive Go spec consistency test for `InputSpec.Schema`.
- Validate schema type names, nested fields, array items, and enum option values.
- Keep the guard structural only; business validation stays in node validators.

## Non-goals

- No frontend renderer change.
- No semantic range/pattern validation.
