# Yotta Go 后端深度审查报告

日期：2026-07-10

范围：仓库内 624 个 Go 文件、59 个 Go package、根装配、节点框架、automation controller、container/runtime、持久化、并发/生命周期、测试与 CI，以及相关 README / Flightdeck knowledge。
原则：只读审查；未修改 Go 业务实现。

## 结论

> 2026-07-11 实施后复核：报告下列 G0-G10 技术项已按 `plan.md` 分批修复并通过本地 test/vet/staticcheck/race/coverage 与双轴 review。新增三平台 CI、生命周期 owner、durable settings/container、runtime capability guard、窄 Ctx、实例 Registry、fuzz 与供应链治理。govulncheck 曾发现 Go 1.25.1 标准库 22 条可达漏洞，提升到 Go 1.25.12 后为 0。尚未完成的外部事实只有未推送所以远端 GUI job 未首跑；战略上当前 LICENSE 禁止商业使用，准确属性仍是 source-available，不是 OSI open source。

如果把 Yotta 评价为“Windows 桌面自动化产品”，后端已经有一批可靠基础：节点 capability 契约严格、错误有结构化路由、controller 已开始按能力拆分、单文件持久化多数采用原子替换、核心 package 的测试密度不低。

如果把目标改为“多平台、大型、可长期演进的开源项目”，目前仍处于架构迁移中段，尚不能称为牢固底座。主要原因不是业务代码量，而是几条 load-bearing seam 还没有闭合：

1. 平台中立 runtime 仍直接依赖 Win32。
2. 启动的后台资源没有统一 owner 和对称 shutdown。
3. settings 与 container package 的持久化缺少完整快照/事务语义。
4. 节点扩展接口位于 `internal/`，并依赖全局 `init()` registry，外部扩展事实上不可用。
5. 没有 PR/push 质量门禁，release workflow 还引用已不存在的版本源。

综合判断：

| 维度 | 评分 | 判断 |
|---|---:|---|
| 现有 Windows 业务内核 | 7/10 | 结构已经明显优于脚本式项目，核心节点与 runtime 有较多测试 |
| 设计理念与模块深度 | 6/10 | controller seam 很好，但 `Ctx` / `RuntimeContext` 暴露过多隐含约束 |
| 后续可扩展性 | 4/10 | 可在仓库内继续加节点；不可形成稳定的外部扩展生态 |
| 代码健壮性 | 5.5/10 | 基线测试绿，但生命周期、持久化和 OS 边界仍有系统性风险 |
| 多平台就绪度 | 2/10 | Linux/macOS 当前不能编译，README/CI/Task 也仍是 Windows 产品形态 |
| 大型开源工程就绪度 | 4/10 | 缺 canonical module path、贡献者架构文档和持续质量门禁 |

## 做得好的部分

### 1. 节点 capability 契约是清楚且有约束力的

`Node` 只负责 `Spec()`，执行语义由 Runnable / RegionRunner / Evaluator 表达；registry 在注册期验证 exactly-one capability、pure-data 必须有 Evaluator、NeedsWindow 必须声明 Window pin。复杂度集中在 framework，而不是散落到 137 个节点调用者中。这是一个有 leverage 的深模块方向。

### 2. automation controller 的能力拆分方向正确

`Controller` 只保留 target、capability 与 health；截图、指针、键盘、应用生命周期按独立 interface 组合。Win32、Android ADB、Browser CDP 可以作为不同 adapter，这是真实 seam，而不是为了测试制造的抽象。

### 3. 运行时并发基本纪律较好

ExecutionQueue / Worker 有明确串行执行不变量；runtime 的 vars、窗口、帧缓存、trace、borderless 状态分别持锁；取消使用 context；inputclip 回放用 defer 释放 held input。针对 services/container/runtime/execution/schedule/inputclip/hotkey 的 race 测试在当前测试覆盖内通过。

### 4. 错误路由比裸字符串错误成熟

`Coded`、`NodeError`、集中错误码和 Fail exec 分支让业务失败可以被图捕获，同时保留 cause 链。这比在 dispatch 中按错误字符串猜类型稳健。

### 5. 多数重要 store 已有单文件原子写

asset、subgraph、schedule、codesnippet 和 container 的单个 JSON 文件普遍采用 temp + rename；这说明项目已经意识到崩溃一致性问题，后续应把该原则提升到“多文件事务”和“目录 fsync”层面，而不是推倒重来。

## 分级问题

### G0 / 目标阻断：多平台 seam 尚未闭合

Linux 与 macOS 的 `GOOS=<os> CGO_ENABLED=0 go build ./...` 都失败，直接原因包括根包导入 `github.com/lxn/win`，以及 `internal/services/autostart.go` 导入 `golang.org/x/sys/windows/registry`。只有 inputclip/QPC 和 ADB command 的少数文件有 build tag。

更深层原因是 `RuntimeContext` 同时持有：

- 平台中立的 `RuntimeControllerFactory`；
- 旧的 `pkg/input.Backend`、`pkg/capture.IBackend`；
- `winutil.WindowHandle`、`win.HWND`、borderless window state；
- Android/browser controller 所需状态。

这意味着 controller seam 已建立，但旧 Win32 seam 没有被替换，两个模型并存。新增平台必须理解并接通两套接口，复杂度会重新扩散到 runtime。

建议：让 `container/runtime` 只依赖 `automation/controller`、`automation/target` 与平台中立值对象。Win32 window/input/capture 全部移到 Windows adapter package，以 `_windows.go` / `_stub.go` 或平台目录隔离；Linux/macOS CI 必须能至少完成 `go build ./...`。

### G1 / P1：没有持续质量门禁，release workflow 已漂移

`.github/workflows/` 只有 tag 触发的 Windows release，没有 pull request / push 的 `go test`、`go vet`、staticcheck、race、coverage 或跨平台 compile matrix。大型开源项目无法依靠开发者本机习惯维持底层不变量。

release workflow 的“同步版本”仍尝试替换 `main.go` 中的 `var version`，但当前真实版本源是 `pkg/version.Version`；项目已经有正确的 `scripts/bump-version.ps1`。如果 tag 不是由脚本产生，release 产物可能显示旧版本。

建议先建立不发布产物的 `ci.yml`：Windows 全量 test；Windows vet/staticcheck；关键并发包 race；Linux/macOS compile；coverage artifact。release 只消费已经通过的 tag/commit，不再在 CI 内临时改源码版本。

### G2 / P1：后台资源没有统一生命周期 owner

`main()` 启动 Worker、MCP HTTP server、ScheduleDaemon、hotkey manager、calibration/recording/tools 等资源；`wailsApp.Run()` 返回后只调用 `app.Shutdown()`，而该方法仅 flush LogSink。

当前未看到正常退出时对以下资源的对称关闭：

- `worker.Stop()` / cancel 当前 container；
- `scheduleDaemon.Stop()`；
- MCP HTTP shutdown；
- hotkey registry/manager 释放；
- recording/calibration/tool window 的统一停止。

进程退出会由 OS 回收句柄，但不会保证当前 input run 的 defer、held-key release、队列状态事件和日志尾部完整执行。它也使集成测试难以“启动应用核心 → 验证 → 关闭且无 goroutine 泄漏”。

建议引入一个真正的应用 runtime 模块：构造时只组装，`Start(ctx)` 按顺序启动，`Close(ctx)` 逆序关闭；所有后台 goroutine/daemon/server 都归它所有。`main()` 只负责平台壳与退出码。

### G3 / P1：settings 快照接口泄漏可变指针，保存不是原子操作

`App.Settings()` 在 RLock 内取得 `*Settings` 后直接返回；锁释放后调用者继续读取同一可变对象。`SaveSettings()` 同样只在锁内复制指针，然后解锁序列化。与此同时 `UpdateWindowSize()` 会原地修改该对象，SettingsService 也会 swap 新指针。

现有 race 测试没有覆盖这些调用同时发生的场景；源码接口本身没有建立 immutable snapshot 保证。

此外，`SaveSettings(path, s)` 直接 `os.WriteFile` 覆盖目标。写盘中途崩溃或磁盘错误可能留下截断的 settings；下次加载会回退默认值，用户配置静默丢失。

建议：删除/私有化裸指针 getter，所有读取返回深拷贝或不可变快照；保存先在锁内取得 clone，再用同目录临时文件 + flush/sync + atomic rename，必要时保留 `.bak`。

### G4 / P1：container 四文件保存不是事务，load 不验证 lock

Container Save 会依次写 `package.json`、`graph.json`、`installation.json`、`yotta-lock.json`。单文件是原子的，但遍历源是 Go map，顺序不确定；任意一步失败或进程中断都会留下新旧文件混合。

启动 `loadOne()` 只读取前三个文件并 aggregate，不读取或校验 `yotta-lock.json`。lock 只在 export 时验证。因此跨代混合可能被当成正常 container 加载，而不是明确标记 incompatible。

建议采用代际目录或 staging directory：完整写入并 fsync 新 generation，校验 lock 后一次切换 manifest/current 指针；最低限度也应固定写入顺序、lock 最后提交、load 时校验 lock，并对不一致进入可恢复状态。

### G5 / P1：节点模型适合仓库内扩展，不适合外部生态

当前 `go.mod` 使用 `module yotta`，节点契约位于 `internal/node`，外部 Go module 无法 import；注册依赖各 package 的 `init()` 和 `internal/nodes/all` blank import，全局 registry 再被 Freeze。

这对“所有节点都由主仓库编译”的单体产品很有效，但对大型开源项目的第三方节点、独立模块或插件生态是硬阻断。README/架构文档没有明确“只接受 in-tree 扩展”还是“未来支持 out-of-tree 扩展”。

建议先做产品决策：

- 如果只支持 in-tree contribution，明确写入架构文档，不为假想插件增加抽象。
- 如果需要外部扩展，把稳定 contract 移到 canonical module path 下的 public package；registry 改为实例依赖注入，不再以全局 init 作为唯一入口；序列化 kind/spec/version 需要兼容策略。

### G6 / P1：`node.Ctx` 是宽 service locator，隐含 nil 与调用顺序

`Ctx` 暴露 13 个 service getter，`VisionService` 单个 interface 又包含十余类能力。knowledge 与源码都约定字段可能为 nil，节点直接调用 nil service 会 panic，再由 framework recover。

仓库内至少有 70 处 `ctx.X().Method()` 直接调用。调用者必须知道“这个 Spec 在什么 setup 阶段保证哪个 service 非 nil”，这部分 interface 不在类型系统里，模块偏浅：实现细节与调用顺序泄漏给每个节点和测试。

建议不要继续给 `Ctx` 横向加方法。先让 Spec 声明所需 capability，dispatch 在执行前校验 bundle 并返回结构化配置/装配错误；再逐步把 automation 类服务统一到 controller capability。只有确实存在多个 adapter 的能力才保留 seam。

### G7 / P2：质量工具当前不全绿

- `go vet ./...`：4 处 possible misuse of `unsafe.Pointer`，位于 calibration、recording LL hook、WGC capture。
- `staticcheck ./...`：发现未使用函数/常量、错误字符串风格等约 29 项；其中有 runtime/config helper、container path helper、ADB coordinate helper 等疑似死代码。
- `go test -cover ./...`：container coverage build 失败。`rewriter.go` 文件开头带 UTF-8 BOM；普通编译接受，但 coverage instrumentation 后 BOM 不再位于文件起点，报 `invalid BOM in the middle of the file`。
- 尚未安装/执行 `govulncheck`；本报告不对依赖漏洞状态作“安全”结论。

建议 CI 将 vet/staticcheck/coverage 设为门禁；unsafe 位置逐处证明指针生命周期和 syscall ABI，不要简单 suppress。

### G8 / P2：平台关键路径覆盖率低，且没有 fuzz 测试

coverage 在 container 处中断，但已完成 package 显示：calibration 7.8%、capture 18.0%、input 20.7%、recording 34.1%、MCP 38.6%、winutil 30.5%。这些正是最依赖 OS、unsafe、网络或生命周期的区域。

仓库没有 `func Fuzz...`。Container/package JSON、graph rewrite、expression/script 输入、导入导出和 MCP 参数都是适合 fuzz 的非可信边界。

建议先为可纯化的 platform adapter contract 建 fake/in-memory adapter 测试；对 parser/importer/rewriter 增加 fuzz corpus。不要用“覆盖率百分比”替代关键不变量测试。

### G9 / P2：高负载下的时序测试不稳定

Windows 全量 test 与 vet、Linux build、macOS build 并行时，inputclip runtime 的 2ms precision test 和 15ms playback tolerance test失败；单独 `go test ./...` 全绿，inputclip runtime 隔离重复 20 次也全绿。

这说明默认实现当前可用，但测试依赖宿主调度延迟，容易在共享 CI runner 或高负载开发机产生假红。测试应该注入 clock/wait strategy 验证调度决策，把真实 QPC 精度测试放到单独的 platform/integration job，并使用统计/环境感知阈值。

### G10 / P2：错误处理存在“刻意忽略但不可观测”的区域

生产 Go 代码中约 79 处显式忽略返回值。很多是 cleanup 的合理 best-effort，但以下类型需要收紧：settings resize 保存失败、schedule reload/register/save、hotkey callback 持久化、日志写盘、PNG encode、queue enqueue。

大型项目可以允许 best-effort，但必须有统一规则：cleanup error 可聚合/记录；用户数据写失败必须可观测；queue/registration 失败不能静默改变行为；测试 helper 和 goja property set 可按明确原因忽略。

## docs / knowledge 漂移

| 位置 | 当前记载 | 源码现实 | 结论 |
|---|---|---|---|
| `README.md` | Windows 桌面工具，围绕《异环》后台自动化 | 用户当前目标是多平台大型开源项目 | 这是战略方向变化，不是小文案错误；需要新的产品定位与平台矩阵 |
| `flightdeck/knowledge/nodes/error-model.md` | 只有 7 个节点拿 Fail，region 只有 3 个 | node catalog 当前有 22 个 kind 带 Fail；ForEach 也是 RegionRunner Fail，另有 AI/Script/Fetch/File/Image/GetWindow 等 | 明显过期，不能继续作为错误模型数量/范围的权威来源 |
| 同一 error-model knowledge | 错误码列表未列 `window_invalid` | `errorcodes.go` 已注册 `CodeWindowInvalid` | 过期 |
| `node-system-architecture.md` | 声称 `interfaces.go:44-48` 仍有一段 GetVar/GetParam 陈旧注释 | 当前该行已是正确的 Evaluator 注释 | “提醒陈旧注释”的提醒本身已陈旧 |
| `.github/workflows/release.yml` | tag 时替换 `main.go` 的 `var version` | 当前版本源是 `pkg/version/version.go`，官方 bump 脚本也更新该文件 | release 流程漂移 |
| `internal/services/app.go` 注释 | `Shutdown` 是“集中退出钩子，挂在 window close 钩子上” | 实际只在 `wailsApp.Run()` 返回后调用，且仅 flush LogSink | 注释夸大了生命周期职责 |

此外，仓库没有面向贡献者的 `docs/architecture`。现有深层设计知识主要存在 Flightdeck knowledge 中，内容质量不低，但数量事实容易漂移，也不容易成为开源贡献者的稳定入口。

建议将“为什么这样设计”的稳定部分提炼为仓库架构文档；节点数量、错误码全集等可生成事实不要手写静态数字，改由 catalog/测试生成或链接单一来源。

## 建议的目标形态

```text
cmd/yotta
  └─ application runtime（唯一生命周期 owner）
       ├─ domain/node engine（平台中立）
       ├─ automation contracts（平台中立、小 interface）
       ├─ storage transactions（settings/container/assets）
       └─ adapters
            ├─ windows（Win32/WGC/LL hook/registry）
            ├─ android（ADB）
            ├─ browser（CDP）
            └─ test/replay/mock
```

约束：

- domain/runtime 不 import `lxn/win`、`x/sys/windows`、`pkg/winutil`。
- 所有后台资源由一个 runtime `Start/Close` 管理。
- 所有用户数据保存要么是单文件原子快照，要么是可校验的多文件 generation。
- 节点执行前验证所需 capability，不以 nil panic 表示装配错误。
- CI 同时验证 Windows 行为与 Linux/macOS 可编译性。

## 推荐治理顺序

### 第 0 阶段：先建立真相门禁

1. 修 BOM、vet、staticcheck。
2. 新增 PR/push CI：test、vet、staticcheck、coverage、关键 race、三平台 compile。
3. 修 release version chain，确定 canonical Go module path。

### 第 1 阶段：闭合平台 seam

1. 画出 Win32 依赖反向树。
2. 让 runtime 只依赖 controller/target contract。
3. Windows 能力移入 adapter；提供非 Windows build 实现或 unsupported adapter。
4. 以 Linux/macOS `go build ./...` 通过作为验收，而不是“文件有 build tag”。

### 第 2 阶段：生命周期与持久化

1. 建立 application runtime `Start/Close`。
2. settings 改 immutable snapshot + atomic save。
3. container 改 generation/transaction + load-time lock validation。
4. 对 shutdown、崩溃中断、部分写入加集成测试。

### 第 3 阶段：扩展契约

1. 明确 in-tree contribution 还是 out-of-tree plugin。
2. 收窄/显式化 Ctx capability，禁止继续扩大 nullable service locator。
3. 在 1.0 前建立 graph/node/schema migration policy。

### 第 4 阶段：开放源代码工程化

1. 增加 architecture、contributing、support matrix、security policy。
2. 对 parser/importer/rewriter 增加 fuzz。
3. 增加依赖漏洞与 license 检查。

## 验证记录

| 检查 | 结果 |
|---|---|
| `go list ./...` | 59 个 package，可解析 |
| `go test ./... -count=1`（隔离） | 通过，约 19 秒 |
| inputclip runtime `-count=20` | 通过 |
| 关键 package `go test -race` | 通过 |
| `go vet ./...` | 失败，4 处 unsafe.Pointer warning |
| `staticcheck ./...` | 失败，dead code / style 等约 29 项 |
| Linux amd64 `go build ./...` | 失败，Win32 package/registry 依赖 |
| macOS amd64 `go build ./...` | 同上 |
| `go test -cover ./...` | 失败，container/rewriter.go BOM 阻断 |
| fuzz tests | 0 |

## 审查边界

- 未运行真机 Windows 输入、截图、ADB、Browser CDP 或 Wails UI smoke。
- 未执行 govulncheck、license scan、性能 profile 或长时稳定性测试。
- 未逐行审计所有 624 个文件；采用入口、依赖图、风险模式、关键模块与工具验证的组合抽样。
