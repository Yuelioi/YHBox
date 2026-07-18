---
kind: trap
summary: "节点全集不变式要遍历 sealed Catalog/Projection；adapter stub 绿不证明真实 provider；产品完成还必须走 Source→Run 的黄金旅程。"
activation: symptom
read_when: "批量修改节点/类型/capability，局部测试全绿但真实工作流失败，或准备声明一组节点能力完成时"
recheck_when: "Catalog build、type capability matrix、adapter conformance 或 Stage acceptance 改变后"
---
# 节点契约验证的三种假绿

1. **文本 grep 假绿**：空格、helper、生成式定义和关联类型会漏项。全集规则必须遍历 `nodes.Build()` 得到的 sealed Catalog/Projection，例如“每个 durable TypeRef 都有 state/observe/equality coverage”“每个 requirement 都有可安装 capability manifest”。
2. **stub adapter 假绿**：节点测试只证明调用了接口；真实 provider 可能没有重解析窗口、验证 SendInput 计数、记录 AdapterAction 或释放 held input。设计要求必须在 `internal/noderuntime`/provider conformance 测试中验证。
3. **组件测试假绿**：contract、adapter、页面各自通过仍可能在 capability projection、generation lifecycle、RPC error 或持久化 invariant 的接缝失败。关键能力必须有 Source → compile → admit → provider → journal/asset round-trip，并在对应 Stage 末做真实宿主 smoke。
4. **normalize/default 假绿**：严格 fixture 必须先进入真实 schema/validator，再由 accepted contract 执行 normalization/default；若测试预先补齐默认值，或只测纯 normalization helper，就会掩盖入口 wiring 和 prompt/schema 漂移。

真实 Press Keys 曾因 Host Profile 漏投影专用 key capability 而 `target_unavailable`；录制曾因 fixture 预先满足首事件归零而掩盖真实 recorder 保存失败。两者都说明全量门禁不等于纵向闭环。
