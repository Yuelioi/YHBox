# ⚠ Vite build 与 Go embed 测试并行会制造旧 chunk 假红
SUMMARY: `frontend/dist` 被 Go `embed` 读取时，不要并行运行会清空并重建该目录的 Vite production build；应先完成前端构建，再顺序运行 Go 测试。
READ WHEN: 当 `go test .` 报嵌入的 Vite manifest/chunk 不存在，而同一时间正在运行 `pnpm build` / `vite build` 时；编排并行验证任务前。

---

Vite production build 会先清理 `frontend/dist`，再写入新的 manifest 和带 hash 的 chunk。根包的 Go 源码通过 `embed` 编译该目录；若 `go test .` 恰好在清理与写回之间读取，或读取到新 manifest 配旧 chunk，就会出现文件不存在的假失败。

这不是源码回归，也不要通过保留旧 chunk 规避。验证顺序应为：

1. `cd frontend && pnpm build`
2. 回到仓库根目录运行 `go test ./... -count=1`

不触碰 `frontend/dist` 的测试仍可并行；任何会重建 dist 的命令必须与读取 Go embed 的编译/测试串行。
