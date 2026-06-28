# Target Controller Upgrade — Phase 35 Notes

## Completed

- Added a test-only fixture timing cap for fishing-v2 runtime state tests.
- Applied it after JSON load for slow state and helper subgraphs.
- Kept production `testdata/fishing-v2` fixtures unchanged.

## Verification

- `go test ./internal/services/container/runtime -run "TestStateBUYBAIT|TestStateSHOPSELL|TestStateCHANGEBAIT|TestStateRESULT|TestStateFISHING_BarMissing_Result|TestTryHookF_Exhausted|TestStateSETUP_NotFoundIdle" -count=1`
- `go test ./internal/services/container/runtime -count=1`
- `go test ./internal/services/container/runtime -json -count=1`

## Result

The full runtime package now passes in about 4.4 seconds in this workspace. The previously slow targeted state-machine set passes in about 2.5 seconds.

## Next Risk

Runtime state tests are now practical for frequent use. Continue with broader quality hardening: target/runtime contract tests, node registry consistency, and frontend inspector behavior around dynamic targets.
