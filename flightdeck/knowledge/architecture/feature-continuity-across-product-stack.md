---
kind: checklist
summary: "替换编辑器或产品栈时，能力连续性必须同时验证入口、管理、创作绑定、运行闭环和真实用户旅程。"
activation: action
read_when: "删除或替换编辑器、产品栈、工作流格式，判断旧能力是否被新架构保留，或准备把一个能力标记 completed 时"
recheck_when: "主导航、Authoring Projection、录制/资产服务、runtime Catalog、Stage acceptance gate 或真实宿主能力改变后"
---
# 跨产品栈验证能力连续性

后端 service、store、节点契约、页面入口或单元测试存在，都不能单独证明用户能力已经保留。一次产品栈迁移必须逐项验证五层：

1. **可见入口**：用户能发现并开始；空状态、取消和失败可恢复。
2. **管理流程**：能完成必要的创建、查看、编辑、删除、搜索、分页或批量处理，并能应对真实规模。
3. **创作绑定**：Authoring Projection 能表达 exact typed reference；编辑器保存 installation slot、BlobRef、state slot 等稳定身份，不保存临时 handle。
4. **运行闭环**：Catalog、Compiler、Admission、provider/runtime 和 journal 消费同一契约与 authority。
5. **黄金旅程**：至少一个从真实输入到真实持久化或宿主副作用的端到端 round-trip；涉及 Windows/ADB 时在 Stage 末有真机证据。

录制必须覆盖 recorder events → canonicalize → finalize → codec → asset reload → playback；模板必须覆盖截图 → 资源库查询 → 搜索式 picker → BlobRef binding → matcher/input effect。只验证其中一段时，应明确写“入口已接”“contract 已有”或“runtime 未验”，不得称为能力完成。

批量验收原则仍然有效：Slice 内只做继续开发所需的定向检查，Stage 末统一跑聚合测试和全量门禁。但 Stage 的边界必须由黄金用户旅程定义；真实宿主 smoke 延期时，Stage 不得 completed。

删除旧 UI/节点前，用旧版本作行为 oracle，反向检查入口、管理、writer/reader、Projection、Compiler、Admission、provider 和 journal。静态节点数量、源码文件数量和页面截图都不是能力连续性的替代指标。
