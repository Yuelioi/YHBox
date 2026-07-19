import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const read = (name: string) => readFileSync(join(process.cwd(), 'src/app/editor', name), 'utf8')
const toolbar = read('WorkflowEditorToolbar.vue')
const diagnostics = read('WorkflowDiagnosticsPanel.vue')
const timeline = read('RunTimelinePanel.vue')
const debuggerPanel = read('WorkflowDebuggerPanel.vue')
const workbench = read('WorkflowRuntimeWorkbench.vue')
const node = read('WorkflowNode.vue')
const editor = readFileSync(join(process.cwd(), 'src/views/WorkflowEditorView.vue'), 'utf8')

describe('workflow runtime inspection UI', () => {
  it('labels the admitted Run honestly and keeps both result panels recoverable', () => {
    expect(toolbar).not.toContain('workflow.action.debug')
    expect(toolbar).toContain('workflow.action.run_timeline')
    expect(toolbar).toContain("emit('toggle-diagnostics')")
    expect(toolbar).toContain("emit('toggle-timeline')")
  })

  it('uses the true debug transport and keeps breakpoints outside Workflow Source', () => {
    expect(toolbar).toContain("emit('start-debug')")
    expect(editor).toContain('session.startDebug(debugBreakpoints())')
    expect(editor).toContain('session.controlDebug(action)')
    expect(editor).toContain('breakpointKeys')
    expect(debuggerPanel).toContain("emit('step')")
    expect(debuggerPanel).toContain("emit('continue')")
    expect(debuggerPanel).toContain("emit('pause')")
    expect(debuggerPanel).toContain('workflow.debug.will_execute')
    expect(debuggerPanel).toContain('snapshot.previousNodeId')
  })

  it('explains breakpoints and keeps the inactive control out of the resting node chrome', () => {
    expect(node).toContain('<UTooltip')
    expect(node).toContain(':text="breakpointLabel"')
    expect(node).toContain("'opacity-0 group-hover/node:opacity-100 focus-within:opacity-100'")
    expect(node).toContain('debugMode || breakpoint')
  })

  it('keeps normal runs quiet and unifies logs, timeline, and debug in one workbench', () => {
    const startRun = editor.slice(
      editor.indexOf('async function startRun'),
      editor.indexOf('async function startDebug'),
    )
    expect(startRun).not.toContain("openRuntimeWorkbench('timeline')")
    expect(workbench).toContain("activate('logs')")
    expect(workbench).toContain("activate('timeline')")
    expect(workbench).toContain("activate('debug')")
    expect(workbench).toContain('<LogPanel v-if=')
    expect(editor).toContain("if (run?.failure) openRuntimeWorkbench('logs')")
    expect(editor).toContain(
      "if (event.snapshot.status === 'paused') openRuntimeWorkbench('debug')",
    )
  })

  it('represents an unset target through the placeholder instead of an invalid empty select item', () => {
    expect(editor).toContain('workflow.target_default.placeholder')
    expect(editor).toContain('workflow.target_default.clear')
    expect(editor).not.toContain("label: t('workflow.target_default.none'), value: ''")
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
    expect(timeline).toContain("emit('page', run.timelinePage + 1)")
  })

  it('renders structured run and RPC failures as localized messages', () => {
    expect(timeline).toContain('failureMessage')
    expect(timeline).toContain('`error.${props.run.failure.code}`')
    expect(editor).toContain('errorMessage(error)')
  })
})
