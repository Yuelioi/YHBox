import { computed, onMounted, ref, watch } from 'vue'
import { MarkerType } from '@vue-flow/core'
import { backend, type Container, type Graph, type GraphNode, type GraphEdge } from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { edgeKind } from '@/components/containers/pinSpec'

// FlowNode / FlowEdge: vue-flow 渲染数据形态 (与后端 GraphNode/GraphEdge 区分)
export interface FlowNode {
  id: string
  type: string
  position: { x: number; y: number }
  data: { kind: string; config?: Record<string, any> }
}
export interface FlowEdge {
  id: string
  source: string
  target: string
  sourceHandle?: string
  targetHandle?: string
  type?: string
  animated?: boolean
  style?: Record<string, any>
  markerEnd?: any
}

/**
 * useContainerDraft 容器编辑器数据中心。
 *  - draft: 本地编辑副本
 *  - dirty: 未保存改动标记
 *  - activeGraph: 跟随 editorPath 切换的当前层级 graph
 *  - flowNodes / flowEdges: vue-flow 渲染数据
 *  - syncFlowFromDraft(): draft → flow 渲染数据
 *  - refreshSubgraphStore(): 后端 listSubgraphs 同步到 store
 */
export function useContainerDraft(containerID: string) {
  const editorStore = useContainerEditorStore()

  const draft = ref<Container | null>(null)
  const dirty = ref(false)
  const flowNodes = ref<FlowNode[]>([])
  const flowEdges = ref<FlowEdge[]>([])

  // activeGraph: 跟随 editorPath 切换. 空 path → 主图; 否则 = 当前 path 末尾对应子图的 graph.
  // 返回 Graph | null, 不带 any: 下游 mutation 必须显式 null check.
  const activeGraph = computed<Graph | null>(() => {
    if (!draft.value) return null
    if (editorStore.editorPath.length === 0) return draft.value.graph
    const sgs = editorStore.subgraphsForCurrentContainer
    let cur = sgs.find((s) => s.id === editorStore.editorPath[0])
    if (!cur) return null
    for (let i = 1; i < editorStore.editorPath.length; i++) {
      // 嵌套子图: 当前层级是 cur.graph 的内部 (cur 本身就是 SubgraphSummary), 但 editorPath
      // 表示层级 id 列表, 嵌套需要在 cur 的子图列表里继续找 — v2 1:1 模型下子图全部平铺在容器
      // subgraphsForCurrentContainer, editorPath 每一段都 lookup 同一个平铺列表.
      cur = sgs.find((s) => s.id === editorStore.editorPath[i])
      if (!cur) return null
    }
    return (cur?.graph as Graph) ?? null
  })

  function syncFlowFromDraft() {
    if (!draft.value) return
    const g = activeGraph.value
    if (!g) {
      flowNodes.value = []
      flowEdges.value = []
      return
    }
    flowNodes.value = g.nodes.map((n: GraphNode) => ({
      id: n.id,
      type: n.kind,
      position: { x: n.x, y: n.y },
      data: { kind: n.kind, config: n.config },
    }))
    flowEdges.value = g.edges.map((e: GraphEdge, i: number) => {
      const dot = e.from.indexOf('.')
      const src = e.from.slice(0, dot)
      const srcPin = e.from.slice(dot + 1)
      const dot2 = e.to.indexOf('.')
      const tgt = e.to.slice(0, dot2)
      const tgtPin = e.to.slice(dot2 + 1)
      const fromKind = g.nodes.find((n) => n.id === src)?.kind ?? ''
      const isData = edgeKind(fromKind, srcPin) === 'data'
      return {
        id: 'e' + i,
        source: src,
        target: tgt,
        sourceHandle: srcPin,
        targetHandle: tgtPin,
        type: 'smoothstep',
        animated: isData,
        style: isData
          ? { stroke: '#60a5fa', strokeWidth: 1.5, strokeDasharray: '4 4' } // data edge: dashed blue
          : { stroke: '#a1a1aa', strokeWidth: 1.5 }, // exec edge: solid zinc
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: isData ? '#60a5fa' : '#a1a1aa',
        },
      }
    })
  }

  // editorPath 变化时重新 sync（进/出子图层级）— 不参与 dirty 判定，可立即装
  watch(
    () => editorStore.editorPath,
    () => {
      syncFlowFromDraft()
    },
    { deep: true },
  )

  // dirty watcher 推迟到 onMounted load 完成后才装 (见末尾 installDirtyWatchers)
  // 原方案: watch 在 setup 阶段立即装 + setTimeout(0) 重置初始 fire — 依赖 microtask 顺序, brittle.
  // 新方案: load 完毕显式装 watcher, 初始化期间根本不触发, 不需要重置 hack.

  onMounted(async () => {
    if (!containerID) return
    const r = await backend.containers.get(containerID)
    if (r === undefined) {
      console.error('container not found:', containerID)
      return
    }
    const c = r as unknown as Container
    draft.value = JSON.parse(JSON.stringify(c))
    // 把子图列表同步给 store（NodeInspector / ContainerFlowNode + activeGraph computed 用）
    // ⚠ 必须把完整 subgraph 数据传进去（含 graph）；只传 summary 会让双击进子图后画布空白。
    try {
      const sgList = (await backend.containers.listSubgraphs(containerID)) as any[]
      editorStore.setActiveContainer(
        containerID,
        (sgList ?? []).map((s) => ({
          id: s.id,
          label: s.label,
          outputPins: s.outputPins ?? [],
          graph: s.graph ?? { id: '', version: 1, nodes: [], edges: [] },
          description: s.description,
          recordingContext: s.recordingContext,
          tags: s.tags,
        })),
      )
    } catch (e) {
      console.warn('listSubgraphs failed', e)
      editorStore.setActiveContainer(containerID, [])
    }
    syncFlowFromDraft()
    // load 完毕后才装 dirty watcher; 初始化阶段不触发 → 不再需要 setTimeout 重置 hack
    watch(draft, () => { dirty.value = true }, { deep: true })
    watch(
      () => editorStore.subgraphsForCurrentContainer,
      () => { dirty.value = true },
      { deep: true },
    )
  })

  /** 重新从后端拉最新子图列表并同步 store */
  async function refreshSubgraphStore() {
    if (!draft.value) return
    try {
      const fresh = (await backend.containers.listSubgraphs(draft.value.id)) as any[]
      editorStore.setActiveContainer(
        draft.value.id,
        (fresh ?? []).map((s) => ({
          id: s.id,
          label: s.label,
          outputPins: s.outputPins ?? [],
          graph: s.graph ?? { id: '', version: 1, nodes: [], edges: [] },
          description: s.description,
          recordingContext: s.recordingContext,
          tags: s.tags,
        })),
      )
    } catch (e) {
      console.warn('refreshSubgraphStore failed', e)
    }
  }

  return {
    draft,
    dirty,
    activeGraph,
    flowNodes,
    flowEdges,
    syncFlowFromDraft,
    refreshSubgraphStore,
  }
}
