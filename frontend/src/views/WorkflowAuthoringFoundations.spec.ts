import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const readSource = (path: string) => readFileSync(join(process.cwd(), path), 'utf8')

describe('workflow authoring foundations', () => {
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
    expect(editor).toContain('<WorkflowStatePanel')
    expect(inspector).not.toContain("kind: 'add-state-variable'")
    expect(inspector).toContain(':select-items="targetItems(field.id)"')
    expect(inspector).toContain(
      "path: '/settings', query: { section: targetSettingsSection(field.id) }",
    )
    expect(inspector).toContain('projectionDescription')
  })

  it('restores a main-window library for clips, templates, and recording', () => {
    const source = readSource('src/views/AssetsView.vue')
    const router = readSource('src/router/index.ts')
    expect(router).toContain("path: '/assets'")
    expect(source).toContain('recording.start(selectedTargetSlot.value)')
    expect(source).toContain("openScreenPicker('template_save'")
    expect(source).toContain('clipsStore.update')
    expect(source).toContain('templatesStore.updateMeta')
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
  })

  it('restores source-native node search and canvas focus', () => {
    const editor = readSource('src/views/WorkflowEditorView.vue')
    const toolbar = readSource('src/app/editor/WorkflowEditorToolbar.vue')
    expect(toolbar).toContain('data-testid="workflow-find-node"')
    expect(editor).toContain("if (key === 'f')")
    expect(editor).toContain('session.source?.graphs')
    expect(editor).toContain('await focusNode([result.graphId], result.nodeId)')
  })
})
