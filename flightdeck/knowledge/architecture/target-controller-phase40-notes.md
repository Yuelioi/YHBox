# Target Controller Upgrade — Phase 40 Notes

## Completed

- Added `internal/nodes/all` as the single full built-in node registration import.
- Replaced full-registration import lists in catalog/spec/node-option/MCP/CLI paths.

## Verification

- `go test ./internal/node ./internal/catalog ./internal/services/nodeoptions ./internal/services/mcpserver -count=1`
- `go test ./cmd/node-catalog -count=1`

## Result

Future node packages now have one obvious registration entry to update, reducing the chance that catalog/i18n/MCP guards miss a new node.
