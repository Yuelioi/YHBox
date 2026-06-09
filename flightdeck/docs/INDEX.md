# docs/ — INDEX

<!-- AUTO:docs -->
- [asset-subsystem.md](asset-subsystem.md) — active — when_to_read: 改模板/clip 存储 / 节点引用素材 / 运行时模板匹配 / 子图分享导入 / 资产 picker UI 前; 撞"刚导入/重拍的素材查不到"、"(子图未找到)"、节点引用 GUID 失效、变体分辨率挑错档 类问题 — applies_to: [asset, template, clip, blob, guid, content-addressed, variant, PickVariant, library, import, export, subgraph-share, internal/services/asset, wire_container.go, wire_templates.go, internal/services/container/library, TemplatePicker.vue, stores/library.ts, stores/templates.ts]
- [error-model.md](error-model.md) — active — when_to_read: 给节点加错误处理 / 加新错误码 / 改 dispatch 失败路由 / 加 region 容错 / 撞「节点报错没被 Fail 出口接住」类问题前 — applies_to: [error-model, node-framework, dispatch, Failf, NodeError, Coded, errorcode, Fail-output, Throw, region, validator]
- [framework-extension-dispatch-context.md](framework-extension-dispatch-context.md) — active — when_to_read: 设计 framework / DI 容器扩展 / 节点 Ctx 加新方法前 / "这种节点该看到不同的 X" 类需求 — applies_to: [framework-design, node-system, ctx-interface, ServiceBundle, dependency-injection]
- [node-system-architecture.md](node-system-architecture.md) — active — when_to_read: 第一次碰节点系统 / 设计新节点前想搞懂"节点怎么被定义·注册·派发" / 不确定一个节点该实现哪种 capability / 改 framework dispatch 或 validator 前 — applies_to: [node-system, framework, spec, registry, capability, dispatch, runnable, regionrunner, evaluator, validation]
- [node-system-reference.md](node-system-reference.md) — active — when_to_read: 查 pin 类型有哪些 / 查节点 Run 里能拿哪些 ctx 服务 / 不确定 pin 值取值优先级 / 想要节点 kind 全目录 / 用拟人化 jitter — applies_to: [node-system, pin-types, ctx-services, inputs, outputs, catalog, jitter, reference]
- [variable-system.md](variable-system.md) — active — when_to_read: 加变量类型 / 加变量节点 (GetVar/SetVar/IncVar/VarLastChange/捕获框) / 改变量引用存储或读写法 / 撞 'config 有 varName 但 widget 空 / runtime missing VarName' 类 bug 前 — applies_to: [variable-system, VarDecl, scope, config-literal, VarName, capture, var-type, consumer-audit, GetVar, SetVar, IncVar, VarLastChange]
<!-- /AUTO -->

<!-- hand-maintained below; AI 不动 AUTO 区，可改这里 -->

## 关于 docs/

自己写的**常驻、解释性**技术知识 —— "系统怎么运作 / 为什么这么建"（架构、设计理由、子系统概览、模块参考）。**读它来理解**。

跟邻居区分：
- vs `checklists/`：docs 读着理解；checklist 照着执行（含约定/规范，如 add-node、node-spec-style）。
- vs `references/`：docs 是我们自己写的；references 是外部导入。
- vs `specs/`：spec 是一次性设计（建完归档）；doc 描述已建成的系统，常驻不归档。feature ship 后 spec 进 archive，"它最终怎么运作"的持久知识 graduate 进 docs。
