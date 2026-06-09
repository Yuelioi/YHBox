// 4 个 context menu (Node / Multi / Edge / Pin) 路由 + 各 action dispatcher.
//
// 内部持有 4 个 menu state ref; capture handler (onNodeContextMenu /
// onSelectionContextMenu / onEdgeContextMenu / onCanvasContextMenuCapture) 开 menu,
// action dispatcher 处理选项. promoteCtx / findRefsState 由 view 持有 (modal state),
// composable 通过 setPromote / setFindRefs 写回 view ref.

import { nextTick, ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useVueFlow } from '@vue-flow/core'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { backend, type Container, type Graph, type GraphNode, type GraphEdge } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { dataInTypeFor, dataOutTypeFor } from '@/components/containers/nodeRegistry/registry'
import { type VarType } from '@/lib/variableRef'
import { newNodeID } from './ids'
import { centerOnNode } from './constants'
import { walkAllGraphs } from './graphWalk'
import type { useVarMutations } from './useVarMutations'
import type { AlignMode } from './useGraphLayout'
import type { NodeMenuAction } from '@/components/containers/menus/NodeContextMenu.vue'
import type { MultiMenuAction } from '@/components/containers/menus/MultiNodeContextMenu.vue'
import type { EdgeMenuAction } from '@/components/containers/menus/EdgeContextMenu.vue'
import type { PinMenuAction, PinInfo } from '@/components/containers/menus/PinContextMenu.vue'
import type { PromoteContext } from '@/components/containers/PromoteToVarModal.vue'
import type { RefEntry } from '@/components/containers/FindReferencesModal.vue'

type ToastApi = { add: (opts: { title: string; description?: string; color?: string; icon?: string }) => void }

interface UseContextMenuRouterOpts {
  containerID: string
  draft: Ref<Container | null>
  activeGraph: Ref<Graph | null>
  selectedID: Ref<string | null>
  promoteCtx: Ref<PromoteContext | null>
  findRefsState: Ref<{ varName: string; refs: RefEntry[] } | null>
  applyDraftMutation: (mutator: (draft: Container) => void) => void
  varMutations: ReturnType<typeof useVarMutations>
  // actions from sibling composables
  onCopySelection: () => void
  onPasteSelection: () => Promise<unknown> | void
  onFoldSelection: () => void
  onAlignSelected: (mode: AlignMode) => void
  onAutoLayout: (dir: 'LR' | 'TB') => void
  emitSaveSnippetIntent: (node: GraphNode) => void
  toast: ToastApi
}

export function useContextMenuRouter(opts: UseContextMenuRouterOpts) {
  const {
    containerID, draft, activeGraph, selectedID,
    promoteCtx, findRefsState,
    applyDraftMutation, varMutations,
    onCopySelection, onPasteSelection, onFoldSelection,
    onAlignSelected, onAutoLayout,
    emitSaveSnippetIntent,
    toast,
  } = opts

  const editorStore = useContainerEditorStore()
  const { t } = useI18n()
  const { getSelectedNodes, removeNodes, setCenter } = useVueFlow()

  // ===== Menu state (4 个 menu 互斥) =====
  const nodeMenu = ref<{ open: boolean; position: { x: number; y: number }; node: GraphNode | null }>({
    open: false, position: { x: 0, y: 0 }, node: null,
  })
  const multiMenu = ref<{ open: boolean; position: { x: number; y: number }; count: number }>({
    open: false, position: { x: 0, y: 0 }, count: 0,
  })
  const edgeMenu = ref<{ open: boolean; position: { x: number; y: number }; edge: GraphEdge | null }>({
    open: false, position: { x: 0, y: 0 }, edge: null,
  })
  const pinMenu = ref<{ open: boolean; position: { x: number; y: number }; pin: PinInfo | null }>({
    open: false, position: { x: 0, y: 0 }, pin: null,
  })

  // ===== Capture handlers =====
  function onNodeContextMenu(event: { event: Event; node: { id: string } }) {
    ;(event.event as Event).preventDefault?.()
    const clientX = event.event instanceof MouseEvent ? event.event.clientX : 0
    const clientY = event.event instanceof MouseEvent ? event.event.clientY : 0
    const node = activeGraph.value?.nodes?.find((n) => n.id === event.node.id)
    if (!node) return
    // 多选且点击节点在选中集合内 → 多选菜单
    const sel = getSelectedNodes.value ?? []
    if (sel.length > 1 && sel.some((n: any) => n.id === node.id)) {
      multiMenu.value = {
        open: true,
        position: { x: clientX, y: clientY },
        count: sel.length,
      }
      nodeMenu.value.open = false
      return
    }
    nodeMenu.value = { open: true, position: { x: clientX, y: clientY }, node }
    multiMenu.value.open = false
  }

  function onSelectionContextMenu(event: { event: MouseEvent; nodes: any[] }) {
    event.event.preventDefault()
    multiMenu.value = {
      open: true,
      position: { x: event.event.clientX, y: event.event.clientY },
      count: event.nodes.length,
    }
    nodeMenu.value.open = false
  }

  function onEdgeContextMenu(event: { event: Event; edge: { source: string; sourceHandle?: string | null; target: string; targetHandle?: string | null } }) {
    if (event.event instanceof MouseEvent) event.event.preventDefault()
    const clientX = event.event instanceof MouseEvent ? event.event.clientX : 0
    const clientY = event.event instanceof MouseEvent ? event.event.clientY : 0
    const flowEdge = activeGraph.value?.edges.find(
      (ed) =>
        ed.from === `${event.edge.source}.${event.edge.sourceHandle}` &&
        ed.to === `${event.edge.target}.${event.edge.targetHandle}`,
    )
    if (!flowEdge) return
    edgeMenu.value = { open: true, position: { x: clientX, y: clientY }, edge: flowEdge }
    nodeMenu.value.open = false
    multiMenu.value.open = false
    pinMenu.value.open = false
  }

  // canvas 右键 capture-phase: pin 上右键 → 拦截开 pinMenu; 否则 let pane/node/edge 处理
  function onCanvasContextMenuCapture(e: MouseEvent) {
    const t = e.target as HTMLElement | null
    if (!t) return
    const handleEl = t.closest('.vue-flow__handle') as HTMLElement | null
    if (!handleEl) return
    e.preventDefault()
    e.stopPropagation()

    const nodeEl = handleEl.closest('[data-id]') as HTMLElement | null
    const nodeID = nodeEl?.getAttribute('data-id') ?? ''
    const pinName = handleEl.getAttribute('data-handleid') ?? ''
    const handleType = handleEl.getAttribute('data-handletype') ?? ''
    const side: 'input' | 'output' = handleType === 'source' ? 'output' : 'input'

    const node = activeGraph.value?.nodes.find((n) => n.id === nodeID)
    if (!node) return

    let pinType: string | undefined
    if (side === 'input') {
      pinType = dataInTypeFor(node.kind, pinName, node.config as Record<string, unknown>) || undefined
    } else {
      pinType = dataOutTypeFor(node.kind, pinName) || undefined
    }

    const edges = activeGraph.value?.edges ?? []
    const matchFrom = `${nodeID}.${pinName}`
    const edgeCount = edges.filter((ed) => ed.from === matchFrom || ed.to === matchFrom).length

    pinMenu.value = {
      open: true,
      position: { x: e.clientX, y: e.clientY },
      pin: { nodeID, pinName, side, pinType, edgeCount },
    }
    nodeMenu.value.open = false
    multiMenu.value.open = false
    edgeMenu.value.open = false
  }

  // ===== Action dispatchers =====
  async function shareSubgraphToLibrary(sgID: string) {
    if (!containerID) return
    if (!window.confirm(t('contextMenu.share_confirm', { sgID }))) return
    try {
      await backend.library.exportSubgraph(containerID, sgID, true)
      toast.add({ title: t('contextMenu.share_success', { sgID }), color: 'success', icon: 'i-tabler-check' })
    } catch (e: any) {
      toast.add({ title: t('contextMenu.share_failed'), description: errorMessage(e), color: 'error' })
    }
  }

  function onNodeMenuAction(a: NodeMenuAction) {
    const node = nodeMenu.value.node
    if (!node) return
    switch (a) {
      case 'copy':
        onCopySelection()
        return
      case 'cut':
        onCopySelection()
        getSelectedNodes.value.forEach((n: any) => removeNodes([n.id]))
        return
      case 'paste':
        void onPasteSelection()
        return
      case 'duplicate':
        onCopySelection()
        void onPasteSelection()
        return
      case 'delete':
        removeNodes([node.id])
        if (selectedID.value === node.id) selectedID.value = null
        return
      case 'toggle-disable':
        applyDraftMutation(() => {
          const g = activeGraph.value
          const n = g?.nodes.find((x) => x.id === node.id) as (GraphNode & { disabled?: boolean }) | undefined
          if (!n) return
          n.disabled = !n.disabled
        })
        return
      case 'save-as-snippet':
        emitSaveSnippetIntent(node)
        return
      case 'find-references': {
        const varName = (node.config as Record<string, unknown> | undefined)?.varName as string | undefined
        if (!varName) return
        const usageRefs = varMutations.listUsageRefs(varName)
        const accessByID = new Map(usageRefs.map(r => [r.nodeID, r.access]))
        const refs: RefEntry[] = []
        if (draft.value) {
          walkAllGraphs(draft.value, (n, { location }) => {
            if (accessByID.has(n.id)) {
              refs.push({ id: n.id, kind: n.kind, label: n.label, location, access: accessByID.get(n.id) })
            }
          })
        }
        findRefsState.value = { varName, refs }
        return
      }
      case 'promote-to-var': {
        const lit = (node.config as Record<string, unknown> | undefined)?.literal as Record<string, unknown> | undefined
        if (!lit || Object.keys(lit).length === 0) {
          toast.add({ title: 'Promote', description: t('contextMenu.no_literal_pin'), color: 'warning' })
          return
        }
        // 单选节点 promote 只挑第一个 literal pin
        const pinName = Object.keys(lit)[0]
        const literal = lit[pinName]
        const pinType = dataInTypeFor(node.kind, pinName, node.config as Record<string, unknown>) as VarType | ''
        if (!pinType) {
          toast.add({ title: 'Promote', description: t('contextMenu.pin_not_data_in', { pin: pinName }), color: 'warning' })
          return
        }
        promoteCtx.value = { nodeID: node.id, pinName, pinType: pinType as VarType, literal }
        return
      }
      case 'jump-to-subgraph': {
        const sgID = (node.config as Record<string, unknown> | undefined)?.SubgraphID as string | undefined
        if (sgID) editorStore.pushPath(sgID)
        return
      }
      case 'share-to-library': {
        const sgID = (node.config as Record<string, unknown> | undefined)?.SubgraphID as string | undefined
        if (!sgID) return
        void shareSubgraphToLibrary(sgID)
        return
      }
    }
  }

  function onMultiMenuAction(a: MultiMenuAction) {
    switch (a) {
      case 'copy':
        onCopySelection()
        return
      case 'cut':
        onCopySelection()
        getSelectedNodes.value.forEach((n: any) => removeNodes([n.id]))
        return
      case 'paste':
        void onPasteSelection()
        return
      case 'duplicate':
        onCopySelection()
        void onPasteSelection()
        return
      case 'delete':
        getSelectedNodes.value.forEach((n: any) => removeNodes([n.id]))
        return
      case 'toggle-disable-all':
        applyDraftMutation(() => {
          const g = activeGraph.value
          if (!g) return
          const sel = new Set(getSelectedNodes.value.map((n: any) => n.id))
          const allDisabled = (g.nodes as Array<GraphNode & { disabled?: boolean }>)
            .filter((n) => sel.has(n.id))
            .every((n) => n.disabled === true)
          for (const n of g.nodes as Array<GraphNode & { disabled?: boolean }>) {
            if (sel.has(n.id)) n.disabled = !allDisabled
          }
        })
        return
      case 'fold':
        onFoldSelection()
        return
      case 'auto-layout-lr':
        onAutoLayout('LR')
        return
      case 'auto-layout-tb':
        onAutoLayout('TB')
        return
      case 'align-left': onAlignSelected('left'); return
      case 'align-right': onAlignSelected('right'); return
      case 'align-top': onAlignSelected('top'); return
      case 'align-bottom': onAlignSelected('bottom'); return
      case 'align-center-h': onAlignSelected('center-h'); return
      case 'align-center-v': onAlignSelected('center-v'); return
      case 'distribute-h': onAlignSelected('h-equal'); return
      case 'distribute-v': onAlignSelected('v-equal'); return
    }
  }

  function onEdgeMenuAction(a: EdgeMenuAction) {
    const edge = edgeMenu.value.edge
    if (!edge) return
    switch (a) {
      case 'delete':
        applyDraftMutation(() => {
          const g = activeGraph.value
          if (!g) return
          g.edges = g.edges.filter((e) => !(e.from === edge.from && e.to === edge.to))
        })
        return
    }
  }

  function onPinMenuAction(a: PinMenuAction) {
    const pin = pinMenu.value.pin
    if (!pin) return
    const matchID = `${pin.nodeID}.${pin.pinName}`
    switch (a) {
      case 'break-all-connections':
        applyDraftMutation(() => {
          const g = activeGraph.value
          if (!g) return
          g.edges = g.edges.filter((ed) => ed.from !== matchID && ed.to !== matchID)
        })
        return
      case 'promote-to-var': {
        const node = activeGraph.value?.nodes.find((n) => n.id === pin.nodeID)
        if (!node) return
        const lit = (node.config as Record<string, unknown> | undefined)?.literal as Record<string, unknown> | undefined
        const literal = lit?.[pin.pinName]
        if (literal === undefined) {
          toast.add({ title: 'Promote', description: t('contextMenu.pin_no_literal', { pin: pin.pinName }), color: 'warning' })
          return
        }
        promoteCtx.value = {
          nodeID: pin.nodeID,
          pinName: pin.pinName,
          pinType: (pin.pinType ?? 'any') as VarType,
          literal,
        }
        return
      }
      case 'reset-to-literal':
        applyDraftMutation(() => {
          const g = activeGraph.value
          if (!g) return
          g.edges = g.edges.filter((ed) => ed.to !== matchID)
          const node = g.nodes.find((n) => n.id === pin.nodeID)
          if (!node) return
          const cfg = node.config as Record<string, unknown>
          const lit = (cfg.literal as Record<string, unknown> | undefined) ?? {}
          const def =
            pin.pinType === 'number' ? 0
            : pin.pinType === 'string' ? ''
            : pin.pinType === 'bool' ? false
            : pin.pinType === 'point' ? { x: 0.5, y: 0.5 }
            : null
          lit[pin.pinName] = def
          cfg.literal = lit
        })
        return
      case 'show-schema':
        toast.add({
          title: `Pin schema: ${pin.pinName}`,
          description: `${pin.side} · type: ${pin.pinType ?? '(exec)'} · edges: ${pin.edgeCount}`,
          color: 'info',
        })
        return
    }
  }

  // ===== Find-References pick (FindReferencesModal 点击 ref entry → 跳到节点) =====
  async function onFindRefsPick(nodeID: string) {
    const currentNodes = activeGraph.value?.nodes
    const inCurrent = currentNodes?.some((n) => n.id === nodeID) ?? false
    if (!inCurrent) {
      let targetSgID: string | null = null
      let targetNode: GraphNode | undefined
      for (const sg of draft.value?.subgraphs ?? []) {
        const found = sg.graph?.nodes?.find((n) => n.id === nodeID)
        if (found) {
          targetSgID = sg.id
          targetNode = found
          break
        }
      }
      if (targetSgID) {
        editorStore.setPath([targetSgID])
        await nextTick()
        selectedID.value = nodeID
        if (targetNode) centerOnNode(setCenter, targetNode)
      } else {
        toast.add({ title: t('contextMenu.jump_failed'), description: t('contextMenu.node_not_in_container', { id: nodeID }), color: 'warning' })
      }
      findRefsState.value = null
      return
    }
    selectedID.value = nodeID
    const targetNode = currentNodes?.find((n) => n.id === nodeID)
    if (targetNode) centerOnNode(setCenter, targetNode)
    findRefsState.value = null
  }

  // ===== Promote-to-Variable confirm (PromoteToVarModal '确定' → 实际 promote) =====
  function onPromoteConfirm(args: { varName: string; varType: VarType }) {
    const ctx = promoteCtx.value
    if (!ctx) return
    applyDraftMutation((d) => {
      const g = activeGraph.value
      if (!g) return

      // 1. Add new var to Container.Vars
      if (!d.vars) d.vars = []
      d.vars.push({
        name: args.varName,
        type: args.varType as 'number' | 'bool' | 'string' | 'point' | 'any',
        default: ctx.literal,
      })

      // 2. Remove literal from original node's config
      const origNode = g.nodes.find((n) => n.id === ctx.nodeID)
      if (!origNode) return
      const cfg = origNode.config as Record<string, unknown>
      const lit = cfg.literal as Record<string, unknown> | undefined
      if (lit) delete lit[ctx.pinName]

      // 3. Insert GetVar node 200px to the left
      const getVarID = newNodeID('GetVar')
      g.nodes.push({
        id: getVarID,
        kind: 'GetVar',
        x: (origNode.x ?? 0) - 200,
        y: origNode.y ?? 0,
        config: { literal: { VarName: args.varName, Scope: 'auto' } },
        createdAt: new Date().toISOString(),
      } as GraphNode)

      // 4. Add data edge: GetVar.Value → originalNode.pinName
      g.edges.push({
        from: `${getVarID}.Value`,
        to: `${ctx.nodeID}.${ctx.pinName}`,
      } as GraphEdge)
    })
    promoteCtx.value = null
  }

  return {
    // menu state refs (template 用)
    nodeMenu, multiMenu, edgeMenu, pinMenu,
    // capture handlers
    onNodeContextMenu, onSelectionContextMenu, onEdgeContextMenu, onCanvasContextMenuCapture,
    // action dispatchers
    onNodeMenuAction, onMultiMenuAction, onEdgeMenuAction, onPinMenuAction,
    // modal flow handlers
    onFindRefsPick, onPromoteConfirm,
  }
}
