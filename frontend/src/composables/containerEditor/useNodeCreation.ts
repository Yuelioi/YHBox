// 节点创建 pipeline — 9 处 push 各自 ID + defaults + auto-wire 决策抽这里.
// 从 ContainerEditorView 抽 (backlog C1).
//
// 范围:
//   drop* — drag-and-drop 入口 (var / node-spec / snippet)
//   onInsertIncVar — VarRow "+" 按钮 中心 insert IncVar
//   onApplySnippet — SnippetsPanel 单击 snippet 中心 insert
//   onPickKind — NodeExplorerModal 选 kind
//   onPickLibrarySubgraph — LibraryExplorerModal 选 library import
//   onAddNode — programmatic add (used by useFlowInteraction + onRecord 完成)
//
// 注: C6 "节点创建 pipeline 统一" 是这步的 follow-up — 将所有 push 统一到单个
//     `addNode({kind, pos, configOverrides?, autoConnect?})` 接口. 这一步只抽 view-side.

import { type Ref } from 'vue'
import { useVueFlow } from '@vue-flow/core'
import { useSnippetsStore, type Snippet } from '@/stores/snippets'
import { useLibraryStore } from '@/stores/library'
import { backend, type Container, type Graph, type GraphNode, type GraphEdge, type VarDecl } from '@/lib/backend'
import { dataInTypeFor, getSpec } from '@/components/containers/nodeRegistry/registry'
import { isCompatibleType, type VarType } from '@/lib/variableRef'
import { KIND_DEFAULTS } from '@/components/containers/pinSpec'
import { type EditorDragPayload } from '@/composables/editor/useEditorDragDrop'
import { newNodeID, genNodeID } from './ids'
import { AUTO_CONNECT_THRESHOLD_FLOW_PX } from './constants'

type ToastApi = { add: (opts: { title: string; description?: string; color?: string; icon?: string }) => void }

interface PinAtPosition {
  nodeID: string
  pinName: string
  pinType: VarType
  dist: number
}

interface UseNodeCreationOpts {
  draft: Ref<Container | null>
  activeGraph: Ref<Graph | null>
  selectedID: Ref<string | null>
  applyDraftMutation: (mutator: (draft: Container) => void) => void
  syncFlowFromDraft: () => void
  refreshSubgraphStore: () => Promise<void>
  autoCreateSubgraphForNewNode: (n: GraphNode) => Promise<boolean>
  toast: ToastApi
}

export function useNodeCreation(opts: UseNodeCreationOpts) {
  const {
    draft, activeGraph, selectedID,
    applyDraftMutation, syncFlowFromDraft, refreshSubgraphStore,
    autoCreateSubgraphForNewNode, toast,
  } = opts
  const { screenToFlowCoordinate } = useVueFlow()

  function defaultLiteralFor(type: VarDecl['type']): unknown {
    switch (type) {
      case 'number': return 0
      case 'string': return ''
      case 'bool': return false
      case 'point': return { x: 0.5, y: 0.5 }
      default: return null  // 'any' — no useful default
    }
  }

  // Data-in handles use Position.Bottom + type="target" → CSS selector
  // `.vue-flow__handle[data-handlepos="bottom"].target` isolates them
  // (exec-in uses Position.Left, 排除).
  function findNearestEligibleDataInPin(
    flowPos: { x: number; y: number },
    srcVarType: VarType,
  ): PinAtPosition | null {
    const handles = document.querySelectorAll<HTMLElement>(
      '.vue-flow__handle[data-handlepos="bottom"].target',
    )
    let best: PinAtPosition | null = null

    for (const handleEl of Array.from(handles)) {
      const nodeID = handleEl.getAttribute('data-nodeid') ?? ''
      const pinName = handleEl.getAttribute('data-handleid') ?? ''
      if (!nodeID || !pinName) continue

      const node = activeGraph.value?.nodes?.find((n) => n.id === nodeID) ?? null
      if (!node) continue
      if (node.disabled === true) continue  // disabled 不参与 auto-connect

      const pinType = dataInTypeFor(node.kind, pinName, node.config as Record<string, unknown>)
      if (!pinType) continue
      if (!isCompatibleType(srcVarType, pinType as VarType)) continue

      const rect = handleEl.getBoundingClientRect()
      const screenCenter = { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
      const flowCenter = screenToFlowCoordinate(screenCenter)
      const dx = flowCenter.x - flowPos.x
      const dy = flowCenter.y - flowPos.y
      const dist = Math.sqrt(dx * dx + dy * dy)
      if (dist > AUTO_CONNECT_THRESHOLD_FLOW_PX) continue

      if (!best || dist < best.dist) {
        best = { nodeID, pinName, pinType: pinType as VarType, dist }
      }
    }
    return best
  }

  function dropVar(
    payload: Extract<EditorDragPayload, { type: 'var' }>,
    pos: { x: number; y: number },
  ) {
    const kind = payload.modifier === 'alt' ? 'SetVar' : 'GetVar'
    const config: Record<string, unknown> = { varName: payload.ref.name, scope: 'auto' }
    if (kind === 'SetVar') {
      config.literal = { value: defaultLiteralFor(payload.ref.type) }
    }

    // Pin-aware auto-connect: DOM query 必须在 applyDraftMutation 触发 re-render 前
    const autoConnectTarget = kind === 'GetVar'
      ? findNearestEligibleDataInPin(pos, payload.ref.type as VarType)
      : null

    applyDraftMutation(() => {
      const g = activeGraph.value
      if (!g) return
      const node: GraphNode = {
        id: newNodeID(kind),
        kind,
        x: pos.x,
        y: pos.y,
        config,
        createdAt: new Date().toISOString(),
      } as GraphNode
      g.nodes.push(node)

      if (autoConnectTarget) {
        ;(g.edges as GraphEdge[]).push({
          from: `${node.id}.value`,
          to: `${autoConnectTarget.nodeID}.${autoConnectTarget.pinName}`,
        } as GraphEdge)
      }
    })
  }

  function dropNodeSpec(
    payload: Extract<EditorDragPayload, { type: 'node-spec' }>,
    pos: { x: number; y: number },
  ) {
    const kind = payload.kind
    applyDraftMutation(() => {
      const g = activeGraph.value
      if (!g) return
      const node: GraphNode = {
        id: newNodeID(kind),
        kind,
        x: pos.x,
        y: pos.y,
        config: getSpec(kind)?.defaults ?? {},
        createdAt: new Date().toISOString(),
      } as GraphNode
      g.nodes.push(node)
    })
  }

  // snippet 拖到画布 → 在 pos 生成节点 with snippet.payload.config
  function dropSnippet(
    payload: Extract<EditorDragPayload, { type: 'snippet' }>,
    pos: { x: number; y: number },
  ) {
    const s = useSnippetsStore().getById(payload.snippetID)
    if (!s) return
    applyDraftMutation(() => {
      const g = activeGraph.value
      if (!g) return
      const node: GraphNode = {
        id: newNodeID(s.payload.kind),
        kind: s.payload.kind,
        x: pos.x,
        y: pos.y,
        config: JSON.parse(JSON.stringify(s.payload.config)),
        label: s.name, // snippet 名当 label, 一眼看出是哪个实例
        createdAt: new Date().toISOString(),
      } as GraphNode
      g.nodes.push(node)
    })
    useSnippetsStore().markUsed(payload.snippetID)
  }

  // VarRow "+" 按钮 → viewport 中心 insert IncVar
  function onInsertIncVar(name: string) {
    const center = screenToFlowCoordinate({
      x: window.innerWidth / 2,
      y: window.innerHeight / 2,
    })
    applyDraftMutation(() => {
      const g = activeGraph.value
      if (!g) return
      const node: GraphNode = {
        id: newNodeID('IncVar'),
        kind: 'IncVar',
        x: center.x,
        y: center.y,
        config: {
          varName: name,
          scope: 'auto',
          literal: { delta: 1 },
        },
        createdAt: new Date().toISOString(),
      } as GraphNode
      g.nodes.push(node)
    })
  }

  // SnippetsPanel 单击 snippet (非 drag) → 画布中心生成节点 with config
  function onApplySnippet(s: Snippet) {
    applyDraftMutation(() => {
      const g = activeGraph.value
      if (!g) return
      // 画布中心 fallback (没 viewport center API 时用固定 offset)
      const pos = { x: 240 + Math.random() * 60, y: 180 + Math.random() * 60 }
      const node: GraphNode = {
        id: newNodeID(s.payload.kind),
        kind: s.payload.kind,
        x: pos.x,
        y: pos.y,
        config: JSON.parse(JSON.stringify(s.payload.config)),
        label: s.name,
        createdAt: new Date().toISOString(),
      } as GraphNode
      g.nodes.push(node)
    })
    useSnippetsStore().markUsed(s.id)
  }

  // NodeExplorerModal 选 kind
  function onPickKind(kind: string) {
    applyDraftMutation(() => {
      const g = activeGraph.value
      if (!g) return
      const node: GraphNode = {
        id: newNodeID(kind),
        kind,
        x: 200 + Math.random() * 100,
        y: 200 + Math.random() * 100,
        config: { ...(getSpec(kind)?.defaults ?? {}) },
        createdAt: new Date().toISOString(),
      } as GraphNode
      g.nodes.push(node)
    })
  }

  // LibraryExplorerModal 选 library — 导入子图 + 加 Subgraph 引用节点
  async function onPickLibrarySubgraph(libraryID: string) {
    if (!draft.value) return
    try {
      await backend.library.importToContainer(libraryID, draft.value.id, '')
      const newSubgraphID = libraryID
      await refreshSubgraphStore()
      applyDraftMutation(() => {
        const g = activeGraph.value
        if (!g) return
        const node: GraphNode = {
          id: newNodeID('Subgraph'),
          kind: 'Subgraph',
          x: 200 + Math.random() * 100,
          y: 200 + Math.random() * 100,
          config: { SubgraphID: newSubgraphID },
          createdAt: new Date().toISOString(),
        } as GraphNode
        g.nodes.push(node)
      })
      useLibraryStore().reload()
      toast.add({
        title: '子图已从库导入',
        description: `SubgraphID: ${newSubgraphID}`,
        color: 'success',
        icon: 'i-tabler-check',
      })
    } catch (e: any) {
      console.error('LibraryExplorer pick failed:', e)
      toast.add({
        title: '从库导入失败',
        description: String(e?.message ?? e),
        color: 'error',
      })
    }
  }

  // Programmatic add (useFlowInteraction palette drop / 录制完成 etc.).
  // v2 Plan B: push 到 activeGraph (主图/子图层级), 不是固定主图. Start 单例.
  async function onAddNode(
    kind: string,
    atX?: number,
    atY?: number,
  ): Promise<string | null> {
    if (!draft.value) return null
    const targetGraph = activeGraph.value
    if (!targetGraph) {
      toast.add({ title: '当前层级 graph 不可用', color: 'error' })
      return null
    }
    const id = kind === 'Start' ? 'start' : genNodeID()
    if (kind === 'Start' && targetGraph.nodes.some((n) => n.kind === 'Start')) {
      toast.add({ title: '只能有一个 Start 节点', color: 'warning' })
      return null
    }
    const x = atX ?? 200 + Math.random() * 200
    const y = atY ?? 100 + Math.random() * 200
    const defaults = KIND_DEFAULTS[kind] ?? {}
    const n: GraphNode = { id, kind, x, y, config: { ...defaults } }

    if (kind === 'Subgraph') {
      const ok = await autoCreateSubgraphForNewNode(n)
      if (!ok) {
        toast.add({ title: '建子图失败，请重试', color: 'error' })
        return null
      }
    }

    targetGraph.nodes.push(n)
    syncFlowFromDraft()
    selectedID.value = id
    return id
  }

  return {
    defaultLiteralFor,
    findNearestEligibleDataInPin,
    dropVar, dropNodeSpec, dropSnippet,
    onInsertIncVar, onApplySnippet,
    onPickKind, onPickLibrarySubgraph,
    onAddNode,
  }
}
