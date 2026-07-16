# Yotta plugin node reference

A package contributes immutable Node Contracts. Node identity is the stable nodeTypeId plus an explicit SemVer version and semanticDigest. The runtime never derives a node name from the Yotta application version.

Each implementation is pinned by packageId, manifest artifactDigest, ABI kind/version, and entrypoint. Process v1 payloads are Windows PE executables; WIT v1 payloads are application/wasm modules executed by the trusted runner without WASI.

Guests receive canonical Value Envelopes and must return the exact declared output ports. Resource, state, entropy, wait, status, and action operations cross the mediated protocol; filesystem paths, credentials, native handles, frontend JavaScript, Vue, and DOM access are not plugin APIs.
