# ⚠ Go coverage instrumentation 会被源码 UTF-8 BOM 阻断
2026-07-10 曾确认 `internal/services/container/rewriter.go` 的字节 0 是 `EF BB BF`。普通 `go test ./...` 通过，但 `go test -cover ./...` 在构建 container test 时失败：

```text
internal/services/container/rewriter.go:1:1: invalid BOM in the middle of the file
```

根因是 coverage 对源码插桩后，原本只允许位于源文件起点的 BOM 不再处于解析器认可的位置。该文件已改为 UTF-8 without BOM，`go test -cover ./...` 已恢复；若症状复现，检查字节前缀并修正编码，不要豁免 coverage。
