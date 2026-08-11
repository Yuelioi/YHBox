# Releasing Yotta

Yotta 4.0 separates an engineering candidate from a public stable release. The current repository license is source-available, not OSI open source.

## Frozen Windows candidate

For a product version's first release candidate, inspect the code-owned inventory and create the two immutable compatibility
floors once:

```powershell
task versions:inventory
task versions:compatibility:freeze
task nodes:compatibility:freeze
```

The freeze tasks are create-only. If a snapshot already exists with different bytes they fail instead of rewriting released
history. Review and commit `contracts/releases/<version>/`, `contracts/node/releases/<version>/`, and
`contracts/catalog/releases/<version>/` with the code that establishes the release.

Then, from a clean worktree, run:

```powershell
task package
```

This requires both compatibility floors for the current `VERSION`, checks every older floor against the current readers and
Catalog, runs the canonical gate, builds the desktop executable plus CLI and isolated workers once, creates the allowlisted
staging tree and deterministic portable archive, then verifies hashes and smoke-tests copies of those exact staged binaries.
No build occurs after staging or smoke.

The resulting `artifacts/Yotta-<version>-windows-amd64.zip` is an unsigned engineering candidate. `artifact-manifest.json` records the source commit, pinned toolchains, exact file set, sizes, hashes, origins and signing state.

## Signing the frozen payload

After configuring a real Authenticode certificate and timestamp service, run:

```powershell
task release:sign-and-stage
```

This command signs the already-built desktop, CLI and worker executables, restages them with `authenticode-signed` manifest state, and repeats the frozen-payload smoke. The signing task has no build dependency and fails if any expected binary is absent.

## Public stable prerequisites

Do not publish `v4.0.0` stable until all of the following are true outside the local build:

- the final version-domain, NodeRef, TypeRef and CapabilityRef snapshots exist for `VERSION`; any retained pre-4.0 reader is
  explicitly classified as a one-time development migration rather than a public support promise;
- the rights holder has made and applied the intended license decision; the current license must never be described as OSI open source;
- canonical repository, module, update and security identities agree;
- main/tag rulesets, protected release environment, non-self approval and at least two real maintainers are configured and exercised;
- Authenticode certificate/timestamp, checksum, SBOM and provenance verification succeed on the final immutable assets;
- Windows installer/upgrade/uninstall smoke runs on a clean host; Linux/macOS remain preview until their native host smoke, signing and permission UX are complete.

The GitHub release workflow currently produces a provenance-attested unsigned candidate. Enabling public stable publication requires an explicit owner-controlled workflow change and the prerequisites above; a tag alone is not a release authorization.
