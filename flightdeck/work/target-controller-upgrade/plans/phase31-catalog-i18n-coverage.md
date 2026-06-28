# Phase 31 — Catalog i18n coverage guard

## Goal

Turn node translation completeness into an automated contract. A node can be added or changed only if its catalog label/description and declared input/output pin labels are present in embedded `node-i18n.json`.

## Scope

- Add a Go test in `internal/catalog` that compares `Build()` structure against `BuildWithI18n()`.
- Fail on missing node label/description or missing labels for declared input/output pins.
- Keep hints optional; many pins are self-explanatory and forcing hints would add noise.
- Treat the standard exec input as shared UI language instead of duplicating `In` labels across every node.
- Do not change runtime behavior.

## Verification

- `go test ./internal/catalog -count=1`
- If translations are missing, regenerate or patch the i18n source and embedded catalog before committing.
