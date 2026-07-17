---
kind: checklist
summary: "替换编辑器或产品栈时，能力连续性必须同时验证可见入口、创作绑定和运行闭环。"
activation: action
read_when: "删除或替换编辑器、产品栈、工作流格式，或判断旧能力是否已被新架构保留时"
recheck_when: "主导航、工作流 Authoring Projection、录制/资产服务或 runtime Node Catalog 改变后"
---
# 跨产品栈验证能力连续性

后端 service、store 或 runtime 仍存在，不等于用户能力已经保留。一次产品栈迁移至少要逐项验证三层：

1. **可见入口**：主导航、编辑器工具栏或相关上下文中存在可发现入口，并能完成创建、管理和错误恢复。
2. **创作绑定**：新工作流的 Authoring Projection 能表达该资源或目标，编辑器能选择精确安装项或 immutable content binding。
3. **运行闭环**：Catalog 存在对应 Node Contract，Compiler 能生成 Program，runtime adapter 能消费同一类型、capability 和 target authority。

检查录制和资源能力时还要确认 pending→finalize 生命周期、全局资产元数据、内容寻址 BlobRef、节点输入编辑器与回放/视觉 adapter 使用同一契约。若只有其中一层存在，应明确称为“入口断开”“创作未接入”或“runtime 未实现”，不能笼统称为已保留或已删除。

删除旧 UI 前，按能力清单反向搜索路由、主导航、store 调用者、Wails RPC、服务注册、Node Catalog 和 runtime conversion；任何 store/service 零调用都应视为产品连续性回归信号。
