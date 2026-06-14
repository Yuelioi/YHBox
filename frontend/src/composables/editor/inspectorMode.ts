// frontend/src/composables/editor/inspectorMode.ts
// 右侧 Inspector 三态决策 (spec B §3)。纯函数 — 无 Vue 依赖, 可直接单测。
export type InspectorMode = 'node' | 'subgraph' | 'collapsed'

export function resolveInspectorMode(args: {
  hasSelectedNode: boolean
  inSubgraph: boolean
}): InspectorMode {
  if (args.hasSelectedNode) return 'node' // 选中节点 → 节点属性
  if (args.inSubgraph) return 'subgraph' // 子图内空选 → 保留 SubgraphPropsPanel
  return 'collapsed' // 根图空选 → 自动收起, 画布全宽
}
