# Yotta architecture and code map

Yotta 是 Wails v3 桌面应用。Vue presentation、Go application command、Workflow compiler/runtime、
configured target 和平台 adapter 分层。桌面进程中的 GUI 与 Schedule 共用一个
`internal/application.Application`；CLI 为所选 profile 打开自己的 `internal/localruntime.Runtime`，但使用相同的
Application 类型和 Program 执行路径。MCP 与 AI authoring 进入同一 Source、typed patch、compile/preview
边界，不拥有 Run 入口或第二套 runtime。

```text
Vue + Schedule (desktop)      CLI
             │                 │
          services / command adapter
                      │
        Application (per profile runtime)
        ┌─────────────┼──────────────┐
     Source → Compiler → Program → Run/Debug
        │          │                  │
   SQLite + CAS  Node Catalog   providers + configured targets
                                      │
                                platform adapters

MCP / AI ──> authoring proposal / typed patch / compile / preview
```

## Composition

启动链只有一条：

```text
main.go
  → internal/desktopapp
  → storage migration/recovery
  → internal/localruntime
  → internal/appbootstrap
  → internal/application
```

- `main.go` 只保留进程入口和嵌入资源。
- `internal/desktopapp/` 组合 Wails 窗口、services、事件和 desktop 生命周期。
- `internal/localruntime/` 是 desktop 与 CLI 共用的核心 profile/Catalog/settings/Application 组装入口；不同进程
  各自打开实例，同一 profile 受 single-writer lease 约束。
- `internal/appbootstrap/` 从 Catalog、节点、provider 和 Target 配置构造执行环境。
- `internal/appruntime/` 管理后台组件的启动、失败回滚和逆序关闭。

`internal/localruntime/` 拥有共享的 storage-backed 核心组装。`internal/desktopapp/` 可以围绕它构造 Wails
presentation、Schedule、Asset、Snippet 等桌面专用 service，但不得再创建第二套 compiler、executor、provider、
Target runtime 或 Application。这个边界由 `internal/architecture/platform_boundaries_test.go` 约束。

## 关键代码地图

| 修改目标 | 权威位置 |
| --- | --- |
| 应用命令、队列、Run/Debug owner | `internal/application/` |
| Workflow typed patch 与 revision CAS | `internal/workflow/authoring/` |
| Workflow schema、编译、Program、scheduler | `internal/workflow/schema/`、`internal/workflow/compiler/` |
| Workflow Source authority 与 Program cache | `internal/workflowstore/` |
| Run Record、journal 与持久化接口 | `internal/run/`、`internal/storage/catalog/` |
| Data Type 与 Node Contract | `internal/datatype/`、`internal/nodecontract/` |
| 内建节点与 sealed Catalog | `internal/nodes/`、`internal/nodecatalog/` |
| 创作投影与 runtime adapter | `internal/nodeauthoring/`、`internal/noderuntime/`、`internal/nodeadapter/` |
| Capability/admission | `internal/capability/`、`internal/admission/` |
| Automation Target/controller | `internal/automation/`、`internal/targetruntime/` |
| AI、HTTP、应用与工作区 provider | `internal/ai/`、`internal/httpegress/`、`internal/appcontrol/`、`internal/workspacefs/` |
| Profile、SQLite、durable files、Blob | `internal/storage/`、`internal/durablefs/`、`internal/blob/` |
| Wails application services | `internal/services/` |
| Vue presentation 与编辑器 | `frontend/src/views/`、`frontend/src/app/`、`frontend/src/lib/` |
| 平台 adapter/helper | `pkg/`；平台中立 package 不应直接依赖它们的 Win32 实现 |
| CLI、生成器与 smoke 工具 | `cmd/`、`scripts/`、`Taskfile.yml` |
| tracked durable/RPC contracts | `contracts/`；`frontend/bindings/` 是本地生成物，不能手改 |

## 核心不变量

- Workflow Source 是唯一创作事实；所有修改通过 typed command 和 revision CAS。
- Compiler 是图类型、端口、state、instruction 和 capability plan 的最终权威；前端不维护第二套规则。
- Node adapter 执行 contract 已声明的 operation，不拥有调度，也不反向依赖 compiler。
- Network、Application 和 Automation 是用户配置的 Target；AI、File、Blob、Stream 和隔离 guest 等资源走
  capability/admission。两类边界不能互相伪装。
- 平台中立核心不 import Win32 adapter；平台差异在带 build tag 的 adapter 或 `pkg/` 边界实现。
- 启动 goroutine、hook、server、worker 或输入持有状态的组件必须有 owner、取消和等待/关闭路径。
- 不可信的 Workflow、package、MCP、文件、网络和进程输入在各自边界 strict/fail closed。

继续阅读：[运行链](runtime.md)、[合同与生成投影](contracts.md)、[本地存储](storage.md)、
[威胁模型](threat-model.md)。面向产品对象的说明从 [Workflow 与创作](../product/workflows.md) 开始；具体
修改步骤见[任务知识](../../flightdeck/knowledge/README.md)。
