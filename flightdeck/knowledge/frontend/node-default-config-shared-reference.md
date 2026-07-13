---
kind: trap
summary: "从 `spec.defaults` / `KIND_DEFAULTS` 创建节点时不能只浅拷贝，嵌套 `config.literal` 会被多个新节点共享引用，表现为改一个节点另一个同步变化。"
activation: symptom
read_when: "新增或修改前端节点创建路径；排查“两个新建节点的配置互相同步/串台”；给节点默认值增加嵌套对象或数组时。"
recheck_when: "`useNodeCreation` / `useInlineMenu` / `KIND_DEFAULTS` / node registry defaults 生成逻辑变化时。"
---
# ⚠ 新增节点默认 config 浅拷贝会共享嵌套 literal
根因：`config: { ...defaults }` 只复制顶层对象，`defaults.literal` 仍是同一个引用。连续新增两个 `Win32WindowTarget` 后，Inspector 直接 mutate 当前节点 `config.literal`，两个节点实际指向同一个 literal，于是保存后两个节点落盘成同一套 Title/Class/ProcessName。

处理原则：

- 创建节点时对 defaults 做深拷贝，再作为该节点的 `config`。
- snippet / 已有节点复制路径如果来自用户数据，也要确认已有深拷贝；不要把 registry 的 defaults 对象直接塞进 graph。
- 回归测试应直接断言两个新建节点的 `config.literal` 不是同一引用，且改第一个不会污染第二个。
