---
status: active
summary: Phase 1 存储平铺 — assets 按类拆 templates//clips/ + blobs 上提 + schedules 拍平 + 死目录清除 + 启动防呆闸 + schemaVersion 字段
last_updated: 2026-06-12
implements: specs/2026-06-12-data-layout-flatten-subgraph-globalize.md
---

# Plan P1: 存储平铺

## Progress

current: 未开工。

**与 P2 的衔接**: P1/P2 代码连续推进, **旧数据迁移只在 P2 末尾跑一次**(迁移脚本一把覆盖两阶段布局变更)。P1 阶段验证 = go test + 空 data 目录冒烟启动(防呆闸只认旧布局精确标记, 空目录不拦); 真机验证统一压到 P2 末。

## Tasks

1. **asset.Store 按 kind 分目录** `internal/services/asset/store.go` + `model.go`:
   - 写入按 `rec.Kind` 落 `<root>/templates/<guid>.json` 或 `<root>/clips/<guid>.json`; 加载扫两个目录, 内存索引仍按 GUID 统一(Get/List/Delete 接口不变, Delete 按内存中 rec.Kind 定位文件)。
   - 加载时校验 kind↔目录一致, 不一致拒载报错(耐久性 #4: kind 权威, 目录是索引)。
   - `AssetRecord` 加 `SchemaVersion int json:"schemaVersion"`(写 1, 无读取分支; 读取契约: >1 拒载)。
   - `gc.go` 逻辑不动 — 确认 live set 仍由 store 内存记录提供(扫描入口随 store 加载路径自然切换, 别留旧 assets/records 路径残读)。
2. **blobstore 上提** `internal/services/asset/blobstore.go` 不动, `main.go` 接线根路径 `data/assets/blobs` → `data/blobs`。
3. **schedule.Store 拍平** `internal/services/schedule/store.go`: `schedules/<id>/schedule.json` → `schedules/<id>.json`(读/写/删/列举四处); model 加 schemaVersion。
4. **死目录清除** `internal/services/container/store.go:176-180`: Save 的 mkdir 循环只留 `subgraphs`(P2 再删它); `store_test.go:102` templates 断言删除。
5. **启动防呆闸 + 布局目录** `main.go`:
   - `ensureV2DataLayout` 改建: containers/ templates/ clips/ blobs/ schedules/(data/subgraphs 留给 P2)。
   - 防呆闸: `data/assets/records/` 或 `data/library/subgraphs/` 存在 → log fatal + 对话框提示"旧数据布局, 先跑迁移脚本"(精确路径防全新安装误拒; 迁移期过后此闸可删)。
6. **main.go 接线**: asset/schedule store 根路径换新; `library.NewStore` 不动(P2 整删)。
7. **验证**: `go build ./...` + `go test ./...` 全绿(asset/schedule/container store 相关测试改 fixture 路径); 删 bin/data 用空目录跑一次 `wails3 dev` 编译级冒烟(我能做的部分: build; 启动行为靠 P2 末真机)。
