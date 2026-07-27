# V4-L Go 清扫审计

## Goal

为 V4 的 Go 全仓清扫确定有证据的边界、顺序和验收方式。目标是减少重复决策和横向依赖，让核心能力
由更深的 Module 承担；不是按行数拆文件，也不建立第二套 runtime。

## Status

Audited

## Scope

- 进程入口、桌面与 CLI composition、本地存储和 runtime 打开路径。
- Application 命令面、Workflow 编译/执行、Run owner/store 与节点 runtime adapter。
- Target/Provider installation 向 Host Profile、Policy 和 runtime provider 的投影。
- Wails Workflow service 的 use case 与 DTO 边界。
- source compatibility、durable compatibility、测试便利 API、smoke 工具和架构文档。

## Out of scope

- 本 Slice 不修改生产代码。
- 不因包或文件较大就拆分。
- 不删除任何尚能从已发布磁盘格式进入的兼容读取。
- 不改动前端体验或 Wails RPC 契约。

## Findings

完整证据见 [Go 清扫审计](../references/go-cleanup-audit.md)。实施优先级为：

1. 本地 runtime 与 execution environment 单一装配。
2. Program / Adapter ABI / Compiler / Executor 边界校正。
3. Application 内部 Source、Run、Debug 职责深化及 Workflow use case 收敛。
4. 已证明无生产调用的表面删除与 durable compatibility 退役机制。
5. smoke 工具和架构文档收尾。

## Implementation constraints

- `internal/application` 继续是 GUI、CLI、AI、MCP、Schedule 进入唯一 Run 路径的稳定入口。
- production 只能存在一个 Executor；测试 preview 不得演化成平行 runtime。
- 不为只有一个实现的本地协作者新增 interface；优先使用具体的不可变输入和具体 Module。
- 安装事实必须一次封存，再派生 Host Profile、Policy、Provider 和 lease；派生结果需做交叉一致性验证。
- 桌面和 CLI 可共享本地 runtime 打开路径，但 Wails window/tray/hotkey 与 CLI 编码仍留在各自 adapter。
- 每阶段先锁定可观察行为，再移动决策；旧测试在新深 Module 的接口测试到位后删除，不叠加双份测试。

## Verification

- 每阶段运行受影响包测试与 `task check`。
- 影响 Source、Program、Target、Run 或 storage 时验证 `fishing-v2` 隔离副本。
- 影响 desktop composition 时运行 production Windows build 和完整 WebView smoke。
- 兼容读取删除前必须用旧版本 fixture 证明已完成一次性持久化改写，并明确最低支持版本。

## Next

1. 为桌面与 CLI 当前启动结果建立等价性测试，覆盖内置 Provider、Target、Profile、Policy 与 limits。
2. 引入一个具体的 sealed execution environment，由安装集合一次派生运行所需全部事实。
3. 让桌面和 CLI 复用本地 runtime 打开/关闭路径，保持 presentation-specific wiring 在外层。
