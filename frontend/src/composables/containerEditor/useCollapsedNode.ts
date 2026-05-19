// CollapsedNode composable — scaffold.
//
// 当前是 stub: 三个 op 都 console.warn no-op. CollapsedNode runtime 工作 (validators 已 ship),
// 但用户必须手写 JSON: container.subgraphs[] 含 isAnonymous:true Subgraph + main graph
// CollapsedNode 引用 config.subgraphId.
//
// 实装 backlog 见 debug/docs/superpowers/specs/2026-05-20-editor-v2-backlog.md (相关条目).
// 需要的后端 RPC: backend.containers.createSubgraph / deleteSubgraph / updateSubgraphMeta.
import type { Ref } from 'vue'

interface CollapseDeps {
  draft: Ref<any>
  activeGraph: Ref<any>
  syncFlowFromDraft: () => void
  refreshSubgraphStore?: () => Promise<void>
}

function warnOnce(msg: string) {
  // eslint-disable-next-line no-console
  console.warn('[useCollapsedNode]', msg)
}

export function useCollapsedNode(_deps: CollapseDeps) {
  // Collapse 选中节点为 CollapsedNode + isAnonymous Subgraph.
  async function collapse(_selectedNodeIDs: string[]): Promise<void> {
    warnOnce('collapse: not yet implemented; 手写 JSON 走 isAnonymous Subgraph')
  }

  // Expand 反操作 — 把 CollapsedNode 内部节点搬回父图, 删 backing Subgraph.
  async function expand(_collapsedNodeID: string): Promise<void> {
    warnOnce('expand: not yet implemented')
  }

  // Upgrade — 把 backing Subgraph.isAnonymous 翻 false + 节点 kind 改 Subgraph, 变成可复用子图.
  async function upgradeToSubgraph(_collapsedNodeID: string): Promise<void> {
    warnOnce('upgradeToSubgraph: not yet implemented')
  }

  return { collapse, expand, upgradeToSubgraph }
}
