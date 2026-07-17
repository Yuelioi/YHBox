# 节点编辑器交互调研

## Research Read

调研了 Unreal Engine 节点/连线/图整理/Blueprint Debugger、React Flow API 与 ELK 示例、Vue Flow 当前能力、ELK layered 算法、VS Code 调试心智，以及 Yotta 新旧实现。

## Source Matrix

| 来源 | 提取内容 | 本地应用 |
| --- | --- | --- |
| [Unreal Nodes](https://dev.epicgames.com/documentation/en-us/unreal-engine/nodes-in-unreal-engine) | pin 拖空白打开兼容菜单并自动连线；多选/框选 | 下一节点推荐由连接意图触发 |
| [Unreal Connecting](https://dev.epicgames.com/documentation/unreal-engine/connecting-nodes-in-unreal-engine?lang=en-US) | hover 提示可连性和原因 | 提交前预防错误 |
| [Unreal Organizing](https://dev.epicgames.com/documentation/en-us/unreal-engine/organizing-a-material-graph-in-unreal-engine) | 六种对齐、水平/垂直等距 | 作为基础图整理能力恢复 |
| [Blueprint Debugger](https://dev.epicgames.com/documentation/unreal-engine/blueprint-debugger-in-unreal-engine?lang=en-US) | 执行前断点、当前节点、watches | 真调试至少 pause-before-node |
| [React Flow API](https://reactflow.dev/api-reference/react-flow) | onNodesChange、connect start/end、selection、drag threshold | 消除外部投影与内部手势双事实源 |
| [React Flow utility classes](https://reactflow.dev/learn/customization/utility-classes) | nodrag 隔离交互控件 | header drag handle，handles/控件 nodrag |
| [React Flow ELK](https://reactflow.dev/examples/layout/elkjs) | 异步布局后更新位置 | revision/context token 防过期写回 |
| [ELK Layered](https://eclipse.dev/elk/reference/algorithms/org-eclipse-elk-layered.html) | 有方向分层、端口约束、减少交叉 | LR/TB 主工作流布局 |
| [VS Code Debugging](https://code.visualstudio.com/docs/debugtest/debugging) | continue/pause/step/stop、变量、watch、断点 | timeline 不能冒充 Debug |
| [Vue Flow](https://vueflow.dev/) | 拖拽、选择、事件与自定义节点 | 用框架事件建适配层 |

## Patterns

1. 从端口意图进入创作：只显示能完成当前连接的节点，多端口时显式选择，保留“显示全部”。
2. 预防错误优于提交后报错：候选、hover 和最终 connect 共用兼容规则。
3. 点击、移动、连线互斥：一次手势只产生一种领域命令。
4. 图整理是上下文动作：多选后才出现，对齐/布局作为一次 undo，保持视口和中心。
5. 调试是执行控制，不是日志皮肤：必须在调用节点前暂停并持有同一运行上下文。

## Local Application

- Vue Flow 是交互投影，EditorSession 是持久位置事实源；手势结束只提交一次。
- 从现有 validateEdge 抽兼容服务；泛型或资源规则若前端表达不全，提供 compiler-backed preview 查询，不复制旧 pinTypeCompat。
- ELK 恢复实测尺寸、过期结果保护和中心锚定。
- 当前 Debug 先改成“运行并查看时间线”；真控制点进入唯一 scheduler。
- 模板节点使用新 contract、BlobRef、exact target 和显式 timeout/error。

## Risks

- 简单类型字符串比较会漏 carrier、instruction、泛型和资源语义。
- 异步布局写入已切换或已编辑的图会破坏内容。
- watches 无限保存 Blob/图像/敏感文本会泄露数据。
- 从节点开始会绕过前置状态和授权。
- 只验 DOM 会漏掉点击产生隐性 move-node 历史，需命令级断言。

## Next Step

先完成 Slice 1 的追踪和位置事实源收敛；Stage 1 三个 Slice 完成后一次性做完整验收。
