# Target Controller Upgrade — Phase 37 Notes

## Completed

- Added frontend contract coverage for async dropdown behavior:
  - `PinInput` calls `NodeService.AsyncOptions` with node ID, spec kind, async source, and current inputs.
  - async selected item values normalize from Nuxt UI item objects.
  - selected option metadata resolves by string-equivalent value.
  - Inspector metadata application writes selected value and non-empty mapped metadata into sibling literals.
- Extracted small pure helpers:
  - `inline/asyncDropdown.ts`
  - `asyncOptionMeta.ts`

## Verification

- `pnpm vitest run src/components/containers/asyncOptionMeta.test.ts src/components/containers/inline/asyncDropdown.test.ts src/components/containers/inline/PinInput.async-dropdown.test.ts`
- `pnpm vue-tsc --noEmit`
- `pnpm test`
- `pnpm build`

## Result

Frontend async dropdowns now have behavior-level tests instead of only adapter mapping coverage.

## Next Risk

Continue with target/runtime contract coverage: active target transitions, controller factory errors, and screenshot/click coordinate assumptions.
