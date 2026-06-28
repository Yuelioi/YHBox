# Target Controller Upgrade — Phase 39 Notes

## Completed

- Tightened node i18n drift coverage:
  - catalog tests now import all node packages, including AI, Image, and Window;
  - generated `node-i18n.json` carries dropdown option labels;
  - catalog tests verify every static dropdown option has translated text.

## Verification

- `pnpm gen:node-i18n`
- `go test ./internal/catalog -count=1`
- `go test ./internal/node ./internal/catalog -count=1`
- `pnpm i18n:check`
- `pnpm vue-tsc --noEmit`
- `pnpm test`
- `pnpm build`

## Result

Node parameter translation coverage is now guardable instead of relying on manual visual inspection.
