---
kind: checklist
summary: "Always use `github.com/yottaapp/yotta` as the canonical Go module/import path; Wails bindings mirror this path under `frontend/bindings/github.com/yottaapp/yotta`."
activation: action
read_when: "before changing go.mod module identity, Go imports, Wails binding imports, repository ownership/name, or generating bindings after a module move."
---
# Go module identity checklist
Yotta 的 canonical repository URL 是 `https://github.com/yottaapp/yotta`，Go module path 是精确小写的 `github.com/yottaapp/yotta`。内部 Go imports 使用完整 module path，不得恢复临时的 `yotta/...` 或旧 owner/case。

Wails 按 Go package import path 生成目录，因此前端绑定引用必须使用：

```text
@bindings/github.com/yottaapp/yotta/internal/...
```

module identity 变更后必须重新运行 `task common:generate:bindings`，并验证 `go list -m`、Go 全量测试、前端 typecheck 与 Vitest。CI 的 module identity step 防止 go.mod 漂回临时路径。
