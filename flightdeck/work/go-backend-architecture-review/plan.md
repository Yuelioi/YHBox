# Yotta Go 后端升级与修复方案

## 目标

把当前“Windows 功能完整、架构迁移中”的 Go 后端，升级为适合大型开源项目长期维护的底座：

- Windows 现有能力不回退；
- Linux/macOS 至少能编译并运行平台中立测试；
- 后台资源有统一、可测试、对称的生命周期；
- 用户数据保存具备明确的崩溃一致性；
- 节点/runtime 的平台与 capability seam 可继续扩展；
- PR 上的 test/vet/staticcheck/race/coverage/跨平台编译成为自动门禁；
- 稳定设计文档与代码单一事实源一致。

## 默认决策

1. **宿主平台矩阵**：Windows 是首个完整功能平台；Linux/macOS 第一里程碑为 backend 可编译 + 平台中立测试可运行。Android/Browser 是 automation target adapter，不等同于宿主 OS。
2. **扩展方式**：近期支持清晰的 in-tree contribution；不承诺 Go `plugin` ABI。代码结构不得继续阻断未来拆出 public node contract。
3. **兼容策略**：项目未发布，内部 Go interface 和 module path 允许一次性切换；用户数据格式只做有价值、可验证的迁移。
4. **Container 一致性**：先实现 lock-last commit + load-time validation；generation directory 作为后续增强，不在第一批同时引入目录迁移。
5. **提交策略**：每个执行批次保持逻辑原子；Codex 已获授权自主创建本地 commit，但没有推送授权。

## 全局验收门

每个阶段至少满足：

- `gofmt` / `git diff --check`；
- 受影响 package 的定向测试；
- `go test ./...`；
- `go vet ./...`；
- `staticcheck ./...`；
- 关键并发 package 的 `go test -race`；
- coverage 命令可以完成，而不是因源码/工具问题中断；
- 阶段 1 完成后，Windows/Linux/macOS 的约定编译命令全部通过。

## 阶段 0：建立可信工程基线

目标：先让所有后续重构都有可靠、快速、自动的反馈。

### 0.1 修复源码与静态工具基线

- 移除 `internal/services/container/rewriter.go` 的 UTF-8 BOM。
- 逐项处理 29 个 staticcheck finding：删除已证实死代码、修正 error string、保留代码则给出真实调用或结构调整，不加无意义 suppress。
- 审核 4 个 vet unsafe warning：优先改用 typed helper/正确 uintptr 生命周期；只有 ABI 明确且无法规避时才做局部、可解释的抑制。
- 补定向测试，确保 cleanup、syscall 返回值和错误链不回退。

验收：`go test -cover ./...`、`go vet ./...`、`staticcheck ./...` 全部退出 0。

### 0.2 新增 PR/push CI

- 新增 `.github/workflows/ci.yml`。
- Windows job：test、vet、staticcheck、coverage artifact。
- race job：services/container/runtime/execution/schedule/inputclip/hotkey。
- 暂时只对已平台中立 package 做 Linux/macOS test；阶段 1 闭合 seam 后切换为 `go build ./...` 门禁。
- 固定 Go 版本来自 `go.mod`，缓存 module/build cache。

验收：workflow 无依赖本地 node_modules 或未声明工具；所有 job 在干净 runner 可复现。

### 0.3 修复 release 与 module identity

- release workflow 不再临时替换不存在的 `main.go version`；校验 tag 与 `pkg/version.Version` 一致。
- release 复用 `scripts/bump-version.ps1` 的单一版本源约定。
- 将 module path 从 `yotta` 切到仓库 canonical path `github.com/yottaapp/yotta`，全量改内部 imports。
- 增加 module/import 回归检查。

验收：本地全量测试通过；tag/version 不一致时 release 明确失败。

## 阶段 1：闭合多平台 seam

目标：平台中立 runtime 不再理解 Win32 实现细节。

### 1.1 建立平台依赖守卫

- 生成/维护允许出现 Win32 import 的 package 清单。
- 增加架构测试：`internal/node`、`internal/automation/target`、平台中立 runtime package 禁止 import `lxn/win`、`x/sys/windows`、`pkg/winutil`。
- 把宿主 OS 与 automation target 两个概念写入架构文档。

### 1.2 收敛 automation controller

- 让输入、截图、窗口/应用操作通过 capability interface 获取。
- 将 runtime 中并存的 `pkg/input.Backend` / `pkg/capture.IBackend` 与 controller 路径合并为单一权威路径。
- Win32 实现在 Windows adapter 内；Android ADB、Browser CDP、Replay/Mock 保持独立 adapter。
- `TargetRef` 的平台字段逐步转为按 kind 验证的 typed payload，避免每加 target 修改所有核心层。

### 1.3 平台文件隔离

- registry/autostart、admin、hotkey、recording LL hook、capture/input/winutil 分出 `_windows.go` 与非 Windows 实现。
- 非 Windows 对不支持能力返回 typed `unsupported capability`，不通过 nil/panic 表示。
- Wails/GUI 壳若暂时仅 Windows，拆成 platform entry；backend package 仍应跨平台编译。

验收：Windows `go test ./...`；Linux/macOS `CGO_ENABLED=0 go build ./...`；平台中立测试三平台通过。

## 阶段 2：统一应用生命周期

目标：每个后台资源都有唯一 owner，启动与关闭严格对称。

### 2.1 引入 application runtime

- 建立应用 runtime 模块，持有 Worker、ScheduleDaemon、MCP HTTP、hotkey、recording/calibration/tools、LogSink。
- `Start(ctx)` 按依赖顺序启动；失败时逆序回滚已启动资源。
- `Close(ctx)` 逆序、幂等关闭；聚合错误，不因一个 Close 失败跳过其余资源。
- `main()` 只负责构造、运行 Wails 壳、传播退出码。

### 2.2 补生命周期契约

- Worker.Stop、Daemon.Stop、server shutdown、hotkey close 全部幂等。
- 退出先拒绝新任务，再 cancel 当前 run，等待 held input release，最后关闭日志。
- MCP server 使用 context/shutdown，不留脱管 goroutine。

验收：启动失败回滚测试、双 Close 测试、活动 run 退出测试、goroutine/held-input 释放测试。

## 阶段 3：强化持久化与状态快照

### 3.1 Settings immutable snapshot + atomic save

- 移除返回 live `*Settings` 的公开读取路径；统一返回 deep snapshot。
- Settings 更新采用 clone → merge/mutate → validate → atomic swap → atomic save。
- 保存使用同目录 temp、file sync、rename、directory sync；失败保留旧文件。
- Window resize 等 best-effort 写失败进入日志/前端可观测事件。

验收：并发读写 race 测试、写失败不破坏旧文件、截断/坏 JSON 恢复测试。

### 3.2 Container lock-last transaction

- 固定写入顺序：package/graph/installation，最后写 yotta-lock 作为 commit record。
- lock 覆盖所有需要一致的文件 hash/schema/generation。
- load 时读取并验证 lock；不一致进入明确 recoverable/incompatible 状态，不静默 aggregate。
- 写入失败保持内存 cache 与磁盘最后已提交 generation 一致。

### 3.3 Generation directory（增强阶段）

- 在需要更强 crash consistency 时，引入 staging generation + current pointer。
- 提供一次性迁移和故障注入测试；无证据不提前复杂化。

验收：对每个写入点注入失败，重启后只能看到完整旧代或完整新代。

## 阶段 4：深化节点/runtime 模块

### 4.1 显式 capability 需求

- Spec 声明节点需要的 runtime capability。
- dispatch 在执行前验证 bundle，缺失能力返回 typed assembly error，不走 nil panic。
- 给 capability 与 Spec 一致性增加 registry guard test。

### 4.2 收窄 `Ctx`

- 核心 Ctx 保留 context/time/output/capture binding 等执行语义。
- automation 操作统一通过 controller capability。
- 仅在存在真实多 adapter 或明确测试 seam 时拆 interface；避免一层转发型假抽象。
- `VisionService` 按调用内聚与 adapter 现实评估是否拆分，不按文件长度机械拆。

### 4.3 Registry 与扩展路线

- 把 global registry 封装成可实例化 Registry；生产仍可有默认实例。
- 测试不再依赖全局 reset；catalog/engine 显式接收 registry snapshot。
- 决定 public contract 后再移出 `internal/`，不提前承诺动态 plugin ABI。

验收：节点缺 capability 不 panic；registry 可并行隔离测试；新增 target/node 不需要修改无关核心模块。

## 阶段 5：可靠性、可观测性与安全

### 5.1 错误处理纪律

- 分类现有显式忽略错误：cleanup、用户数据、队列/注册、日志、编码。
- 用户数据与行为改变类错误必须上报；cleanup 错误聚合记录；允许忽略的调用写出窄范围理由。
- 统一 shutdown/startup/adapter 错误分类与日志字段。

### 5.2 高风险 package 测试

- 提升 calibration/capture/input/recording/MCP/winutil 的契约测试。
- 时序单测注入 clock/wait strategy；真实 QPC 精度移到 Windows integration job。
- 对 graph rewrite、container/package import、expression/script、MCP 参数增加 fuzz corpus。

### 5.3 供应链

- CI 增加 `govulncheck`、依赖 license 检查、Dependabot/Renovate 策略。
- 对网络监听、文件导入、脚本执行、MCP armed 模型形成 threat model。

验收：工具可在干净 runner 运行；安全例外有 owner、原因和复查条件。

## 阶段 6：开源文档与治理

- 新增 `docs/architecture/`：runtime、node engine、automation target/controller、storage、lifecycle。
- 新增 `CONTRIBUTING.md`、平台支持矩阵、兼容/迁移策略、`SECURITY.md`。
- README 从单一游戏/Windows 说明升级为项目定位入口，同时保留当前可用平台的诚实声明。
- 可生成事实（节点数量、Fail 节点、错误码）由 catalog/测试输出，文档不手写静态计数。
- 修正 error-model、node-system、release 与 Shutdown 注释漂移。

验收：新贡献者能从 README 找到构建、测试、架构与贡献路径；文档中的事实可由 CI 校验。

## 执行批次与依赖

| 批次 | 内容 | 依赖 | 主要风险 |
|---|---|---|---|
| A | BOM、staticcheck、vet、coverage | 无 | 删除“死代码”前必须全仓确认调用 |
| B | Windows CI、release version 校验 | A | runner 工具安装与缓存 |
| C | canonical module path | A | 全仓 import 机械变更 |
| D | 平台依赖守卫与 adapter 收敛 | A-C | Windows 行为回归 |
| E | application runtime lifecycle | D 可并行部分 | 关闭顺序、重复 Close |
| F | settings/container durability | A | 用户数据恢复语义 |
| G | Ctx/Registry 深化 | D-F | 节点契约变更面大 |
| H | fuzz/security/docs | A-G 持续推进 | 避免文档再次手写漂移事实 |

## 当前执行点

批次 A-C 已完成：质量工具、CI/release 和 canonical module path 均已落地。当前进入批次 D：建立平台依赖守卫并开始收敛 Windows adapter seam。

## 执行状态

- [x] A — BOM、staticcheck、vet、coverage、关键 race
- [x] B — Windows CI、release version 校验
- [x] C — canonical module path
- [ ] D — 平台依赖守卫与 adapter 收敛
- [ ] E — application runtime lifecycle
- [ ] F — settings/container durability
- [ ] G — Ctx/Registry 深化
- [ ] H — fuzz/security/docs
