// useFolding 折叠选中节点为新子图 + 主图换 Subgraph 调用节点。
// 1in1out 时 (外部入边 + 外部出边 都唯一) 自动重连; 其他情况保留 dangling 提示手动.
import type { Ref, ComputedRef } from 'vue'
import type { GraphNode as VueFlowNode } from '@vue-flow/core'
import { backend, type Container, type Graph } from '@/lib/backend'
import { useConfirm } from '@/composables/useConfirm'

export function useFolding(opts: {
  draft: Ref<Container | null>
  activeGraph: ComputedRef<Graph | null>
  refreshSubgraphStore: () => Promise<void>
  syncFlowFromDraft: () => void
  getSelectedNodes: Ref<VueFlowNode[]>
  toast: { add: (o: Record<string, unknown>) => unknown }
}) {
  const { draft, activeGraph, refreshSubgraphStore, syncFlowFromDraft, getSelectedNodes, toast } = opts
  const { confirm } = useConfirm()

  async function onFoldSelection() {
    if (!draft.value) return
    const selected = getSelectedNodes.value
    const candidates = selected.filter(
      (n: any) => n.data.kind !== 'Start' && n.data.kind !== 'MouseCalibration',
    )
    if (candidates.length === 0) {
      toast.add({
        title: '选中里没有可折叠的节点（Start / MouseCalibration 不允许折叠）',
        color: 'warning',
      })
      return
    }
    // v2 Plan B Task C.4：用 ConfirmDialog 带 input 替换 window.prompt
    const labelResult = await confirm({
      title: '新建子图',
      description: '为折叠出来的子图取个名字',
      inputDefault: '折叠 ' + new Date().toLocaleTimeString().slice(0, 5),
      inputLabel: '子图名称',
      inputPlaceholder: '例如：上钩处理',
    })
    if (typeof labelResult !== 'string' || !labelResult.trim()) return
    const label = labelResult.trim()

    const sgRaw = (await backend.containers.createSubgraph(draft.value.id, label)) as any
    if (!sgRaw) {
      toast.add({ title: '创建子图失败', color: 'error' })
      return
    }
    const candidateIDs = new Set(candidates.map((n: any) => n.id))
    const movedNodes = candidates.map((n: any) => ({
      id: n.id,
      kind: n.data.kind,
      x: n.position?.x ?? 0,
      y: n.position?.y ?? 0,
      config: n.data.config ?? {},
      createdAt: new Date().toISOString(),
    }))
    const allEdges = activeGraph.value?.edges ?? []
    const movedEdges: any[] = []
    const externalIns: any[] = []
    const externalOuts: any[] = []
    for (const e of allEdges) {
      const fromID = e.from.split('.')[0]
      const toID = e.to.split('.')[0]
      if (candidateIDs.has(fromID) && candidateIDs.has(toID)) movedEdges.push(e)
      else if (candidateIDs.has(toID)) externalIns.push(e)
      else if (candidateIDs.has(fromID)) externalOuts.push(e)
    }
    if (externalIns.length > 1) {
      toast.add({ title: '多个外部入口暂不支持折叠（v1 限制）', color: 'warning' })
      return
    }

    // auto-connect 条件: 外部入边 == 1 + 外部出边 == 1
    // 此时确定唯一 boundary 节点对 (firstNode = 外部入边目标 / lastNode = 外部出边源)
    // 子图内补 SubgraphInput.out → firstNode.in 和 lastNode.out → SubgraphOutput.in
    // 主图原 external 边改写到新 Subgraph 调用节点
    const autoConnectable = externalIns.length === 1 && externalOuts.length === 1
    const firstNodeID = externalIns[0]?.to.split('.')[0]
    const lastNodeID = externalOuts[0]?.from.split('.')[0]
    const sgInNode = (sgRaw.graph.nodes as any[]).find((n) => n.kind === 'SubgraphInput')
    const sgOutNode = (sgRaw.graph.nodes as any[]).find((n) => n.kind === 'SubgraphOutput')
    const declID = sgRaw.outputPins?.[0]?.id as string | undefined

    const innerEdges: any[] = [...movedEdges]
    if (autoConnectable && sgInNode && sgOutNode && declID && firstNodeID && lastNodeID) {
      innerEdges.push({ from: `${sgInNode.id}.out`, to: `${firstNodeID}.in` })
      innerEdges.push({ from: `${lastNodeID}.out`, to: `${sgOutNode.id}.in` })
    }

    await backend.containers.updateSubgraph(draft.value.id, sgRaw.id, JSON.stringify({
      graph: {
        id: sgRaw.graph.id,
        version: sgRaw.graph.version,
        nodes: [...sgRaw.graph.nodes, ...movedNodes],
        edges: innerEdges,
      },
    }))

    let callNodeID = ''
    if (activeGraph.value) {
      callNodeID = 'n-call-' + Math.random().toString(36).slice(2, 8)
      activeGraph.value.nodes = activeGraph.value.nodes.filter((n: any) => !candidateIDs.has(n.id))
      // 仅删除内部边; external 边保留, 后续按 autoConnectable 改写或留 dangling 给用户手动修
      activeGraph.value.edges = activeGraph.value.edges.filter((e: any) => {
        const fromID = e.from.split('.')[0]
        const toID = e.to.split('.')[0]
        return !(candidateIDs.has(fromID) && candidateIDs.has(toID))
      })
      activeGraph.value.nodes.push({
        id: callNodeID,
        kind: 'Subgraph',
        x: candidates[0].position?.x ?? 100,
        y: candidates[0].position?.y ?? 100,
        config: { SubgraphID: sgRaw.id },
        createdAt: new Date().toISOString(),
      })

      if (autoConnectable && declID) {
        // 改写 external 边的 dangling 端指向新 call node
        for (const e of activeGraph.value.edges as any[]) {
          if (e === externalIns[0]) e.to = `${callNodeID}.in`
          else if (e === externalOuts[0]) e.from = `${callNodeID}.${declID}`
        }
      }
    }

    await refreshSubgraphStore()
    syncFlowFromDraft()

    toast.add({
      title: '已折叠 ' + candidates.length + ' 个节点为子图: ' + label,
      color: 'success',
    })
    if (autoConnectable) {
      toast.add({ title: '外部连线已自动重连到子图调用节点', color: 'primary', duration: 4000 })
    } else if (externalIns.length > 0 || externalOuts.length > 0) {
      toast.add({
        title: '注意：父图与子图之间的外部连线未自动重连，请手动检查',
        color: 'warning',
        duration: 8000,
      })
    }
  }

  return { onFoldSelection }
}
