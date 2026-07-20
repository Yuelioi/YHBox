# 3.0 历史 Knowledge 退役清单

## Policy

这些条目曾被标记为“历史 3.0”或“仅旧基线取证”，保留目的是在 3.1 恢复期间查行为，不代表现行架构。R5 发布门禁通过后已统一处理。

退役动作只有两类：

- `delete-after-promotion`：先把仍有效的一般原则写进现行 contract/knowledge，再删除旧实现条目。
- `archive-or-delete`：仍有取证价值的内容保留在对应 Finished Work 的参考资料中；没有依赖则删除，旧版本由 Git 追溯。

不得把旧实现条目继续当作现行 `flightdeck/knowledge/`，也不得为了清理而破坏 Finished Work 的证据链。

## Registry

| Path | Marker | R5 action |
| --- | --- | --- |
| `knowledge/architecture/container-commit-durability.md` | 旧 Container lock-last | archive-or-delete |
| `knowledge/frontend/canvas-pinliteral-scalar-vs-inspector-structured.md` | 旧 PinLiteral/StructuredInput | archive-or-delete |
| `knowledge/frontend/node-default-config-shared-reference.md` | 旧 KIND_DEFAULTS；含不可变默认值原则 | delete-after-promotion |
| `knowledge/frontend/yt-scripting-console.md` | 旧 yt/ambient JS | archive-or-delete |
| `knowledge/input/node-timed-input-loses-backend-activate.md` | 旧节点拆 timed input；含 provider 原子语义 | delete-after-promotion |
| `knowledge/logging/short-run-flush-loses-dump.md` | 旧 Container LogMerger | archive-or-delete |
| `knowledge/mcp/normalize-masks-schema-prompt-drift.md` | 旧 Normalize；含严格 fixture 原则 | delete-after-promotion |
| `knowledge/nodes/expression-system.md` | 旧 Expr/$变量 | archive-or-delete |
| `knowledge/nodes/framework-extension-dispatch-context.md` | 旧 Ctx/ServiceBundle；含窄 Invocation 原则 | delete-after-promotion |
| `knowledge/nodes/geometry-pin-value-pct-shape.md` | 旧 Geometry pin DTO | archive-or-delete |
| `knowledge/nodes/held-exec-outputs.md` | 旧 ContainerRunner execOutputs | archive-or-delete |
| `knowledge/nodes/pin-presence-check-must-mirror-pinvalue.md` | 旧 PinValue fallback | archive-or-delete |
| `knowledge/subgraph/autolayout-skips-subgraph-virtual-markers.md` | 旧 virtual markers | archive-or-delete |
| `knowledge/subgraph/draft-subgraphs-phantom-field.md` | 旧 draft/editorStore 双源 | delete-after-promotion |
| `knowledge/subgraph/import-bypasses-container-store-cache.md` | 旧 Container store；含单 owner 原则 | delete-after-promotion |
| `knowledge/subgraph/keepalive-singleton-subgraph-store-stale.md` | 旧 singleton store；含 revision identity 原则 | delete-after-promotion |
| `knowledge/subgraph/merge-pool-preserves-created-empty-shell.md` | 旧 mergePool；含原子 authoring 原则 | delete-after-promotion |
| `knowledge/subgraph/subgraph-marker-pin-convention.md` | 旧 marker pin；含单一 contract 派生原则 | delete-after-promotion |

## R5 retirement result

- 15 条无 recovery dependency 的旧知识删除。
- 3 条历史 Work 固定引用的旧知识保留原路径作为证据；它们不再参与新会话的默认恢复。
- 4 条主动 Knowledge 的旧链接修复；Vue Flow 相机改为 Source-native graph viewport。
- 现行 Knowledge 吸收 defaults、strict fixture、Source/revision ownership 与 native smoke 完整性规则。
- R5 checkpoints exact knowledgeCount = 1 + 10 + 15 = 26。

## R5 retirement gate

1. 对每个路径运行 Active/Finished Work 反向引用扫描。
2. `delete-after-promotion` 必须指出承接原则的现行文档或 Knowledge。
3. 有 Finished Work 依赖的条目保留证据并修复引用；无依赖条目删除。
4. 检查 Work 必需文件和本地 Markdown 链接，确保恢复路径完整。
5. 主动 Knowledge 搜索不再返回 Container、Expr、yt、旧 Pin DTO 或 virtual marker 作为现行规则。
6. 退役变更与 R5 发布文档由同一 Git commit 留下历史记录。
