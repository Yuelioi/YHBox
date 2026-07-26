# M4c — Offline Pack container

## Journey

用户需要离线搬运一个可完整再分发的安装集合时，`.yotta-offline-pack` 保存 M4a canonical
Installation Plan 和该计划锁定的 Workflow/Node Package 原始 artifact bytes。接收端可在不联网时确认
所有字节的 digest/size 完整性，但容器本身不签名、不自举 publisher trust、不写本机安装状态，也不授予
Workflow execution consent。

## Boundary

- `offlinepack` 只拥有 deterministic ZIP transport；`installationplan` 仍是 artifact identity 唯一清单。
- artifact entry 只按 raw SHA-256 digest 寻址；相同 digest 可去重，但 size/media type 冲突必须拒绝。
- Writer 从调用方提供的只读 source 逐项复制并重新哈希；Inspect 严格拒绝 missing、extra、tampered、
  duplicate、traversal、symlink、encrypted 与超预算 entry。
- Inspect 只返回原 Installation Plan。后续导入协调器必须分别调用 Workflow proof verifier 与
  Node Package signature/trust/install Module，不能把 pack 完整性当成 authority。

## Verification

- 相同 Plan 与 artifact bytes 产生完全相同 pack bytes，并可重开为同一 Plan digest。
- Writer 拒绝 source bytes 与 descriptor 不符；Inspect 拒绝缺失、额外和篡改 artifact。
- `go test ./internal/offlinepack -count=1` 与增量 `task check` 通过。

## Status

Finished.
