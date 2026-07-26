# ADR-0003: Separate durable blobs from Run-scoped resources

## Status

Accepted for Yotta 3.1.

## Context

Workflow values include large immutable bytes, incremental streams, and host/plugin objects such as files, windows, processes, and sessions. Treating all of them as strings or Go objects would leak storage paths and platform handles, bypass authorization, make cleanup nondeterministic, and create incompatible builtin/Wasm/Process semantics.

## Decision

- A Blob Reference is durable and portable: canonical parameter-free media type, raw-byte SHA-256 digest, and exact byte size. The digest addresses bytes; media type is interpretation metadata. Storage paths are private to `internal/blob`.
- Blob Store writes are immutable, quota-bounded, uniquely staged, atomically committed, and integrity-checked before read or dedup reuse. Asset reference commit excludes GC; callers cannot obtain the asset store's raw blob implementation.
- A Resource Reference is an ephemeral, Run-scoped opaque 256-bit token resolved only by `internal/resource`. Each lease is bound to Program hash, capability-plan digest, principal, plugin instance, session, Run, invocation, provider, target, kind, operation set, and UTC expiry. Borrow may only narrow operations and lifetime inside that exact authority lineage.
- Resource providers are fixed at Broker composition. Open, borrow, and every call are independently authorized before provider side effects. Raw provider objects never enter a Value Envelope. Last drop, expiry sweep, Run revocation, or Broker shutdown cancels active calls and closes the provider object exactly once.
- A Stream is only a Resource Broker provider. It has explicit bounded capacity and chunk budget. Send blocks under backpressure; finish drains queued chunks then yields EOF; cancel discards queued chunks and wakes blocked producers/consumers.
- Value Envelope 3.1 is a sealed union of inline JSON, Blob Reference, Stream Reference, or Resource Reference. Inline/blob envelopes may be durable. Stream/resource envelopes are runtime transport only and must be redacted from durable Program, trace, log, clipboard, and cache artifacts.
- Program literals cannot contain stream/resource references. Blob/inline/stream conversions are explicit nodes because they allocate, perform I/O, can fail, and consume quotas. A resource handle has no generic conversion to durable data; only a capability-specific operation may produce a new inline or blob value.

## Consequences

- Old string SHA fields and `asset.BlobStore` are deleted; asset schema version 2 is exact, unknown/old/corrupt records fail startup, and every durable blob reference is integrity-checked during preload.
- Same bytes can be deduplicated across media types while every consumer still validates declared size, digest, and expected media type.
- Replay may reuse a retained Blob Reference. Stream and Resource References are never replayed; replay must reopen from a durable recipe under a new Run Grant.
- Provider cleanup is a required conformance behavior for builtin, Wasm, and Process hosts.
