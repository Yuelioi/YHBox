# Minimal Wasm example

This zero-output node demonstrates the complete stable low-level ABI: exported memory, `yotta_alloc`, `yotta_run`, and the sole `yotta_plugin_v1.exchange` import. It emits the generated canonical successful-result conformance vector.

Compile it as a freestanding `wasm32-unknown-unknown` module with exported memory. The runner does not instantiate WASI; filesystem, sockets, clocks, environment, process, and random imports are intentionally unavailable.

