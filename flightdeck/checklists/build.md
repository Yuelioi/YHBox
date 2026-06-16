---
status: active
last_updated: 2026-06-17
when_to_read: before compiling / building / verifying production artifact / 跑 runtime 测试套件 / 跑前端 vitest / 真机 smoke
applies_to: [build, compile, task-dev, task-build, wails, exe, vite, bindings, smoke, test-fixture, vitest, frontend-test]
when_to_update: 改构建命令 (task dev/build) / wails 配置 / vite 配置 / bindings 生成 / 测试套件入口 / 前端测试跑法时
---

# Build Playbook

编译 / 验证产物时**前置**读这份.

- **frontend 包管理只用 pnpm** (有 pnpm-lock.yaml): `npm install` 撞 `Cannot read properties of null (reading 'matches')` — npm 的 arborist 解析不了 node_modules/.pnpm 布局, 不是网络/缓存问题, 换 `pnpm add` 即好.

- **开发**: `task dev` — vite (port 9245) + wails3 webview 热重载. 改前端实时刷, 改 Go 要重启.
- **打包**: `task build` — 内部先 `vite build` 嵌入 dist, 再 `go build -tags production -ldflags="-w -s -H windowsgui" -o bin/YHBox.exe .`
- **仅语法 check**: `go build ./...` 可用 (不产 exe), 但产 exe 一定走 task.

**永远别裸 `go build -o YHBox.exe`** — 缺 vite build → frontend/dist 旧/空 → 启动空白; 缺 `wails3 generate syso` → 没 icon + 缺 manifest (admin 提权检测不对).

Taskfile: 顶层 `Taskfile.yml` → `build/Taskfile.yml` (common) + `build/windows/Taskfile.yml`. 仅留 windows + common; `build/darwin/` 空目录是为了 `wails3 generate icons` 不 fail.

## bindings（gitignore 生成物）

`frontend/bindings/` 是 wails 生成物、gitignore. 改 Go 导出符号 / 路由后, 下次 `task dev` / `task build` 自动 regenerate; 手动改名要同步 rename + 内容替换 (vue-tsc 过) 再 build, 否则前端引用旧名.

## 测试留意（预存非回归失败）

runtime 套件 fixture 已入仓 (2026-06-12 子图全局化时迁): `internal/services/container/runtime/testdata/fishing-v2/` + `testdata/templates/`, 不再读 bin/data (用户数据已重铸完整 uuid, 按名读必死)。`TestApplyDirection_*` / `TestWatchdog_*` 仍红 = `apply_direction.json` / `watchdog_check.json` 两个 fixture 本机从来没有 — 从有它们的机器补进 testdata/fishing-v2/subgraphs/ 即愈。`TestScanSubgraphDependencies_*` 也是预存失败、非回归 (撞到别当成自己改坏的).

`TestFishingV2Main_StateCycleSmoke` 本机预存红 (2026-06-11 git stash 实证改动前同样 `clicks=0 finalState=IDLE`): mock 模板没命中 → state_IDLE 一直走 NotFound 兜底分支, 疑与本机 fishing-v2 数据迁 GUID 资产后 mock 名解析有关, 待单独排查.

## 前端单测 (vitest)

- 前端**有 vitest 套件** (配置在 `vite.config.ts` 的 `test` 块 —— **不是**单独 `vitest.config.ts`; 测试文件 `src/**/*.{test,spec}.ts`, 已有 useEditorSave / useVarMutations / scriptCompletions / ytConsole 等)。
- 跑: `cd frontend && ./node_modules/.bin/vitest run [路径]`。**本环境 `pnpm -C frontend test` 会炸 `ENOENT lstat …/frontend/frontend`** (pnpm `-C` + cwd 下项目插件路径解析的坑) —— 直接调 vitest 二进制即好, 别去动 vite.config / 另建 vitest.config (踩过这个误诊坑)。
- 全量跑里有个**摸网络的预存测试**, 无网环境 (sandbox) 会 `ETIMEDOUT` 中断 —— 按目录跑 (如 `vitest run src/lib`) 避开。

## 运行 / smoke 留意

- **校准 / HUD 是 AlwaysOnTop**: 独占全屏游戏可能盖不住 (Windows 层限制) → 用窗口化 / 无边框全屏.
- **通道 B（worker 事件校验失败本地化）没真机端到端 smoke**: 要造得手改磁盘 `bin/data/containers/<id>/container.json` 删 WindowTarget 再走热键; 走事件通道 `d.Error` 是对象、不受 [[2026-06-02-wails-dev-fetch-transport-flattens-error]] 影响, 有 Go 单测背书, 按需补.
