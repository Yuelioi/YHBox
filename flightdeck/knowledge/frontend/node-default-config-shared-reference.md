---
kind: trap
summary: "历史 3.0 KIND_DEFAULTS 浅拷贝串台案例；3.1 仍保留‘默认 artifact 必须深复制/不可变’的一般教训。"
activation: symptom
read_when: "排查 3.1 EditorSession 新建节点配置串台，或审查 3.0 KIND_DEFAULTS 旧实现时"
recheck_when: "`useNodeCreation` / `useInlineMenu` / `KIND_DEFAULTS` / node registry defaults 生成逻辑变化时。"
---
# ⚠ 新增节点默认 config 浅拷贝会共享嵌套 literal

> 历史案例使用的 `KIND_DEFAULTS/spec.defaults` 已删除；现行原则是 Authoring Projection/default artifact 与 EditorSession snapshot 不可共享可变嵌套引用。
根因：`config: { ...defaults }` 只复制顶层对象，`defaults.literal` 仍是同一个引用。连续新增两个 `Win32WindowTarget` 后，Inspector 直接 mutate 当前节点 `config.literal`，两个节点实际指向同一个 literal，于是保存后两个节点落盘成同一套 Title/Class/ProcessName。

处理原则：

- 创建节点时对 defaults 做深拷贝，再作为该节点的 `config`。
- snippet / 已有节点复制路径如果来自用户数据，也要确认已有深拷贝；不要把 registry 的 defaults 对象直接塞进 graph。
- 回归测试应直接断言两个新建节点的 `config.literal` 不是同一引用，且改第一个不会污染第二个。
