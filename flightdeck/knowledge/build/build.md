# Build checklist

SUMMARY: 编译 / 验证产物的前置约定 — task check/dev/build / pnpm / bindings contract / bundle budget / 测试套件 / smoke 留意
READ WHEN: before compiling / building / verifying production artifact / 跑 runtime 测试套件 / 跑前端 vitest / 真机 smoke
RECHECK WHEN: 改构建命令 (task dev/build) / wails 配置 / vite 配置 / bindings 生成 / 测试套件入口 / 前端测试跑法时

---

编译 / 验证产物时**前置**读这份.

- **frontend 包管理只用 pnpm** (`frontend/package.json` 固定 `pnpm@11.1.2`，有 pnpm-lock.yaml): `npm install` 撞 `Cannot read properties of null (reading 'matches')` — npm 的 arborist 解析不了 node_modules/.pnpm 布局, 不是网络/缓存问题, 换 `pnpm add` 即好. 安装与 CI 一律 `--frozen-lockfile`。

- **Wails library 与 CLI 必须同版**: 当前 pin 是 `v3.0.0-alpha2.117`。安装用 `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.117`；`./scripts/verify-wails-version.ps1` 核对 source pins，`-CheckInstalled` 还会验证 PATH 中实际 CLI。根 `task build/package/dev` 已自动执行实际 CLI 检查。

- **开发**: `task dev` — vite (port 9245) + wails3 webview 热重载. 改前端实时刷, 改 Go 要重启.
- **完整本地门禁**: `task check` — 版本/Wails pin、Go test + coverage floor 65% + vet + staticcheck、frozen frontend install、bindings contract、format/lint/typecheck/i18n/Vitest/production build/bundle budget。
- **打包**: `task build` — 内部先 `vite build` 嵌入 dist, 再 `go build -tags production -trimpath -ldflags="-w -s -H windowsgui" -o bin/Yotta.exe .`, **最后 `upx --best bin/Yotta.exe` 压缩**(~48M→~13M, 启动多 ~300ms 自解壳; manifest/icon/DPI 资源 UPX 原样保留; `DEV=true` 跳过)。产物名是 **`Yotta.exe`**（由顶层 Taskfile 的 `APP_NAME` 决定）。需要 `upx` 在 PATH(`scoop install upx`)。
- **仅语法 check**: `go build ./...` 可用 (不产 exe), 但产 exe 一定走 task.

**永远别裸 `go build -o Yotta.exe`** — 缺 vite build → frontend/dist 旧/空 → 启动空白; 缺 `wails3 generate syso` → 没 icon + 缺 manifest (admin 提权检测不对).

Taskfile: 顶层 `Taskfile.yml` → `build/Taskfile.yml` (common) + `build/windows/Taskfile.yml`. 仅留 windows + common; `build/darwin/` 空目录是为了 `wails3 generate icons` 不 fail.

## bindings（gitignore 生成物）

`frontend/bindings/` 是 wails 生成物、gitignore. 改 Go 导出符号 / 路由后, 下次 `task dev` / `task build` 自动 regenerate; 手动改名要同步 rename + 内容替换 (vue-tsc 过) 再 build, 否则前端引用旧名.

Wails CLI 的 `wails3 generate bindings -dry` 在默认 `-clean=true` 下会先清空现有 bindings；只做预检时必须加 `-clean=false`，否则要立即正式 regenerate，避免 Vitest 因 gitignored import 消失而假红。统一入口 `node frontend/scripts/generate-bindings.mjs` 会拒绝非零 warning；随后 `pnpm -C frontend bindings:check` 对比 tracked `contracts/wails-rpc.json`。alpha2.117 当前基线是 14 services / 112 methods / 86 model+enum declarations；数量不再硬编码到 workflow。接口有意变化后审查 diff，再运行 `pnpm -C frontend bindings:update`。

## 测试基线

当前基线 (2026-07-13) 是 **`task check` 全绿**。过去记录过的 runtime fixture / fishing-v2 / dependency scanner 预存红已经不再成立;如果这些测试再红,先按回归处理,不要套旧豁免。

Go 后端质量门禁同时包括：

```powershell
go test -count=1 -covermode=atomic -coverprofile=coverage.out ./...
./scripts/check-go-coverage.ps1
go vet ./...
staticcheck ./...
go test -race ./internal/services ./internal/services/tools ./internal/services/container ./internal/services/container/runtime ./internal/services/execution ./internal/services/schedule ./internal/services/inputclip/runtime ./internal/hotkey ./pkg/winutil ./pkg/capture
task check:fuzz
task version:verify
./scripts/verify-wails-version.ps1 -CheckInstalled
```

对应 CI 是 `.github/workflows/ci.yml`。Windows `quality-windows` 安装固定工具链后直接运行同一个 `task check`，不在 workflow 复制 Go/frontend 命令。Linux/macOS portable core 已包含 services/tools。独立 `gui-build` job 在 Ubuntu 24.04 amd64、macOS 15 arm64 和 Windows 上核对实际 Wails CLI、frozen install、生成 bindings、执行 production frontend build 并编译 production-tag GUI；Windows 的 `CGO_ENABLED=0` 产物只是 Wails/frontend compile smoke，不替代含 capture DLL、installer 和真实 WebView 启动的发布验收。三个平台直接上传二进制；首次远端运行和 GUI 宿主 smoke 仍是发布前置项。
`pkg/platform`、input/capture/winutil、nodes/input/io、container/runtime 及其平台中立消费者已进入 Linux/macOS 原生测试矩阵；backend dependency graph 不得重新出现 Win32 或 Wails presentation import。

前端 i18n 当前基线也是 **应绿**: `cd frontend && pnpm i18n:check` 应输出 parity / compile / residue 全 OK。旧的 SettingsLauncher / FloatingLauncher residue 42 处硬编码中文记录已过期。lint 的 `lint` 是 check-only；只有 `lint:fix` 会改文件。既有 276 个 `no-explicit-any` 由 `lint-baseline.json` 精确 ratchet：增加或减少都会要求审查并显式更新，不能静默关闭规则。

`cd frontend && pnpm build` 当前应绿且会自动执行 bundle gate。预算按 gzip level 9、十进制 bytes 计算：entry ≤350,000；editor 初始同步 JS ≤650,000，最终 target 450,000。2026-07-13 基线是 entry 308,104 bytes、editor 468,811 bytes。ELK 只在首次自动布局时加载；图标搜索只懒加载 21,825 bytes gzip 的 Tabler 名称索引，完整 `icons.json` 出现在 manifest 会直接失败。当前已知非阻塞 warning / 提示是:

- `Some chunks are larger than 500 kB`: ELK 等按需 chunk 仍可能触发 Vite 通用 raw-size warning；是否阻断只以 `bundle:check` 的同步闭包预算和 forbidden checks 为准。
- `PLUGIN_TIMINGS` 可能间歇出现: nuxt/ui 与 wails typed-events 插件耗时占比提示,按构建性能议题处理。

## 前端单测 (vitest)

- 前端**有 vitest 套件** (配置在 `vite.config.ts` 的 `test` 块 —— **不是**单独 `vitest.config.ts`; 测试文件 `src/**/*.{test,spec}.ts`, 已有 useEditorSave / useVarMutations / scriptCompletions / ytConsole 等)。
- 跑: `cd frontend && pnpm test` 或根目录 `pnpm -C frontend test`。两者当前都应绿。
- 单文件 / 单目录可用: `cd frontend && ./node_modules/.bin/vitest run <路径>`。

## 运行 / smoke 留意

- **校准 / HUD 是 AlwaysOnTop**: 独占全屏游戏可能盖不住 (Windows 层限制) → 用窗口化 / 无边框全屏.
- **通道 B（worker 事件校验失败本地化）没真机端到端 smoke**: 要造得手改磁盘 `bin/data/containers/<id>/container.json` 删 Win32WindowTarget 再走热键; 走事件通道 `d.Error` 是对象、不受 [wails-dev-fetch-transport-flattens-error.md](../wails/wails-dev-fetch-transport-flattens-error.md) 影响, 有 Go 单测背书, 按需补.
