// 节点创建 pipeline. 所有 9 callsite 统一走 addNode({kind, pos, config?, label?, id?,
// connectEdge?}) 单接口 (C6). 内部 buildNode + applyDraftMutation 一致.
//
// callsite:
//   drop* — drag-and-drop 入口 (var / node-spec / snippet)
//   onInsertIncVar — VarRow "+" 按钮 中心 insert IncVar
//   onApplySnippet — SnippetsPanel 单击 snippet 中心 insert
//   onPickKind — NodeExplorerModal 选 kind
//   onPickLibrarySubgraph — LibraryExplorerModal 选 library import
//   onPickLibraryClip — ClipExplorerModal 选 clip → 插裸 PlayClip 引用节点
//   onAddNode — programmatic add (录制完成等程序化加点).
//              特殊: 不走 applyDraftMutation (直接 push + syncFlowFromDraft),
//              Start 单例 guard, autoCreateSubgraphForNewNode 前置 hook.

import { type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVueFlow } from '@vue-flow/core'
import { useSnippetsStore, type Snippet } from '@/stores/snippets'
import { useLibraryStore } from '@/stores/library'
import { type Container, type Graph, type GraphNode, type GraphEdge, type VarDecl } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { dataInTypeFor, getSpec } from '@/components/containers/nodeRegistry/registry'
import { isCompatibleType, type VarType } from '@/lib/variableRef'
import { KIND_DEFAULTS } from '@/components/containers/pinSpec'
import { type EditorDragPayload } from '@/composables/editor/useEditorDragDrop'
import { newNodeID, genNodeID } from './ids'
import { AUTO_CONNECT_THRESHOLD_FLOW_PX } from './constants'
import { useInsertPoint } from './useInsertPoint'

type ToastApi = { add: (opts: { title: string; description?: string; color?: string; icon?: string }) => void }

interface PinAtPosition {
  nodeID: string
  pinName: string
  pinType: VarType
  dist: number
}

interface BuildNodeOpts {
  kind: string
  pos: { x: number; y: number }
  config?: Record<string, unknown>
  label?: string
  /** 默认 newNodeID(kind); onAddNode 用 'start' 或 genNodeID 覆盖 */
  id?: string
}

interface AddNodeOpts extends BuildNodeOpts {
  /** 同 mutation 内自动加边 — 接 new node, 返 edge 或 null. */
  connectEdge?: (newNode: GraphNode) => GraphEdge | null
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
  const { viewportCenterForNode, screenPointToFlow } = useInsertPoint()
  const { t } = useI18n()

  // ===== 核心 helper =====

  function buildNode(o: BuildNodeOpts): GraphNode {
    const n: GraphNode = {
      id: o.id ?? newNodeID(o.kind),
      kind: o.kind,
      x: o.pos.x,
      y: o.pos.y,
      config: o.config ?? {},
      createdAt: new Date().toISOString(),
    } as GraphNode
    if (o.label !== undefined) (n as any).label = o.label
    return n
  }

  /** 统一入口: applyDraftMutation + push + optional connectEdge. 返回 new node 或 null. */
  function addNode(o: AddNodeOpts): GraphNode | null {
    let result: GraphNode | null = null
    applyDraftMutation(() => {
      const g = activeGraph.value
      if (!g) return
      const node = buildNode(o)
      g.nodes.push(node)
      if (o.connectEdge) {
        const edge = o.connectEdge(node)
        if (edge) (g.edges as GraphEdge[]).push(edge)
      }
      result = node
    })
    return result
  }

  // ===== Utilities for callers =====

  function defaultLiteralFor(type: VarDecl['type']): unknown {
    switch (type) {
      case 'number': return 0
      case 'string': return ''
      case 'bool': return false
      case 'point': return { x: 0.5, y: 0.5 }
      case 'list': return []
      default: return null  // 'any' — no useful default
    }
  }

  // Data-in handles use Position.Bottom + type="target" → CSS selector
  // `.vue-flow__handle[data-handlepos="bottom"].target` isolates them.
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
      if (node.disabled === true) continue

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

  // ===== 8 callsite — 全走 addNode =====

  function dropVar(
    payload: Extract<EditorDragPayload, { type: 'var' }>,
    pos: { x: number; y: number },
  ) {
    const kind = payload.modifier === 'alt' ? 'SetVar' : 'GetVar'
    // VarName/Scope/Value 是 pin 字面量 → config.literal[PinName] (大写, 跟后端 + 真实存盘 shape 对齐)。
    const literal: Record<string, unknown> = { VarName: payload.ref.name, Scope: 'auto' }
    if (kind === 'SetVar') {
      literal.Value = defaultLiteralFor(payload.ref.type)
    }
    const config: Record<string, unknown> = { literal }
    // Pin-aware auto-connect: DOM query 必须在 applyDraftMutation 触发 re-render 前
    const autoConnectTarget = kind === 'GetVar'
      ? findNearestEligibleDataInPin(pos, payload.ref.type as VarType)
      : null

    addNode({
      kind, pos, config,
      connectEdge: autoConnectTarget
        ? (node) => ({
            from: `${node.id}.Value`,
            to: `${autoConnectTarget.nodeID}.${autoConnectTarget.pinName}`,
          } as GraphEdge)
        : undefined,
    })
  }

  function dropNodeSpec(
    payload: Extract<EditorDragPayload, { type: 'node-spec' }>,
    pos: { x: number; y: number },
  ) {
    addNode({
      kind: payload.kind,
      pos,
      config: getSpec(payload.kind)?.defaults ?? {},
    })
  }

  // snippet 拖到画布 → pos 生成节点 with snippet.payload.config
  function dropSnippet(
    payload: Extract<EditorDragPayload, { type: 'snippet' }>,
    pos: { x: number; y: number },
  ) {
    const s = useSnippetsStore().getById(payload.snippetID)
    if (!s) return
    addNode({
      kind: s.payload.kind,
      pos,
      config: JSON.parse(JSON.stringify(s.payload.config)),
      label: s.name, // snippet 名当 label
    })
    useSnippetsStore().markUsed(payload.snippetID)
  }

  // VarRow "+" 按钮 → 当前视口中心 insert IncVar
  function onInsertIncVar(name: string) {
    addNode({
      kind: 'IncVar',
      pos: viewportCenterForNode(),
      config: { literal: { VarName: name, Scope: 'auto', Delta: 1 } },
    })
  }

  // SnippetsPanel 单击 snippet → 当前视口中心生成 with config
  function onApplySnippet(s: Snippet) {
    addNode({
      kind: s.payload.kind,
      pos: viewportCenterForNode(),
      config: JSON.parse(JSON.stringify(s.payload.config)),
      label: s.name,
    })
    useSnippetsStore().markUsed(s.id)
  }

  // NodeExplorerModal 选 kind. screenPos = 唤起 explorer 那刻的鼠标屏幕坐标
  // (view 在 explorer 打开时快照 lastMousePos); 给了就落在鼠标附近, 没给落当前视口中心。
  function onPickKind(kind: string, screenPos?: { x: number; y: number }) {
    const pos = screenPos
      ? screenPointToFlow(screenPos)
      : viewportCenterForNode()
    addNode({
      kind,
      pos,
      config: { ...getSpec(kind)?.defaults },
    })
  }

  // LibraryExplorerModal 选子图 — 全局化后选中即插入引用节点 (无导入复制)。
  // 缺变量检测: 该子图闭包的 requiredGlobals 名字 − 容器已声明 → 自动补 (type any),
  // toast 告知 (老 import 流程的 auto-add 同款体验, 校验层另有 SUBGRAPH_VAR_UNDECLARED 兜底)。
  async function onPickLibrarySubgraph(libraryID: string) {
    if (!draft.value) return
    try {
      await refreshSubgraphStore()
      const lib = useLibraryStore()
      if (lib.pool.length === 0) await lib.reload()
      const declared = new Set((draft.value.vars ?? []).map((v) => v.name))
      const missing = lib.missingGlobalsFor(libraryID, declared)
      if (missing.length > 0) {
        if (!draft.value.vars) draft.value.vars = []
        for (const name of missing) {
          draft.value.vars.push({ name, type: 'any' })
        }
        toast.add({
          title: t('nodeCreation.auto_added_vars', { n: missing.length, names: missing.join(', ') }),
          color: 'info',
        })
      }
      addNode({
        kind: 'Subgraph',
        pos: viewportCenterForNode(),
        config: { SubgraphID: libraryID },
      })
    } catch (e: any) {
      console.error('LibraryExplorer pick failed:', e)
      toast.add({
        title: t('nodeCreation.lib_import_failed'),
        description: errorMessage(e),
        color: 'error',
      })
    }
  }

  // ClipExplorerModal 选 clip — 插入裸 PlayClip 引用节点 (config.ClipID)。
  // PlayClip 自给自足 (校准从 clip 自身 Meta 读), 不引用容器全局变量 →
  // 不需要子图那套缺变量补全, 直接 addNode 即可。
  function onPickLibraryClip(clipID: string) {
    if (!draft.value) return
    addNode({
      kind: 'PlayClip',
      pos: viewportCenterForNode(),
      config: { ClipID: clipID },
    })
  }

  // Programmatic add (录制完成等程序化加点).
  // 特殊: 不走 applyDraftMutation (直接 push + syncFlowFromDraft + 设 selected).
  // Start 单例 guard + autoCreateSubgraphForNewNode 前置 hook (失败不 push).
  async function onAddNode(
    kind: string,
    atX?: number,
    atY?: number,
  ): Promise<string | null> {
    if (!draft.value) return null
    const targetGraph = activeGraph.value
    if (!targetGraph) {
      toast.add({ title: t('nodeCreation.no_graph'), color: 'error' })
      return null
    }
    if (kind === 'Start' && targetGraph.nodes.some((n) => n.kind === 'Start')) {
      toast.add({ title: t('nodeCreation.only_one_start'), color: 'warning' })
      return null
    }

    const n = buildNode({
      kind,
      pos: { x: atX ?? 200 + Math.random() * 200, y: atY ?? 100 + Math.random() * 200 },
      config: { ...KIND_DEFAULTS[kind] },
      id: kind === 'Start' ? 'start' : genNodeID(),
    })
    // onAddNode 老代码: 用 builtNode 时不 set createdAt — buildNode 强 set,
    // 行为对齐. (老代码这一处确实没 createdAt; 修齐为标准统一.)

    if (kind === 'Subgraph') {
      const ok = await autoCreateSubgraphForNewNode(n)
      if (!ok) {
        toast.add({ title: t('nodeCreation.create_subgraph_failed'), color: 'error' })
        return null
      }
    }

    targetGraph.nodes.push(n)
    syncFlowFromDraft()
    selectedID.value = n.id
    return n.id
  }

  return {
    // 主接口
    addNode,
    // utilities (导出给外部 backward-compat, 实际外部不再直调)
    defaultLiteralFor,
    findNearestEligibleDataInPin,
    // 8 callsite 包装 (面向 view)
    dropVar, dropNodeSpec, dropSnippet,
    onInsertIncVar, onApplySnippet,
    onPickKind, onPickLibrarySubgraph, onPickLibraryClip,
    onAddNode,
  }
}
