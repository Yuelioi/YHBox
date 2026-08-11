# Targets, capabilities and resources

Yotta 把“Workflow 想使用什么”与“这台机器当前连接到什么”分开。这个分界让 Source 可移植，也避免把本机
路径、设备身份和凭据写进 Workflow。

## 四个不同概念

| 概念 | 持久位置 | 作用 |
| --- | --- | --- |
| Target Slot | Workflow node config / Target default | 稳定逻辑名称，例如 `target`、`network` |
| Configured Target | 当前 profile 的 settings | 把 slot 对应到 HTTP origin、应用或 Automation profile |
| Capability requirement | Node Contract / compiled Program | 声明 AI、文件、Blob、Stream 或隔离 guest 所需的窄资源操作 |
| Credential Binding | 当前设备的运行配置 / secure store | 把 Workflow 的逻辑凭据需求连接到本机 secret |

Network、Application 和 Automation 是 **Configured Target**。用户保存配置即完成授权；Application 在每次
Run 开始时取得不可变配置代并直接调用，不为每个节点重复 consent、grant、TTL、identity pinning 或安全扫描。
不要在 node adapter 前重新增加通用安全层：这会改变已选 Target 的语义并直接损害热路径性能。

AI、workspace file、Blob、Stream、Script/Process/Wasm guest 等仍按各自 capability/provider 边界工作。两种
模型不能因为都使用“slot”就互相替代。

## 当前 Configured Target

Settings 中有三类直接 Target：

- Network：slot、label、HTTP origin、响应大小上限和 timeout。HTTP 节点只能相对已配置 origin 发请求。
- Application：slot、label、绝对 executable 和固定 arguments。节点按该配置启动应用。
- Automation：slot、label、target kind、adapter kind、profile version 和 adapter-owned profile JSON。

Automation registry 当前提供以下生产语义：

| Target kind | Adapter | 本机配置保存什么 |
| --- | --- | --- |
| `desktop-window` | `win32` | Application slot、窗口选择规则、input/capture backend 与 timeout 等 `desktop-window` profile |
| `android-device` | `android-adb` | exact serial、package 和 `android-device` profile |
| `browser-cdp` | `browser-cdp` | browser endpoint/page 选择与 `browser-page` profile |

Workflow 只引用 slot。HWND、ADB command/session 和 CDP target 都在 Run 内解析并由 Run owner 关闭；它们不能
写回 Source。不同 adapter 支持的 operation 集合来自 `internal/automation/installed` descriptor，节点和 UI
不应自行维护第二张兼容表。

Settings 更新按下面的真实提交顺序执行：

1. clone/validate candidate，并完整构造尚未发布的 AI/Network/Application/Automation generation；
2. durable 保存 settings，再发布 service 的内存 snapshot；
3. `PreparedInstallations.Commit` 把完整 runtime generation 发布给之后的 Run。

第 3 步的 generation 替换对 future Run 是整体发布；已经排队或运行的 Run 继续持有旧代 lease。它和第 2 步
不是可回滚的跨层事务：如果 durable settings 已提交而 runtime activation 失败，调用方收到 committed error，
不能盲目重试或假装磁盘已回滚；下次启动会从已提交 settings 重建 runtime。

## Workflow Resource 与 Global Asset

两者都可引用 content-addressed Blob，但 authority 和生命周期不同：

| 对象 | 当前 kind | 生命周期 |
| --- | --- | --- |
| Workflow Resource | image、macro、input-clip | 属于一份 Source；导出 bundle 时随 Workflow 携带 |
| Global Asset | template、macro、clip | 属于本机 Asset Library；可以被多个 Workflow 选用或提升为 Source resource |

Image/template 可以有多个 resolution variant，并记录 bbox、regions 与 BlobRef。Macro 是可编辑的语义动作
文档；InputClip 是带时间戳的精准录制流。Asset metadata 包含名称、描述、分类、标签和 origin，实际 bytes
位于 Blob store，引用/GC metadata 位于 Content Catalog。

不要把 `objects/sha256/` 当普通资源文件夹手工增删。资源创建、复制、提升、重写和清理必须通过 Asset/
ResourceAuthoring service，以便 Source/Catalog 引用与 Blob lease 一起提交。

`Snippet` 与上述两种资源无关：它是单个 NodeTemplate 的创作快捷方式，不拥有 Blob 或多节点子图。

## 凭据与可移植性

AI profile 的 provider、endpoint、model、能力与评估信息保存在 settings；API key 由 `internal/securestore`
保存，RPC 只返回是否存在，不返回 secret。Windows 后端使用 Credential Manager；当前非 Windows backend
明确返回 unavailable。

导出 Workflow 前应检查 Target definition/default、Credential requirement 和 package dependency 是否足够让
另一台机器解释需求，但不要把当前机器的实际 binding、路径或 secret 填回 Source。Compiler 负责指出
package/Graph/Node/type 等 Source 问题；`StartRun` readiness 负责当前机器缺失的 Target、credential/provider。
修复应发生在 Source 或当前设备设置的正确一侧。

自动化输入与捕获的实现规则见[Automation task knowledge](../../flightdeck/knowledge/automation/input-and-capture.md)。
