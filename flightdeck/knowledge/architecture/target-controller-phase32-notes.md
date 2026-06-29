# Target Controller Upgrade — Phase 32 Notes

## Completed

- Added `internal/services/nodeoptions` as the app-level async dropdown source registrar.
- Registered the previously missing `clipIDs` source for `PlayClip.ClipID`.
- Registered the previously missing `subgraphIDs` source for `Subgraph.SubgraphID` and `CollapsedNode.SubgraphID`.
- Added `NodeService.RegisteredAsyncSources()` for diagnostics and coverage tests.
- Added a source coverage test that scans all node specs and fails if an `async-dropdown` declares an unregistered source.
- Wired `main.go` to register asset/subgraph async sources after `assetSvc` and `sgSvc` are constructed.

## Verification

- `go test ./internal/services/nodeoptions ./internal/node ./internal/nodes/io ./internal/nodes/system -count=1`
- `go test . -count=1`
- `go test ./internal/services/androidadb -count=1`
- `go test ./internal/services/browsercdp -count=1`
- `go test ./internal/automation/controller -count=1`
- `go test ./internal/services/container/runtime -count=1`

## Note

The combined command including `./internal/services/container/runtime` timed out at 124 seconds because the runtime package alone took about 125 seconds in this run. The package passed when run separately with a longer timeout.

## Next Risk

Widget props still rely on loosely typed maps after `MarshalProps`. The next useful guard is validating widget kind/props shape across all node specs so typos like `asyncsource` or invalid `applyMeta` targets fail at test time.
