// 跨 graph (主图 + 所有子图) walker. useNodeSearch / useContextMenuRouter
// (find-references) 共用.
//
// Usage:
//   walkAllGraphs(container, (node, { location, sgID }) => {
//     // location = '主图' | '子图: <label || id>'
//     // sgID    = null (主图) | sg.id
//   })

import { useI18n } from 'vue-i18n'
import type { Container, GraphNode, Subgraph } from '@/lib/backend'

export interface GraphVisitCtx {
  /** UI-display 路径串 — '主图' 或 '子图: <label || id>' */
  location: string
  /** null = 主图; 否则 = 子图 ID. */
  sgID: string | null
}

// 子图已全局化: 容器不再带 subgraphs 字段, 由 caller 传要遍历的子图集
// (一般 = editorStore.subgraphList 全池).
export function walkAllGraphs(
  container: Container,
  subgraphs: Subgraph[],
  visit: (node: GraphNode, ctx: GraphVisitCtx) => void,
): void {
  const { t } = useI18n()
  for (const n of container.graph.nodes) {
    visit(n, { location: t('editorAux.root_graph'), sgID: null })
  }
  for (const sg of subgraphs) {
    if (!sg.graph) continue
    const location = t('editorAux.subgraph_label', { name: sg.label || sg.id })
    for (const n of sg.graph.nodes) {
      visit(n, { location, sgID: sg.id })
    }
  }
}
