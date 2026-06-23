# plans/ — INDEX

<!-- AUTO:plans -->
- [2026-06-23-ai-nodes.md](2026-06-23-ai-nodes.md) — active — ② AI 节点三阶段 TDD 实现计划: Phase A 文本节点(ctx.AI() Provider 指纹缓存 + AI 节点 Spec/Run + {{}} 模板 + validator + 删连接确认 + FE)/ Phase B 结构化类型输出(DynamicDataFields 机制 + ChatStructured 三模式 + 类型 coerce + FE 输出声明)/ Phase C vision(node.Image + Capture/SaveImage/LoadImage 删 Screenshot + Message 多模态 + applyExecDataEdges 扩动态输入)。
- [2026-06-23-held-exec-outputs.md](2026-06-23-held-exec-outputs.md) — done — TDD 实现 held exec output 缓存: ContainerRunner.execOutputs(per-run, 键 nodeID.field) 在 routeResult 两处 applyCaptures 旁写; pullDataPin 对 exec 出口字段改读缓存(替原 nil); 删 applyExecDataEdges 收编单跳并给 buildDataWireFor 动态分支补 coerceToType; 删 validateExecDataAdjacency + EXEC_DATA_NOT_ADJACENT i18n; 反转跨跳警告测试为端到端直连; 子图键唯一性 + fan-out + loop + 稀疏 + 回归(Fail.Code/vision) 全测。
<!-- /AUTO -->
