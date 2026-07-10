# Build checklist

SUMMARY: 编译 / 验证产物的前置约定 — task dev/build / pnpm / bindings / 测试套件 / smoke 留意
READ WHEN: before compiling / building / verifying production artifact / 跑 runtime 测试套件 / 跑前端 vitest / 真机 smoke
RECHECK WHEN: 改构建命令 (task dev/build) / wails 配置 / vite 配置 / bindings 生成 / 测试套件入口 / 前端测试跑法时

---

编译 / 验证产物时**前置**读这份.

- **frontend 包管理只用 pnpm** (有 pnpm-lock.yaml): `npm install` 撞 `Cannot read properties of null (reading 'matches')` — npm 的 arborist 解析不了 node_modules/.pnpm 布局, 不是网络/缓存问题, 换 `pnpm add` 即好.

- **开发**: `task dev` — vite (port 9245) + wails3 webview 热重载. 改前端实时刷, 改 Go 要重启.
- **打包**: `task build` — 内部先 `vite build` 嵌入 dist, 再 `go build -tags production -trimpath -ldflags="-w -s -H windowsgui" -o bin/Yotta.exe .`, **最后 `upx --best bin/Yotta.exe` 压缩**(~48M→~13M, 启动多 ~300ms 自解壳; manifest/icon/DPI 资源 UPX 原样保留; `DEV=true` 跳过)。产物名是 **`Yotta.exe`**(= `APP_NAME`, 顶层 Taskfile.yml), 不是 YHBox。需要 `upx` 在 PATH(`scoop install upx`)。
- **仅语法 check**: `go build ./...` 可用 (不产 exe), 但产 exe 一定走 task.

**永远别裸 `go build -o YHBox.exe`** — 缺 vite build → frontend/dist 旧/空 → 启动空白; 缺 `wails3 generate syso` → 没 icon + 缺 manifest (admin 提权检测不对).

Taskfile: 顶层 `Taskfile.yml` → `build/Taskfile.yml` (common) + `build/windows/Taskfile.yml`. 仅留 windows + common; `build/darwin/` 空目录是为了 `wails3 generate icons` 不 fail.

## bindings（gitignore 生成物）

`frontend/bindings/` 是 wails 生成物、gitignore. 改 Go 导出符号 / 路由后, 下次 `task dev` / `task build` 自动 regenerate; 手动改名要同步 rename + 内容替换 (vue-tsc 过) 再 build, 否则前端引用旧名.

## 测试基线

当前基线 (2026-06-29) 是 **Go 全量测试应绿**: `go test ./...`。过去记录过的 runtime fixture / fishing-v2 / dependency scanner 预存红已经不再成立;如果这些测试再红,先按回归处理,不要套旧豁免。

Go 后端质量门禁同时包括：

```powershell
go test -count=1 -covermode=atomic -coverprofile=coverage.out ./...
go vet ./...
staticcheck ./...
go test -race ./internal/services ./internal/services/container ./internal/services/container/runtime ./internal/services/execution ./internal/services/schedule ./internal/services/inputclip/runtime ./internal/hotkey ./pkg/winutil ./pkg/capture
task version:verify
```

对应 CI 是 `.github/workflows/ci.yml`。Linux/macOS 当前先跑 platform-neutral core；完整 `go build ./...` 要在平台 seam 闭合后升级为门禁。

前端 i18n 当前基线也是 **应绿**: `cd frontend && pnpm i18n:check` 应输出 parity / compile / residue 全 OK。旧的 SettingsLauncher / FloatingLauncher residue 42 处硬编码中文记录已过期。

`cd frontend && pnpm build` 当前应绿。2026-06-29 已消掉 `pinSpec.ts` 的 ineffective dynamic import warning;当前已知非阻塞 warning / 提示是:

- `Some chunks are larger than 500 kB`: 主要 chunk 仍偏大,属于后续 code-splitting / bundle budget 议题,不是当前构建失败基线。
- `PLUGIN_TIMINGS` 可能间歇出现: nuxt/ui 与 wails typed-events 插件耗时占比提示,按构建性能议题处理。

## 前端单测 (vitest)

- 前端**有 vitest 套件** (配置在 `vite.config.ts` 的 `test` 块 —— **不是**单独 `vitest.config.ts`; 测试文件 `src/**/*.{test,spec}.ts`, 已有 useEditorSave / useVarMutations / scriptCompletions / ytConsole 等)。
- 跑: `cd frontend && pnpm test` 或根目录 `pnpm -C frontend test`。两者当前都应绿。
- 单文件 / 单目录可用: `cd frontend && ./node_modules/.bin/vitest run <路径>`。

## 运行 / smoke 留意

- **校准 / HUD 是 AlwaysOnTop**: 独占全屏游戏可能盖不住 (Windows 层限制) → 用窗口化 / 无边框全屏.
- **通道 B（worker 事件校验失败本地化）没真机端到端 smoke**: 要造得手改磁盘 `bin/data/containers/<id>/container.json` 删 Win32WindowTarget 再走热键; 走事件通道 `d.Error` 是对象、不受 [wails-dev-fetch-transport-flattens-error.md](../wails/wails-dev-fetch-transport-flattens-error.md) 影响, 有 Go 单测背书, 按需补.
