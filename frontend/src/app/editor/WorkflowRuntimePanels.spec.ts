import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const read = (name: string) => readFileSync(join(process.cwd(), 'src/app/editor', name), 'utf8')
const toolbar = read('WorkflowEditorToolbar.vue')
const diagnostics = read('WorkflowDiagnosticsPanel.vue')
const timeline = read('RunTimelinePanel.vue')
const node = read('WorkflowNode.vue')
const editor = readFileSync(join(process.cwd(), 'src/views/WorkflowEditorView.vue'), 'utf8')

describe('workflow runtime inspection UI', () => {
  it('labels the admitted Run honestly and keeps both result panels recoverable', () => {
    expect(toolbar).not.toContain('workflow.action.debug')
    expect(toolbar).toContain('workflow.action.run_timeline')
    expect(toolbar).toContain("emit('toggle-diagnostics')")
    expect(toolbar).toContain("emit('toggle-timeline')")
  })

  it('groups compiler diagnostics and shows only compiler-declared fixes', () => {
    expect(diagnostics).toContain('groupDiagnostics')
    expect(diagnostics).toContain('diagnostic.fix')
    expect(diagnostics).not.toContain('suggestedFix')
    expect(editor).toContain('@focus="focusDiagnostic"')
  })

  it('locates timeline facts and projects journal-derived status onto nodes', () => {
    expect(timeline).toContain("emit('focus-node', entry.graphPath, entry.nodeId)")
    expect(editor).toContain('session.openGraphPath(graphPath)')
    expect(editor).toContain('await setCenter(')
    expect(node).toContain('data-testid="node-run-status"')
  })
})
