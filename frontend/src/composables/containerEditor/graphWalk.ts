// 跨 graph (主图 + 所有子图) walker. 从 useNodeSearch / useContextMenuRouter
// (find-references) 共用模板抽 (backlog C5).
//
// Usage:
//   walkAllGraphs(container, (node, { location, sgID }) => {
//     // location = '主图' | '子图: <label || id>'
//     // sgID    = null (主图) | sg.id
//   })

import { useI18n } from 'vue-i18n'
import type { Container, GraphNode } from '@/lib/backend'

export interface GraphVisitCtx {
  /** UI-display 路径串 — '主图' 或 '子图: <label || id>' */
  location: string
  /** null = 主图; 否则 = 子图 ID. */
  sgID: string | null
}

export function walkAllGraphs(
  container: Container,
  visit: (node: GraphNode, ctx: GraphVisitCtx) => void,
): void {
  const { t } = useI18n()
  for (const n of container.graph.nodes) {
    visit(n, { location: t('editorAux.root_graph'), sgID: null })
  }
  for (const sg of container.subgraphs ?? []) {
    if (!sg.graph) continue
    const location = t('editorAux.subgraph_label', { name: sg.label || sg.id })
    for (const n of sg.graph.nodes) {
      visit(n, { location, sgID: sg.id })
    }
  }
}
