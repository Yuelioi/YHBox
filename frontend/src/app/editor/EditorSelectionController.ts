import { nextTick, type Ref } from 'vue'
import type { EditorCommand, EditorSession } from './EditorSession'

export type EditorSelectionCommand =
  | { kind: 'clear' }
  | { kind: 'remove' }
  | { kind: 'duplicate' }
  | { kind: 'copy' }
  | { kind: 'cut' }
  | { kind: 'paste' }
  | { kind: 'select-inserted'; nodeIds: string[] }

interface WorkflowSelectionClipboard {
  format: 'yotta.workflow-selection'
  version: 2
  nodes: ReturnType<EditorSession['selectionSnapshot']>['nodes']
  calls: ReturnType<EditorSession['selectionSnapshot']>['calls']
  annotations: ReturnType<EditorSession['selectionSnapshot']>['annotations']
  edges: ReturnType<EditorSession['selectionSnapshot']>['edges']
}

interface EditorSelectionDependencies<TFlowNode> {
  session: EditorSession
  selectedNodeId: Ref<string>
  selectedNodeIds: Ref<Set<string>>
  selectedEdgeId: Ref<string>
  selectedFlowNodes: () => TFlowNode[]
  findNode: (nodeId: string) => TFlowNode | undefined
  addSelectedNodes: (nodes: TFlowNode[]) => void
  removeSelectedNodes: (nodes: TFlowNode[]) => void
  applyCommand: (command: EditorCommand) => boolean
  disconnectEdge: (edgeId: string) => void
  clipboard?: Pick<Clipboard, 'readText' | 'writeText'>
  translate: (key: string) => string
  showError: (title: string, error: unknown) => void
}

export function createEditorSelectionController<TFlowNode>(
  deps: EditorSelectionDependencies<TFlowNode>,
) {
  let workflowClipboard: WorkflowSelectionClipboard | null = null
  let pasteOffset = 0

  async function execute(command: EditorSelectionCommand): Promise<void> {
    switch (command.kind) {
      case 'clear':
        clearSelection()
        return
      case 'remove':
        removeSelection()
        return
      case 'duplicate':
        duplicateSelection()
        return
      case 'copy':
        await copySelection()
        return
      case 'cut':
        await copySelection()
        removeSelection()
        return
      case 'paste':
        await pasteSelection()
        return
      case 'select-inserted':
        await selectInsertedNodes(command.nodeIds)
    }
  }

  function clearSelection(): void {
    deps.removeSelectedNodes(deps.selectedFlowNodes())
    deps.selectedNodeId.value = ''
    deps.selectedNodeIds.value = new Set()
    deps.selectedEdgeId.value = ''
  }

  function removeSelection(): void {
    const ids = deps.selectedNodeIds.value.size
      ? [...deps.selectedNodeIds.value]
      : deps.selectedNodeId.value
        ? [deps.selectedNodeId.value]
        : []
    const graph = deps.session.currentGraph
    const nodeIds = ids.filter((id) => graph?.nodes.some((node) => node.id === id))
    const callIds = ids.filter((id) => graph?.calls?.some((call) => call.id === id))
    const annotationIds = ids.filter((id) =>
      graph?.annotations?.some((annotation) => annotation.id === id),
    )
    if (nodeIds.length) deps.applyCommand({ kind: 'remove-nodes', nodeIds })
    for (const callId of callIds) deps.applyCommand({ kind: 'remove-graph-call', callId })
    for (const annotationId of annotationIds) {
      deps.applyCommand({ kind: 'remove-annotation', annotationId })
    }
    if (ids.length) {
      deps.selectedNodeId.value = ''
      deps.selectedNodeIds.value = new Set()
      return
    }
    if (deps.selectedEdgeId.value) deps.disconnectEdge(deps.selectedEdgeId.value)
  }

  function duplicateSelection(): void {
    try {
      const ids = deps.session.duplicateNodes([...deps.selectedNodeIds.value])
      if (ids.length) void selectInsertedNodes(ids)
    } catch (error) {
      deps.showError(deps.translate('workflow.toast.edit_rejected'), error)
    }
  }

  async function copySelection(): Promise<void> {
    const snapshot = deps.session.selectionSnapshot([...deps.selectedNodeIds.value])
    if (!snapshot.nodes.length && !snapshot.calls.length && !snapshot.annotations.length) return
    workflowClipboard = {
      format: 'yotta.workflow-selection',
      version: 2,
      ...snapshot,
    }
    pasteOffset = 0
    try {
      await deps.clipboard?.writeText(JSON.stringify(workflowClipboard))
    } catch {
      return
    }
  }

  async function pasteSelection(): Promise<void> {
    let clipboard = workflowClipboard
    try {
      const text = await deps.clipboard?.readText()
      if (text) clipboard = parseWorkflowClipboard(text)
    } catch {
      if (!clipboard) {
        deps.showError(
          deps.translate('workflow.selection.clipboard_failed'),
          new Error('clipboard is unavailable'),
        )
        return
      }
    }
    if (!clipboard) return
    try {
      pasteOffset += 24
      const ids = deps.session.insertNodeSelection(clipboard, {
        x: pasteOffset,
        y: pasteOffset,
      })
      if (ids.length) await selectInsertedNodes(ids)
    } catch (error) {
      deps.showError(deps.translate('workflow.toast.edit_rejected'), error)
    }
  }

  async function selectInsertedNodes(nodeIds: string[]): Promise<void> {
    await nextTick()
    deps.removeSelectedNodes(deps.selectedFlowNodes())
    const nodes = nodeIds.flatMap((nodeId) => {
      const node = deps.findNode(nodeId)
      return node ? [node] : []
    })
    if (nodes.length) deps.addSelectedNodes(nodes)
    deps.selectedNodeIds.value = new Set(nodeIds)
    deps.selectedNodeId.value = nodeIds.at(-1) ?? ''
  }

  return { execute }
}

function parseWorkflowClipboard(value: string): WorkflowSelectionClipboard {
  if (value.length > 1_000_000) throw new Error('workflow clipboard exceeds size budget')
  const parsed = JSON.parse(value) as Partial<WorkflowSelectionClipboard>
  if (
    parsed.format !== 'yotta.workflow-selection' ||
    parsed.version !== 2 ||
    !Array.isArray(parsed.nodes) ||
    !Array.isArray(parsed.calls) ||
    !Array.isArray(parsed.annotations) ||
    !Array.isArray(parsed.edges)
  ) {
    throw new Error('clipboard does not contain a workflow selection')
  }
  return parsed as WorkflowSelectionClipboard
}
