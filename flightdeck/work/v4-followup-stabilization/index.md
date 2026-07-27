# V4 后续稳定性收尾

## Goal

修复新增节点首次保存时作者补丁 schema 错误拒绝 `$临时句柄` 的问题，并对 2026-07-28 的产品反馈修复
完成一次最终增量验收，使当前 V4 工作区可以稳定续用。

## Status

Open

## Current

2026-07-28 的阶段性产品修复已整理为待提交状态：包括单实例启动恢复、启动器编排、AI 密钥编辑、
工作流保存冲突反馈、Macro 滚轮与编辑入口、工作流首页默认管理视图等。相关修改开发过程中跑过定向
测试；最后一次跨范围 `task check` 尚未运行，留待补丁 schema 修复完成后统一执行。

已最小复现新增“启动已安装应用”节点保存失败：应用槽位 `htgame` 与后端执行路径均有效，失败发生在
生成的作者补丁 schema。`connect.edge.to.nodeId` 复用了持久化 Source 的正式 Endpoint 规则，拒绝同一
补丁内合法的 `$node_launch` 句柄，因此 RPC 提前返回 `INVALID_FIELD`。

## Next

在 [作者补丁合同](../../../internal/workflow/authoring/contract.go) 增加补丁专用节点引用/连线类型，以回归
测试锁定 `$handle` 可用于新增节点的同批配置与连线，同时保持持久化
[Workflow Source Endpoint](../../../internal/workflow/schema/model.go) 仍只接受正式 ID；完成后按
[构建验收指南](../../knowledge/build/build.md) 先跑定向测试，再只运行一次 `task check`。

## Progress

- 2026-07-28 建立阶段性交接：确认 `INVALID_FIELD` 不是应用槽位配置错误，而是作者补丁 schema 与执行器
  的临时句柄语义不一致；后端同场景保存通过，生成 schema 对
  `/commands/2/connect/edge/to/nodeId = "$node_launch"` 稳定复现失败。
