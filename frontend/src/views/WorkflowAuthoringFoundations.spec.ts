import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const readSource = (path: string) => readFileSync(join(process.cwd(), path), 'utf8')

describe('workflow authoring foundations', () => {
  it('opens a completed recording from the authoritative pending state only once', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const assets = readSource('src/views/AssetsView.vue')
    const recordingMetadata = readSource('src/components/recording/RecordingMetadataFields.vue')
    expect(editor).not.toContain('if (payload) openRecordingPreview(payload)')
    expect(assets).not.toContain('if (payload) openRecordingSave(payload)')
    expect(editor).toContain('() => recording.state.pending')
    expect(assets).toContain('() => recording.state.pending')
    expect(editor).toContain('!editorViewActive.value')
    expect(editor).toContain('onDeactivated(() =>')
    const recordingModal = editor.slice(
      editor.indexOf(':open="!!pendingRecording"'),
      editor.indexOf(':open="!!macroEditing"'),
    )
    expect(recordingModal).toContain('size="3xl"')
    expect(recordingModal).not.toContain('\n      tall')
    expect(recordingModal).not.toContain("t('recordingSave.optional_metadata')")
    expect(recordingModal).toContain('<RecordingMetadataFields')
    expect(editor).toContain('const RecordingMetadataFields = defineAsyncComponent(')
    expect(editor).not.toContain('<UInputMenu')
    expect(recordingMetadata).toContain(':create-item="\'always\'"')
    expect(recordingMetadata).toContain('@create="createCategory"')
    expect(recordingMetadata).toContain('@create="createTag"')
    expect(recordingMetadata).toContain('multiple')
    expect(assets).not.toContain("t('recordingSave.optional_metadata')")
  })

  it('provides searchable grouped catalog and an actionable empty canvas', () => {
    const source = readSource('src/views/WorkflowEditorView.vue')
    expect(source).toContain('v-model="catalogQuery"')
    expect(source).toContain('v-for="group in catalogGroups"')
    expect(source).toContain('data-testid="workflow-empty-canvas"')
    expect(source).toContain('addNode(RUN_STARTED_NODE_ID')
  })

  it('routes ordinary Delete and Backspace through editor commands', () => {
    const source = readSource('src/views/WorkflowEditorView.vue')
    expect(source).toContain(':delete-key-code="null"')
    expect(source).toContain("event.key !== 'Delete' && event.key !== 'Backspace'")
    expect(source).toContain(
      'target?.matches(\'input, textarea, select, [contenteditable="true"]\')',
    )
    expect(source).toContain("applyCommand({ kind: 'remove-nodes'")
    expect(source).toContain("applyCommand({ kind: 'disconnect'")
    expect(source).toContain('@edge-click="selectEdge"')
  })

  it('keeps workflow state outside the selected-node inspector', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const inspector = readSource('src/app/editor/WorkflowInspector.vue')
    const generatedField = readSource('src/app/editor/GeneratedFieldEditor.vue')
    expect(editor).toContain('<WorkflowStatePanel')
    expect(inspector).not.toContain("kind: 'add-state-variable'")
    expect(inspector).toContain('<WorkflowAuthoringSurfaceItem')
    expect(readSource('src/app/editor/WorkflowAuthoringSurfaceItem.vue')).toContain(
      "path: '/settings', query: { section: targetSettingsSection }",
    )
    expect(inspector).toContain('projectionDescription')
    expect(generatedField).toContain('<USelectMenu')
    expect(generatedField).toContain("t('workflow.inspector.search_target')")
    expect(generatedField).toContain(':virtualize="selectItems.length > 40"')
  })

  it('separates editable macros, precise clips, templates, and recording entry points', () => {
    const source = readSource('src/views/AssetsView.vue')
    const router = readSource('src/router/index.ts')
    expect(router).toContain("path: '/assets'")
    expect(source).toContain("openResourceAction(activeTab === 'macros' ? 'macro' : 'precise')")
    expect(source).toContain("activeTab.value === 'macros' ? 'macro'")
    expect(source).toContain('<MacroActionEditor')
    expect(source).toContain("openScreenPicker('template_save'")
    expect(source).toContain('backend.assets.updateMeta')
    expect(source).toContain("'template_recapture'")
    expect(source).toContain('backend.assets.removeVariant')
  })

  it('keeps recording and resource binding inside the workflow workspace', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const dock = readSource('src/app/editor/WorkflowResourceDock.vue')
    const toolbar = readSource('src/app/editor/WorkflowEditorToolbar.vue')

    expect(editor).toContain('<WorkflowResourceDock')
    expect(editor).toContain('@capture-template="openTemplateCapture"')
    expect(editor).toContain('@use="useWorkspaceResource"')
    expect(editor).toContain('session.insertLinearDraft(')
    expect(editor).toContain("kind: 'bind-blob'")
    expect(dock).toContain('pageSize = 20')
    expect(dock).toContain('assets.query(')
    expect(dock).toContain("emit('start-recording'")
    expect(dock).toContain("emit('edit', asset)")
    expect(editor).toContain('@edit="openMacroEditor"')
    expect(editor).toContain('backend.macros.get(asset.guid)')
    expect(dock).toContain("allCategoriesValue = '__yotta_all_categories__'")
    expect(dock).not.toContain("value: ''")
    expect(toolbar).not.toContain('workflow-macro-recording-start')
  })

  it('shares a scannable resource list and lets resources be dragged onto the canvas', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const dock = readSource('src/app/editor/WorkflowResourceDock.vue')
    const assets = readSource('src/views/AssetsView.vue')

    expect(dock).toContain('<AssetLibraryList')
    expect(assets).toContain('<AssetLibraryList')
    expect(dock).toContain('draggable')
    expect(dock).toContain('RESOURCE_DRAG_FORMAT')
    expect(editor).toContain('RESOURCE_DRAG_FORMAT')
    expect(editor).toContain('dropWorkspaceResource')
  })

  it('offers save, discard, and cancel when leaving a dirty workflow', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const dialog = readSource('src/components/common/ConfirmDialog.vue')
    expect(editor).toContain("alternateValue: 'discard'")
    expect(editor).toContain('if (decision === true) return save()')
    expect(dialog).toContain('data-testid="confirm-alternate"')
  })

  it('uses a node-generic context menu and keeps visual templates in typed editor fields', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const node = readSource('src/app/editor/WorkflowNode.vue')
    const dock = readSource('src/app/editor/WorkflowResourceDock.vue')
    const inspector = readSource('src/app/editor/WorkflowInspector.vue')
    const surfaceItem = readSource('src/app/editor/WorkflowAuthoringSurfaceItem.vue')

    expect(node).toContain('<UDropdownMenu')
    expect(node).toContain('@contextmenu.prevent.stop="openNodeContextMenu"')
    expect(node).not.toContain('@contextmenu.prevent.stop="emit(\'save-snippet\')"')
    expect(node).toContain('workflow-node-menu-copy')
    expect(node).toContain('workflow-node-menu-toggle-disabled')
    expect(node).toContain('workflow-node-menu-toggle-breakpoint')
    expect(node).toContain('workflow-node-menu-collapse')
    expect(node).toContain('workflow-node-menu-save-snippet')
    expect(node).toContain('workflow-node-menu-remove')
    expect(node).toContain("color: 'error'")
    expect(node).not.toContain('workflow-node-menu-visual-template')
    expect(node).not.toContain('workflow-node-menu-choose-template')
    expect(node).not.toContain('workflow-node-menu-capture-template')
    expect(editor).toContain('@context-open="selectNodeForContextMenu')
    expect(editor).not.toContain('@open-template-resources="openTemplateResources')
    expect(editor).toContain(
      '@capture-template="selectedNode && captureTemplateForNode(selectedNode.id)"',
    )
    expect(inspector).toContain('@capture-template="emit(\'capture-template\')"')
    expect(surfaceItem).toContain('@capture-template="emit(\'capture-template\')"')
    expect(editor).toContain('v-model:kind="workspaceResourceKind"')
    expect(dock).toContain("defineModel<ResourceKind>('kind'")
    expect(dock).toContain(':aria-pressed="kind === item.value"')
  })

  it('makes both editor sidebars collapsible and resizable', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    expect(editor).toContain('data-testid="workflow-workspace-sidebar"')
    expect(editor).toContain('data-testid="workflow-inspector-sidebar"')
    expect(editor).toContain('role="separator"')
    expect(editor).toContain('resizeWorkspaceSidebar')
    expect(editor).toContain('resizeInspectorSidebar')
    expect(editor).toContain('toggleWorkspacePanel')
    expect(editor).toContain('inspectorSidebarOpen')
  })

  it('switches from Run State to node properties without overriding a hidden Inspector preference', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const selectNode = editor.slice(
      editor.indexOf('function selectNode('),
      editor.indexOf('function selectNodeForContextMenu'),
    )
    expect(selectNode).toContain('statePanelOpen.value = false')
    expect(selectNode).toContain('if (inspectorAutoOpen.value) inspectorSidebarOpen.value = true')
    expect(editor).toContain('function setInspectorVisibility(')
    expect(editor).toContain('inspectorAutoOpen.value = open')
  })

  it('keeps workflow settings and reload next to Save with an overflow seam', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const toolbar = readSource('src/app/editor/WorkflowEditorToolbar.vue')

    expect(toolbar).toContain('data-testid="workflow-settings"')
    expect(toolbar).toContain('data-testid="workflow-more"')
    expect(toolbar).toContain("emit('settings')")
    expect(toolbar).toContain("emit('reload')")
    expect(editor).toContain('<WorkflowMetadataDialog')
    expect(editor).toContain('const WorkflowMetadataDialog = defineAsyncComponent(')
    expect(editor).toContain('@reload="reloadWorkflow"')
  })

  it('asks for an automation target only when a library creation action starts', () => {
    const source = readSource('src/views/AssetsView.vue')
    const header = source.slice(source.indexOf('<header'), source.indexOf('</header>'))

    expect(header).not.toContain('selectedTargetSlot')
    expect(source).toContain('v-model:open="resourceActionOpen"')
    expect(source).toContain('openResourceAction')
  })

  it('scales the asset library with server paging, cross-page batches, and guarded cleanup', () => {
    const source = readSource('src/views/AssetsView.vue')
    expect(source).toContain('assets.query')
    expect(source).toContain('page: page.value')
    expect(source).toContain('pageSize: pageSize.value')
    expect(source).toContain('toggleCurrentPage')
    expect(source).toContain('backend.assets.batchUpdateMeta')
    expect(source).toContain('backend.assets.batchDelete')
    expect(source).toContain('retainFailedSelection')
    expect(source).toContain('backend.assets.previewCleanup')
    expect(source).toContain('backend.assets.commitCleanup(preview.token)')
  })

  it('uses one paged asset picker boundary for node and graph-call BlobRef bindings', () => {
    const inputEditor = readSource('src/app/editor/WorkflowInputBindingEditor.vue')
    const inspector = readSource('src/app/editor/WorkflowInspector.vue')
    const graphCallInspector = readSource('src/app/editor/WorkflowGraphCallInspector.vue')
    const picker = readSource('src/components/assets/AssetPickerModal.vue')

    expect(inputEditor).toContain('<AssetPickerModal')
    expect(picker).toContain('confirmSelection')
    expect(picker).toContain("'assetPicker.use_template'")
    expect(picker).toContain('assets.query')
    expect(picker).toContain('thumbnailBudget: 12')
    expect(inspector).not.toContain('clipsStore.refresh')
    expect(inspector).not.toContain('templatesStore.reload')
    expect(graphCallInspector).not.toContain('clipsStore.refresh')
  })

  it('offers compatible nodes when a typed connection ends on the canvas', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    expect(editor).toContain(':is-valid-connection="isValidConnection"')
    expect(editor).toContain('@connect-start="startConnection"')
    expect(editor).toContain('@connect-end="endConnection"')
    expect(editor).toContain('<WorkflowConnectionMenu')
    expect(editor).toContain('session.insertConnectedNode(')
    expect(editor).toContain("targetHandle: graphHandle(edge.channel, 'input'")
  })

  it('restores multi-selection, atomic batch editing, snapping, and auto-layout', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    expect(editor).toContain('@nodes-change="handleNodesChange"')
    expect(editor).toContain('<WorkflowSelectionToolbar')
    expect(editor).toContain("applyCommand({ kind: 'move-nodes'")
    expect(editor).toContain('snapNodePosition(')
    expect(editor).toContain('autoLayoutNodePositions(')
    expect(editor).toContain('session.duplicateNodes(')
    const selectionToolbar = readSource('src/app/editor/WorkflowSelectionToolbar.vue')
    expect(selectionToolbar).toContain('shrink-0 whitespace-nowrap')
  })

  it('projects subgraph boundaries from the canonical Source interface', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const boundary = readSource('src/app/editor/workflowGraphBoundary.ts')
    const panel = readSource('src/app/editor/WorkflowGraphInterfacePanel.vue')

    expect(editor).toContain('<WorkflowGraphBoundary')
    expect(editor).toContain('<WorkflowGraphInterfacePanel')
    expect(editor).toContain('session.bindGraphBoundary(boundary)')
    expect(boundary).toContain("type: 'graph-boundary'")
    expect(boundary).not.toContain("kind: 'add-node'")
    expect(panel).toContain('graph.entries')
    expect(panel).toContain('graph.exits')
  })

  it('keeps a new subgraph editable while explaining when interface refresh is unavailable', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const boundary = readSource('src/app/editor/WorkflowGraphBoundary.vue')

    expect(editor).toContain('canInferGraphInterface')
    expect(editor).toContain(':disabled="!canInferGraphInterface.valid"')
    expect(editor).toContain('v-if="session.currentGraph?.kind === \'main\'"')
    expect(editor).toContain('data-testid="workflow-subgraph-empty-hint"')
    expect(editor).toContain('pointer-events-none')
    expect(boundary).toContain('min-w-0 truncate')
  })

  it('restores source-native node search and canvas focus', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const toolbar = readSource('src/app/editor/WorkflowEditorToolbar.vue')
    expect(toolbar).toContain('data-testid="workflow-find-node"')
    expect(editor).toContain("if (key === 'f')")
    expect(editor).toContain('session.source?.graphs')
    expect(editor).toContain('await focusNode([result.graphId], result.nodeId)')
  })

  it('opens contextual quick add from Tab and inserts snippets through safe shortcuts', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const quickAdd = readSource('src/app/editor/WorkflowQuickAddMenu.vue')
    const snippetModal = readSource('src/app/editor/WorkflowSnippetModal.vue')
    expect(editor).toContain('<WorkflowQuickAddMenu')
    expect(editor).toContain("event.key === 'Tab'")
    expect(editor).toContain('@pointermove.capture="trackCanvasPointer"')
    expect(editor).toContain('shortcutFromKeyboardEvent(event)')
    expect(editor).toContain('useSnippet(snippet.id, canvasInsertionPosition())')
    expect(quickAdd).toContain('workflow-quick-add-search')
    expect(quickAdd).toContain('@keydown.down.prevent="move(1)"')
    expect(quickAdd).toContain('<Teleport to="body">')
    expect(quickAdd).toContain('@mouseenter="previewCategory(entry.value)"')
    expect(editor).toContain(':anchor="quickAddAnchor"')
    expect(snippetModal).toContain('<HotkeyCaptureInput')
  })

  it('uses the viewport-native context menu instead of a node-local hidden anchor', () => {
    const node = readSource('src/app/editor/WorkflowNode.vue')
    expect(node).toContain('<Teleport to="body">')
    expect(node).toContain('class="pointer-events-none fixed size-px opacity-0"')
    expect(node).toContain('x: event.clientX, y: event.clientY')
    expect(node).not.toContain('clientX - bounds.left')
  })

  it('keeps region fields readable and makes state initial values editable', () => {
    const region = readSource('src/app/editor/RegionValueEditor.vue')
    const state = readSource('src/app/editor/WorkflowStatePanel.vue')
    const valueEditor = readSource('src/app/editor/WorkflowValueEditor.vue')

    expect(region).toContain('grid-cols-2')
    expect(region).not.toContain('grid-cols-4')
    expect(valueEditor).toContain(":size=\"compact ? 'xs' : 'sm'\"")
    expect(state).toContain('<StateDefaultValueEditor')
    expect(state).toContain('newVariableDefault')
    expect(state).toContain('updateVariableDefault(variable, $event)')
  })
})
