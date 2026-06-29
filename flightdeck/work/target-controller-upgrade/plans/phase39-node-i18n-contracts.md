# Phase 39 — Node I18n Contract Guards

## Problem

The UI renders node labels, parameter labels, and dropdown option labels from `frontend/src/i18n/*.ts`, while backend catalog text is generated into `internal/catalog/node-i18n.json`.

Existing guards were incomplete:

- the catalog test package did not import every node package, so newer AI/Image/Window nodes could escape i18n checks;
- pin labels were checked, but static dropdown option labels were not;
- the generated JSON dropped option labels, so Go-side drift tests could not verify them.

## Scope

- Extend node i18n extraction to include input dropdown `option` labels.
- Make catalog tests load every node package.
- Add a Go drift guard for every static dropdown option label.
- Regenerate `internal/catalog/node-i18n.json` and fill missing zh/en node text revealed by the stricter guard.

## Non-goals

- No copy rewrite beyond missing labels.
- No frontend visual changes.
