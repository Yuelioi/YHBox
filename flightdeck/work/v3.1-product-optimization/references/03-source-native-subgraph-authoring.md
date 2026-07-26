# Source-native subgraph authoring

## Outcome / Question

在不增加第二套 runtime 的前提下，让用户在子图内部清楚看见流程从哪里进入、从哪些命名出口离开，并让内部边界、接口面板、调用节点和面包屑都读取同一份 Workflow Source。

## Completion criterion

- 子图画布投影一个不可删除的入口和每个命名 exit 对应的出口；它们不进入 Catalog、Program 或 runtime。
- 父图 GraphCall 保持一个 exec 输入并镜像 typed data ports 与命名 exits。
- 折叠选择自动推导并重连一个入口、多个 exec/error exits 和 typed data boundary。
- 多执行入口被明确拒绝并定位冲突，不暗改为真正多入口语义。
- 用户可以进入子图、理解内部流向、返回父图、保存重开并继续运行。
- authoring 投影由一个深 Module 负责边界节点、展示连线、连接命令与位置策略，View 不复制 Graph interface 规则。

## Blocked by

- Slice 02 完成稳定的选择与画布手势。
- 现有 workflow-source、EditorSession collapseSelection 与 compiler GraphCall expansion 保持权威。

## Verification

Slice 内只运行快速定向测试：

- authoring 投影测试覆盖一个入口、完成出口、失败出口和 typed data boundary。
- 连接命令测试覆盖入口/出口更新与非法多入口拒绝。
- collapseSelection 回归测试覆盖自动重连和错误解释。
- GraphCall 与内部边界使用同一 Source 事实的 round-trip 测试。
- 阶段 A 完成后统一运行聚合前端测试、task check、真实 Windows WebView/UAC smoke 和人工视觉验收。

## Out of scope

- 真正多执行入口 GraphCall。
- 旧 Container、旧 Subgraph RPC、旧虚拟节点持久化或旧 runtime。
- 调试器修复。
- 宏与精准录制。

## Result

- 新增 Source-native authoring projection：每个子图投影一个入口、命名执行/错误出口和可选 typed data output，边界不进入 Catalog、Program 或 runtime。
- 边界连线手势直接更新同一份 Graph interface；接口面板、GraphCall 和内部边界不维护平行状态。
- 多执行入口推导会拒绝并解释冲突；折叠子图预检会定位不可安全折叠的边。
- 图切换会清理陈旧选择并在 DOM 稳定后适配视口；minimap 默认关闭，避免遮挡子图出口。
- WebView smoke 增加子图截图、边界裁剪和遮挡硬断言；最新真实截图中入口、完成/失败出口与接口面板均完整可见。
