---
status: active
last_updated: 2026-06-06
when_to_read: before compiling / building / verifying production artifact / 跑 runtime 测试套件 / 真机 smoke
applies_to: [build, compile, task-dev, task-build, wails, exe, vite, bindings, smoke, test-fixture]
when_to_update: 改构建命令 (task dev/build) / wails 配置 / vite 配置 / bindings 生成 / 测试套件入口时
---

# Build Playbook

编译 / 验证产物时**前置**读这份.

- **开发**: `task dev` — vite (port 9245) + wails3 webview 热重载. 改前端实时刷, 改 Go 要重启.
- **打包**: `task build` — 内部先 `vite build` 嵌入 dist, 再 `go build -tags production -ldflags="-w -s -H windowsgui" -o bin/YHBox.exe .`
- **仅语法 check**: `go build ./...` 可用 (不产 exe), 但产 exe 一定走 task.

**永远别裸 `go build -o YHBox.exe`** — 缺 vite build → frontend/dist 旧/空 → 启动空白; 缺 `wails3 generate syso` → 没 icon + 缺 manifest (admin 提权检测不对).

Taskfile: 顶层 `Taskfile.yml` → `build/Taskfile.yml` (common) + `build/windows/Taskfile.yml`. 仅留 windows + common; `build/darwin/` 空目录是为了 `wails3 generate icons` 不 fail.

## bindings（gitignore 生成物）

`frontend/bindings/` 是 wails 生成物、gitignore. 改 Go 导出符号 / 路由后, 下次 `task dev` / `task build` 自动 regenerate; 手动改名要同步 rename + 内容替换 (vue-tsc 过) 再 build, 否则前端引用旧名.

## 测试留意（预存非回归失败）

跑 `internal/services/container/runtime/` 套件前先 `task build` 填 `bin/data/.../subgraphs/*.json` fixture, 否则 `TestApplyDirection_*` / `TestWatchdog_*` 缺 fixture 失败. `TestScanSubgraphDependencies_*` 也是预存失败、非回归 (撞到别当成自己改坏的).

## 运行 / smoke 留意

- **校准 / HUD 是 AlwaysOnTop**: 独占全屏游戏可能盖不住 (Windows 层限制) → 用窗口化 / 无边框全屏.
- **通道 B（worker 事件校验失败本地化）没真机端到端 smoke**: 要造得手改磁盘 `bin/data/containers/<id>/container.json` 删 WindowTarget 再走热键; 走事件通道 `d.Error` 是对象、不受 [[2026-06-02-wails-dev-fetch-transport-flattens-error]] 影响, 有 Go 单测背书, 按需补.
