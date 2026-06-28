# Target Controller Upgrade — Phase 44 Notes

## Completed

- Added `[refs]` to `pnpm i18n:check`, covering static `t('...')` / `te('...')` key references.
- Corrected the checker's TS module loading so it validates the actual default-exported message objects.
- Added 11 missing zh/en keys for variable promotion and delete confirmation UI.

## Verification

- `pnpm i18n:check`
- `pnpm vue-tsc --noEmit`
- `pnpm test`

## Result

Static frontend i18n key drift now fails CI-style validation instead of rendering raw key strings in the UI. Node-specific dynamic keys remain covered by the catalog/node i18n guard.
