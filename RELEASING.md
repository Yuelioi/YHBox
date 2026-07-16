# Releasing Yotta

Yotta 3.1 separates an engineering candidate from a public stable release. The current repository license is source-available, not OSI open source.

## Frozen Windows candidate

From a clean worktree, run:

```powershell
task package
```

This runs the canonical gate, builds the desktop executable plus CLI and isolated workers once, creates the allowlisted staging tree and deterministic portable archive, then verifies hashes and smoke-tests copies of those exact staged binaries. No build occurs after staging or smoke.

The resulting `artifacts/Yotta-<version>-windows-amd64.zip` is an unsigned engineering candidate. `artifact-manifest.json` records the source commit, pinned toolchains, exact file set, sizes, hashes, origins and signing state.

## Signing the frozen payload

After configuring a real Authenticode certificate and timestamp service, run:

```powershell
task release:sign-and-stage
```

This command signs the already-built desktop, CLI and worker executables, restages them with `authenticode-signed` manifest state, and repeats the frozen-payload smoke. The signing task has no build dependency and fails if any expected binary is absent.

## Public stable prerequisites

Do not publish `v3.1.0` stable until all of the following are true outside the local build:

- the rights holder has made and applied the intended license decision; the current license must never be described as OSI open source;
- canonical repository, module, update and security identities agree;
- main/tag rulesets, protected release environment, non-self approval and at least two real maintainers are configured and exercised;
- Authenticode certificate/timestamp, checksum, SBOM and provenance verification succeed on the final immutable assets;
- Windows installer/upgrade/uninstall smoke runs on a clean host; Linux/macOS remain preview until their native host smoke, signing and permission UX are complete.

The GitHub release workflow currently produces a provenance-attested unsigned candidate. Enabling public stable publication requires an explicit owner-controlled workflow change and the prerequisites above; a tag alone is not a release authorization.
