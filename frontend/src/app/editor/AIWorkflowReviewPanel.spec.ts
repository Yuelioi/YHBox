import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const panel = readFileSync(join(process.cwd(), 'src/app/editor/AIWorkflowReviewPanel.vue'), 'utf8')
const editor = readFileSync(join(process.cwd(), 'src/views/WorkflowEditorView.vue'), 'utf8')

describe('AI workflow authoring review boundary', () => {
  it('keeps conversation send, acceptance, rejection, and history recovery as separate calls', () => {
    expect(panel).toContain('backend.ai.sendConversationMessage')
    expect(panel).toContain('backend.ai.listConversations')
    expect(panel).toContain('backend.ai.createConversation')
    expect(panel).toContain('backend.ai.getConversation')
    expect(panel).toContain('backend.ai.acceptWorkflowProposal')
    expect(panel).toContain('backend.ai.rejectWorkflowProposal')
  })

  it('blocks conversation sends over unsaved editor state and renders review facts', () => {
    expect(panel).toContain('!props.dirty')
    expect(panel).toContain('review.permissions.added')
    expect(panel).toContain("message.review.status === 'proposed'")
    expect(panel).toContain('message.review.changes')
  })

  it('does not render an interactive model select when no profile is eligible', () => {
    expect(panel).toContain('v-if="profileOptions.length"')
    expect(panel).toContain('workflow.ai.configure_profile')
  })

  it('keeps conversations isolated by workflow and exposes live progress', () => {
    expect(panel).toContain('props.workflowId')
    expect(panel).toContain('ai-conversation-select')
    expect(panel).toContain('ai-conversation-new')
    expect(panel).toContain('ai-conversation-delete')
    expect(panel).toContain('backend.ai.deleteConversation')
    expect(panel).toContain('onAIConversationProgress')
  })

  it('renders durable Problems through the canonical error formatter with correlation ID', () => {
    expect(panel).toContain(
      'errorMessage({ id: message.problemId, params: message.problemParams })',
    )
    expect(panel).toContain("t('workflow.ai.operation_id', { id: message.operationId })")
    expect(panel).not.toContain("return te(key) ? t(key) : t('error.UNKNOWN_ERROR')")
  })

  it('reloads the durable source only after the accepted event', () => {
    expect(editor).toContain('@accepted="acceptAIProposal"')
    expect(editor).toContain('await session.load(session.workflowId)')
  })
})
