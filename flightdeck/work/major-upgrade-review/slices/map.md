# Slice registry

完整 frontier 只在这里登记；每项仅包含 identity、status、blocker 与 outcome gist。

| Slice | Status | Blocked by | Outcome gist |
| --- | --- | --- | --- |
| active-entry-switch | completed (be1fc04b) | — | Launcher、Settings、Hotkey 和运行入口切到 Workflow 3.1。 |
| platform-adapter-decontainer | completed (d060798c) | — | Asset、Tools、Recording 使用 installed target/provider。 |
| legacy-product-tree-removal | completed (9fce7870) | — | 删除旧 Container/Subgraph/Node 产品树、RPC、Store 与迁移。 |
| node-contract-single-source | completed (e29ff25d) | — | Catalog、Authoring Projection、文档与 runtime 只消费 3.1 contract。 |
| ai-generic-fallback-removal | completed (99c3f5ff) | — | AI 收口到 provider-native installation/profile/resource contract。 |
| node-package-manifest | completed (a8c0cfb5) | — | 冻结 package/publisher/host API/ABI/payload identity。 |
| node-package-archive-verification | completed (ba2efb65) | node-package-manifest | 安全验证并解包 exact archive，失败不发布。 |
| node-package-local-lifecycle | completed (53e6d8a9) | node-package-archive-verification | 建立无执行面的 immutable generation、registry 与 lifecycle。 |
| startup-workflow-entry | completed (221441e8) | legacy-product-tree-removal | EXE 首屏进入 Workflow，窄 viewport 下列表操作可达。 |
| workflow-editor-interaction-recovery | completed (f3c83737) | — | 修复 DataCloneError，并恢复节点目录单击、加号和拖放。 |
| desktop-startup-privilege-boundary | completed (c3cab6e4) | workflow-editor-interaction-recovery | 普通桌面启动不再进程级提权，高权限能力显式 fail closed。 |
| webview-self-debug-smoke | completed (ab5b644f) | desktop-startup-privilege-boundary | agent 可一键运行隔离 Wails/WebView 交互、错误采集与截图。 |
| restore-go-quality-gate | completed (27e01b17) | — | 恢复 coverage、go vet、staticcheck 与完整 task check 的可信绿色基线。 |
| ai-native-design-disposition | completed (b25a0c6c) | restore-go-quality-gate | 对照代码处置 AI-native 完成定义，只为真实剩余 outcome 创建 Slices。 |
| ai-prompt-tool-provenance | completed (b674664c) | ai-native-design-disposition | 版本化 Prompt/Tool artifacts，并禁止不可信值进入 system/developer。 |
| ai-agent-budget-runtime | completed (d22b5bd5) | ai-prompt-tool-provenance | 建立受 ToolSet、approval 与多维预算约束的 Agent tool loop。 |
| ai-eval-upgrade-gate | completed (cfa12703) | ai-prompt-tool-provenance | 以离线 corpus、确定性 grader 和阈值 gate 管理模型/prompt/tool/schema 升级。 |
| ai-authoring-review-trace | completed (c71cc19f) | ai-agent-budget-runtime, ai-eval-upgrade-gate | AI 只经 typed patch 闭环，并展示 diff、diagnostics、权限 delta 与脱敏 trace。 |
| node-package-signing-trust | current | restore-go-quality-gate | 定义签名 envelope、publisher identity、namespace ownership 与 revocation/quarantine。 |
| stable-code-names-explicit-versions | ready | node-package-signing-trust | 删除 nodes31 等 release 后缀；代码名稳定，版本进入显式 contract/identity 属性。 |
| plugin-hosts-sdk-conformance | fog | stable-code-names-explicit-versions | Wasm/Process host、SDK 与 conformance；禁止 Go/前端插件代码。 |
| final-contract-and-release-acceptance | fog | all implementation slices | projection/reference/golden、task check、review 与真实 Windows smoke。 |
