import { ref, type Ref } from 'vue'
import type { NodeDragEvent } from '@vue-flow/core'
import type { EditorCommand, EditorSession } from './EditorSession'
import { flowGuideToCanvasCoordinate, type CanvasViewport } from './editorCanvasCoordinates'
import {
  alignNodePositions,
  autoLayoutNodePositions,
  distributeNodePositions,
  snapNodePosition,
  type AlignMode,
  type DistributeMode,
  type SizedWorkflowNode,
} from './workflowLayout'

export type EditorCanvasLayoutCommand =
  | { kind: 'align'; mode: AlignMode }
  | { kind: 'distribute'; mode: DistributeMode }
  | { kind: 'auto-layout'; direction: 'LR' | 'TB' }
  | { kind: 'make-space' }
  | { kind: 'clear-guides' }

interface EditorCanvasLayoutDependencies {
  session: EditorSession
  canvasElement: Ref<HTMLElement | null>
  selectedNodeIds: Ref<Set<string>>
  findNode: (nodeId: string) => { dimensions?: { width?: number; height?: number } } | undefined
  fitView: (options: { padding: number; duration: number }) => Promise<unknown>
  getViewport: () => CanvasViewport
  applyCommand: (command: EditorCommand) => boolean
  layoutErrorTitle: () => string
  showError: (title: string, error: unknown) => void
}

export function createEditorCanvasLayoutController(deps: EditorCanvasLayoutDependencies) {
  const snapGuides = ref<{ x?: number; y?: number }>({})
  const layouting = ref(false)

  async function execute(command: EditorCanvasLayoutCommand): Promise<void> {
    switch (command.kind) {
      case 'align':
        applyPositions(alignNodePositions(selectedSizedNodes(), command.mode))
        return
      case 'distribute':
        applyPositions(distributeNodePositions(selectedSizedNodes(), command.mode))
        return
      case 'auto-layout':
        await autoLayout(command.direction)
        return
      case 'make-space':
        makeSpace()
        return
      case 'clear-guides':
        snapGuides.value = {}
    }
  }

  function dragPositions(
    event: NodeDragEvent,
  ): Array<{ nodeId: string; position: { x: number; y: number } }> {
    const dragged = event.nodes.length ? event.nodes : [event.node]
    const draggedIds = new Set(dragged.map((node) => node.id))
    const primary = sizedFlowNode(event.node.id, event.node.position)
    const disableSnap = event.event instanceof MouseEvent && event.event.altKey
    const snapped = disableSnap
      ? { position: event.node.position }
      : snapNodePosition(
          primary,
          (deps.session.currentGraph?.nodes ?? []).flatMap((node) => {
            if (draggedIds.has(node.id)) return []
            return [sizedFlowNode(node.id, node.position)]
          }),
        )
    const delta = {
      x: snapped.position.x - event.node.position.x,
      y: snapped.position.y - event.node.position.y,
    }
    updateSnapGuides(snapped.guideX, snapped.guideY)
    return dragged.map((node) => ({
      nodeId: node.id,
      position: { x: node.position.x + delta.x, y: node.position.y + delta.y },
    }))
  }

  function applyPositions(
    positions: Array<{ nodeId: string; position: { x: number; y: number } }>,
  ): boolean {
    const graph = deps.session.currentGraph
    if (!graph) return false
    let applied = false
    const nodes = positions.filter((item) => graph.nodes.some((node) => node.id === item.nodeId))
    if (nodes.length)
      applied = deps.applyCommand({ kind: 'move-nodes', positions: nodes }) || applied
    for (const item of positions) {
      const call = graph.calls?.find((candidate) => candidate.id === item.nodeId)
      if (call) {
        applied =
          deps.applyCommand({
            kind: 'update-graph-call',
            call: { ...call, position: item.position },
          }) || applied
      }
      const annotation = graph.annotations?.find((candidate) => candidate.id === item.nodeId)
      if (annotation) {
        applied =
          deps.applyCommand({
            kind: 'update-annotation',
            annotation: { ...annotation, position: item.position },
          }) || applied
      }
    }
    return applied
  }

  async function fitCurrentGraph(): Promise<void> {
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(resolve, 50))),
    )
    await deps.fitView({ padding: 0.24, duration: 180 })
  }

  function sizedFlowNode(nodeId: string, position: { x: number; y: number }): SizedWorkflowNode {
    const dimensions = deps.findNode(nodeId)?.dimensions
    const element = [
      ...(deps.canvasElement.value?.querySelectorAll<HTMLElement>('.vue-flow__node') ?? []),
    ].find((candidate) => candidate.dataset.id === nodeId)
    return {
      id: nodeId,
      position,
      width: element?.offsetWidth || dimensions?.width || 230,
      height: element?.offsetHeight || dimensions?.height || 90,
    }
  }

  function selectedSizedNodes(): SizedWorkflowNode[] {
    return [...deps.selectedNodeIds.value].flatMap((nodeId) => {
      const graph = deps.session.currentGraph
      const position =
        graph?.nodes.find((candidate) => candidate.id === nodeId)?.position ??
        graph?.calls?.find((candidate) => candidate.id === nodeId)?.position ??
        graph?.annotations?.find((candidate) => candidate.id === nodeId)?.position
      return position ? [sizedFlowNode(nodeId, position)] : []
    })
  }

  function updateSnapGuides(guideX?: number, guideY?: number): void {
    if (!deps.canvasElement.value) {
      snapGuides.value = {}
      return
    }
    const viewport = deps.getViewport()
    snapGuides.value = {
      x: guideX === undefined ? undefined : flowGuideToCanvasCoordinate(guideX, viewport, 'x'),
      y: guideY === undefined ? undefined : flowGuideToCanvasCoordinate(guideY, viewport, 'y'),
    }
  }

  async function autoLayout(direction: 'LR' | 'TB'): Promise<void> {
    if (layouting.value) return
    const graph = deps.session.currentGraph
    const source = deps.session.source
    if (!graph || !source || graph.nodes.length + (graph.calls?.length ?? 0) === 0) return
    const nodes = [
      ...graph.nodes.map((node) => sizedFlowNode(node.id, node.position)),
      ...(graph.calls ?? []).map((call) => sizedFlowNode(call.id, call.position)),
    ]
    layouting.value = true
    try {
      const positions = await autoLayoutNodePositions(nodes, graph.edges, direction)
      if (deps.session.source !== source || deps.session.currentGraph?.id !== graph.id) return
      if (applyPositions(positions)) await fitCurrentGraph()
    } catch (error) {
      deps.showError(deps.layoutErrorTitle(), error)
    } finally {
      layouting.value = false
    }
  }

  function makeSpace(): void {
    const graph = deps.session.currentGraph
    if (!graph || deps.selectedNodeIds.value.size !== 2) return
    const selected = graph.nodes
      .filter((node) => deps.selectedNodeIds.value.has(node.id))
      .sort(
        (left, right) => left.position.x - right.position.x || left.position.y - right.position.y,
      )
    if (selected.length !== 2) return
    const [first, second] = selected
    const horizontal =
      Math.abs(second.position.x - first.position.x) >=
      Math.abs(second.position.y - first.position.y)
    const positions = graph.nodes.flatMap((node) => {
      const after = horizontal
        ? node.position.x >= second.position.x
        : node.position.y >= second.position.y
      if (!after) return []
      return [
        {
          nodeId: node.id,
          position: {
            x: node.position.x + (horizontal ? 280 : 0),
            y: node.position.y + (horizontal ? 0 : 160),
          },
        },
      ]
    })
    applyPositions(positions)
  }

  return {
    snapGuides,
    layouting,
    execute,
    dragPositions,
    applyPositions,
    fitCurrentGraph,
  }
}
