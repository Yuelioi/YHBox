# Target Controller Upgrade — Phase 36 Notes

## Completed

- Made Vitest hermetic against Wails runtime side effects and external fetches.
- Skipped the Wails Vite plugin in test mode.
- Added a Vitest setup file that replaces `fetch` with a deterministic empty response.
- Added a Vitest-only `@wailsio/runtime` alias that supplies local `Events`, `Window`, `Browser`, `Call`, `CancellablePromise`, and `Create` stubs for app code and generated bindings.
- Let the production build regenerate `frontend/components.d.ts`, adding the missing `ActionTraceDrawer` global component declaration.

## Verification

- `pnpm test`
- `pnpm vue-tsc --noEmit`
- `pnpm build`
- `pnpm i18n:check`

## Result

Vitest now passes without the previous `/wails/custom.js`, external HTTPS timeout, or happy-dom fetch abort noise.

## Next Risk

Frontend test gates are cleaner. Continue with contract coverage around dynamic target dropdowns, metadata apply behavior, and inspector value synchronization.
