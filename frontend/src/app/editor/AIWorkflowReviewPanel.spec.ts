import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const panel = readFileSync(join(process.cwd(), 'src/app/editor/AIWorkflowReviewPanel.vue'), 'utf8')
const editor = readFileSync(join(process.cwd(), 'src/views/WorkflowEditorView.vue'), 'utf8')

describe('AI workflow authoring review boundary', () => {
  it('keeps proposal, acceptance, rejection, and trace recovery as separate calls', () => {
    expect(panel).toContain('backend.ai.proposeWorkflow')
    expect(panel).toContain('backend.ai.acceptWorkflowProposal')
    expect(panel).toContain('backend.ai.rejectWorkflowProposal')
    expect(panel).toContain('getWorkflowProposal')
  })

  it('blocks proposals over unsaved editor state and renders exact review facts', () => {
    expect(panel).toContain('!props.dirty')
    expect(panel).toContain('review.candidateHash')
    expect(panel).toContain('review.permissions.added')
    expect(panel).toContain('review.diagnostics')
    expect(panel).toContain('review.trace')
    expect(panel).toContain("review.status === 'proposed'")
  })

  it('reloads the durable source only after the accepted event', () => {
    expect(editor).toContain('@accepted="acceptAIProposal"')
    expect(editor).toContain('await session.load(session.workflowId)')
  })
})
