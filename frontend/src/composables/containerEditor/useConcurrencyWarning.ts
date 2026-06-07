// useConcurrencyWarning Parallel / Race 节点的并发分支变量写入冲突检测.
// 一旦不同 branch 路径下都有 SetVar / IncVar 改写同名变量 → 结果不确定 → 警告.
//
// 抽自 NodeInspector: ContainerEditorView 也能用 (Inspector 之外的位置展示警告).
import { computed, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GraphNode, GraphEdge } from '@/lib/backend'

// BFS: 从 (startNodeID.startPin) 出发, 沿 edges 走到的所有节点 id
function reachable(startNodeID: string, startPin: string, allEdges: GraphEdge[]): string[] {
  const out = new Set<string>()
  const queue: string[] = []
  for (const e of allEdges) {
    if (e.from === `${startNodeID}.${startPin}`) {
      const to = e.to.split('.')[0]
      queue.push(to)
      out.add(to)
    }
  }
  while (queue.length > 0) {
    const cur = queue.shift()!
    for (const e of allEdges) {
      if (e.from.startsWith(cur + '.')) {
        const to = e.to.split('.')[0]
        if (!out.has(to)) {
          out.add(to)
          queue.push(to)
        }
      }
    }
  }
  return [...out]
}

export function useConcurrencyWarning(opts: {
  node: Ref<GraphNode | null>
  nodes: Ref<GraphNode[] | undefined>
  edges: Ref<GraphEdge[] | undefined>
}) {
  const { t } = useI18n()
  const concurrencyWarning = computed<string>(() => {
    const n = opts.node.value
    if (!n || (n.kind !== 'Parallel' && n.kind !== 'Race')) return ''
    const nodes = opts.nodes.value ?? []
    const edges = opts.edges.value ?? []
    const branchVars = new Map<number, Set<string>>()
    for (const e of edges) {
      if (!e.from.startsWith(n.id + '.branch')) continue
      const pin = e.from.slice(n.id.length + 1)
      const idx = Number(pin.replace('branch', ''))
      if (Number.isNaN(idx)) continue
      const reached = reachable(n.id, pin, edges)
      const vars = new Set<string>()
      for (const id of reached) {
        const node = nodes.find((x) => x.id === id)
        if (!node) continue
        if (node.kind === 'SetVar' || node.kind === 'IncVar') {
          // VarName pin 字面量 = config.literal.VarName (跟后端 + 真实存盘 shape 对齐)。
          const lit = node.config?.literal as Record<string, unknown> | undefined
          const v = (lit?.VarName as string | undefined) ?? ''
          if (v) vars.add(v)
        }
      }
      branchVars.set(idx, vars)
    }
    const allBranchVars = [...branchVars.values()]
    const conflicts = new Set<string>()
    for (let i = 0; i < allBranchVars.length; i++) {
      for (let j = i + 1; j < allBranchVars.length; j++) {
        for (const v of allBranchVars[i]) {
          if (allBranchVars[j].has(v)) conflicts.add(v)
        }
      }
    }
    if (conflicts.size === 0) return ''
    return t('editorAux.warning_concurrent_write', { nodes: [...conflicts].join(', ') })
  })

  return { concurrencyWarning }
}
