// useFolding 折叠选中节点为新子图 + 主图换 Subgraph 调用节点。
// 1in1out 时 (外部入边 + 外部出边 都唯一) 自动重连; 其他情况保留 dangling 提示手动.
import type { Ref, ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GraphNode as VueFlowNode } from '@vue-flow/core'
import { backend, type Container, type Graph } from '@/lib/backend'
import { useConfirm } from '@/composables/useConfirm'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { nextSubgraphName } from '@/lib/subgraphNaming'
import { randID } from './ids'

export function useFolding(opts: {
  draft: Ref<Container | null>
  activeGraph: ComputedRef<Graph | null>
  refreshSubgraphStore: () => Promise<void>
  syncFlowFromDraft: () => void
  getSelectedNodes: Ref<VueFlowNode[]>
  toast: { add: (o: Record<string, unknown>) => unknown }
}) {
  const { draft, activeGraph, refreshSubgraphStore, syncFlowFromDraft, getSelectedNodes, toast } =
    opts
  const { confirm } = useConfirm()
  const editorStore = useContainerEditorStore()
  const { t } = useI18n()

  async function onFoldSelection() {
    if (!draft.value) return
    const selected = getSelectedNodes.value
    const candidates = selected.filter(
      (n: any) => n.data.kind !== 'Start' && n.data.kind !== 'MouseCalibration',
    )
    if (candidates.length === 0) {
      toast.add({
        title: t('folding.no_foldable'),
        color: 'warning',
      })
      return
    }
    // v2 Plan B Task C.4：用 ConfirmDialog 带 input 替换 window.prompt
    const labelResult = await confirm({
      title: t('folding.new_subgraph_title'),
      description: t('folding.new_subgraph_desc'),
      inputDefault: nextSubgraphName(
        editorStore.visibleSubgraphs.map((s) => s.label),
        t('subgraphLifecycle.default_name_prefix'),
      ),
      inputLabel: t('folding.subgraph_name_label'),
      inputPlaceholder: t('folding.subgraph_name_placeholder'),
    })
    if (typeof labelResult !== 'string' || !labelResult.trim()) return
    const label = labelResult.trim()

    const sgRaw = (await backend.subgraphs.create(label)) as any
    if (!sgRaw) {
      toast.add({ title: t('folding.create_failed'), color: 'error' })
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
      toast.add({ title: t('folding.multi_entry_unsupported'), color: 'warning' })
      return
    }

    // auto-connect 条件: 外部入边 == 1 + 外部出边 == 1
    // 此时确定唯一 boundary 节点对 (firstNode = 外部入边目标 / lastNode = 外部出边源)
    // 子图内补 entry.out → firstNode.in 和 lastNode.out → outputPin.in
    // 主图原 external 边改写到新 Subgraph 调用节点
    // B2: entry / output 是 virtual marker (sg.entry.nodeID / sg.outputPins[0].nodeID), 不再在 graph.nodes.
    const autoConnectable = externalIns.length === 1 && externalOuts.length === 1
    const firstNodeID = externalIns[0]?.to.split('.')[0]
    const lastNodeID = externalOuts[0]?.from.split('.')[0]
    const sgEntryNodeID = (sgRaw as any).entry?.nodeID as string | undefined
    const sgOutPin = sgRaw.outputPins?.[0]
    const sgOutNodeID = (sgOutPin as any)?.nodeID as string | undefined
    const declID = sgOutPin?.id as string | undefined

    const innerEdges: any[] = [...movedEdges]
    if (autoConnectable && sgEntryNodeID && sgOutNodeID && declID && firstNodeID && lastNodeID) {
      // pin 名走真实约定, 不硬编码 in/out:
      // - entry marker exec-out = "Done" (runtime 从 entryID+".Done" 播种)
      // - output marker exec-in = "In" (SubgraphOutput.execIn)
      // - 边界节点端保留原 external 边的真实 pin (e.g. Win32WindowTarget 的 In/Fire), 不重建成 in/out
      innerEdges.push({ from: `${sgEntryNodeID}.Done`, to: externalIns[0].to })
      innerEdges.push({ from: externalOuts[0].from, to: `${sgOutNodeID}.In` })
    }

    // 刚 create 的子图 rev=1, 乐观锁基准直接用返回值的 rev。
    await backend.subgraphs.update(
      sgRaw.id,
      JSON.stringify({
        graph: {
          id: sgRaw.graph.id,
          schemaVersion: sgRaw.graph.schemaVersion,
          nodes: [...sgRaw.graph.nodes, ...movedNodes],
          edges: innerEdges,
        },
      }),
      sgRaw.rev ?? 1,
    )
    const savedSubgraph = await backend.subgraphs.get(sgRaw.id)
    if (savedSubgraph) editorStore.replaceSubgraph(savedSubgraph)

    let callNodeID = ''
    if (activeGraph.value) {
      callNodeID = randID('n-call')
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
          if (e === externalIns[0])
            e.to = `${callNodeID}.In` // Subgraph 调用节点 exec-in = "In"
          else if (e === externalOuts[0]) e.from = `${callNodeID}.${declID}`
        }
      }
    }

    await refreshSubgraphStore()
    syncFlowFromDraft()

    if (autoConnectable) {
      toast.add({ title: t('folding.auto_reconnected'), color: 'primary', duration: 4000 })
    } else if (externalIns.length > 0 || externalOuts.length > 0) {
      toast.add({
        title: t('folding.manual_check_needed'),
        color: 'warning',
        duration: 8000,
      })
    }
  }

  return { onFoldSelection }
}
