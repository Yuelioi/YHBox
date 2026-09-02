# 运行证据与职责边界

## Outcome

基于当前生产代码确定 Timeline、Diagnostic、Log、Run 值/附件的真实存储与消费路径，形成一份能约束编辑器定位、
模板调优和 AI 工具设计的职责决策。

## Questions

1. Run 事件当前保存哪些稳定定位信息、计数、状态与值引用，哪些信息在导出时丢失？
2. timeout 在节点语义上是正常 exec output、失败还是二者皆可，UI 当前如何表达？
3. 日志为何稀少：哪些运行事实有意进入 Run Ledger，哪些实现异常仍应进入进程日志？
4. 模板匹配最高分、阈值、候选位置、分辨率与模板来源应进入事件 counters、结构化 detail、Run value 还是附件？
5. AI 读取时最小而充分的 Run Evidence 投影是什么，如何避免泄漏与无界数据？

## Current

当前 Run Ledger 已把 `graphPath`、`nodeId`、attempt、status/action/error code 和最多 64 个非负整数 counters、
64 个短 ASCII facts 持久化；Timeline UI 直接消费该投影。编辑器已有按 graph path/node ID 聚焦节点的 seam。

Program 保存 Workflow ID、Source hash 和 revision，但 Run/Timeline view 只暴露 Program hash；历史 Run 缠绕到
具体 Source revision 需要额外读取 Program，当前 UI 和 MCP 没有这个投影。

模板匹配每帧已经产生 score/center/bounds，轮询结束只保留最后结果且 counters 只接受整数，因此最高分和
采用阈值没有进入 Run Evidence。timeout 是节点的正常 exec output；未连接时 UI 能标识未处理 route，但 Run
仍可整体 succeeded。这需要在产品呈现中区分“Run terminal status”和“节点未满足条件”。

MCP 已提供 catalog search/describe、Workflow list/inspect、typed patch、compile、run preview 和 compiler diagnostic
解释；尚无 Run list/evidence、历史 Source attribution、运行诊断或编辑器导航信息。

进程 Log 使用 zerolog/LogSink，负责显式实现日志并可分别送实时 UI 与文件；Scheduler/Run 的产品事实主要写
Ledger，因此日志少本身不是丢失 Run 历史，但 adapter/infrastructure 异常缺乏系统化维护者上下文。

## Decision

- Timeline 是 Run Evidence 的主要产品视图；不得退化成从文本日志重建 Run。
- Diagnostic 是对 Source/Contract/Run Evidence 的解释，不写回为既成 Run 事实。
- Log 只记录维护实现所需的进程、module、adapter 和 infrastructure 上下文；与 Run 相关时携带 run/node attribution，
  但不复制每个节点的完整 Timeline。
- 小型结构化证据直接进入 Run summary；大图等附件只保存有界、显式、可清理的引用，默认不为每次轮询截图。
- Run Evidence 必须直接投影 Workflow ID、Source hash、revision 和 Program hash，使历史定位不依赖当前 Source。
- 模板 timeout 至少聚合 `best_score_ppm`、`threshold_ppm`、best bounds/center、frame/template dimensions、captures；
  使用定点整数适配现有 counters，避免扩大持久化格式和热路径分配。
- AI 首先读取同一 Run Evidence 投影并产出 Workflow Repair proposal；应用仍使用现有 typed patch + exact revision CAS，
  不能让模型直接替换 Source JSON。

## Next

把以上职责转化为交付接口和验收，随后进入编辑器定位/重新加载决策。
