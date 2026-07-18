---
kind: trap
summary: "Vue Flow 只有一个活动相机；3.1 每个 Source graph 必须拥有独立、非语义的 viewport 状态。"
activation: symptom
read_when: "修改工作流编辑器视口、切换 main graph/subgraph、GraphCall 导航，或遇到切图后相机跑飞、首次进入看不到内容时"
---
# Vue Flow 单相机与 Source-native graph viewport

Vue Flow 实例只有一个活动 viewport；切换 Workflow Source 中的 graph 只是替换当前 nodes/edges，不会自动恢复目标 graph 的相机。若不显式保存，用户从远处 subgraph 返回 main graph 时会继续停在旧坐标，看起来像内容丢失。

## 现行规则

- viewport 是 presentation state，不进入 Workflow Source、Program 或 graph semantic digest。
- 以稳定的 `sourceId + graphId` 保存 viewport；离开 graph 前读取当前 viewport，进入目标 graph 后恢复。
- 目标 graph 没有缓存时，优先聚焦其 Run Start 或 graph interface anchor；都不存在才 `fitView`。
- Source 关闭/删除时清理对应 presentation state；不能用进程级“当前 graph”单例。
- 异步 layout、fit 或导航结果必须绑定 editor session、source revision 与 graph ID；上下文已变化时丢弃，不回写新 graph 的相机。
- GraphCall、typed interface、annotation 与 reroute 都使用 Source-native graph identity；不得恢复 Container、virtual marker 或旧 subgraph store。

## 验证

main graph 与两个 subgraph 分别 pan/zoom 后往返，三者 viewport 独立；首次进入能看到起始锚点；切换过程中完成的旧 layout 不会移动当前 graph。
