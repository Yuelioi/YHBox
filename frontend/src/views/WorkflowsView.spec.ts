import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/WorkflowsView.vue'), 'utf8')

describe('WorkflowsView entry points', () => {
  it('makes each workflow name a direct editor destination', () => {
    expect(source).toContain(':to="`/workflows/${source.workflowId}/edit`"')
    expect(source).toContain("t('workflow.action.edit_named', { name: source.name })")
  })

  it('keeps creation and row actions reachable without a fixed four-column viewport', () => {
    expect(source).toContain('flex-col')
    expect(source).toContain('flex-wrap')
    expect(source).not.toContain('grid-cols-[minmax(220px,1fr)_88px_minmax(180px,0.8fr)_170px]')
    expect(source).toContain('sm:grid-cols-[auto_minmax(0,1fr)_auto]')
  })

  it('keeps queued Run feedback on the workflow row instead of a success toast', () => {
    expect(source).toContain('runFeedbackById[source.workflowId]')
    expect(source).toContain("label: t('workflow.toast.queued')")
    expect(source).not.toContain("title: t('workflow.toast.queued')")
  })

  it('queries a server-side page and preserves explicit cross-page selection', () => {
    expect(source).toContain('workflowTransport.querySources')
    expect(source).toContain('search: search.value')
    expect(source).toContain('page: page.value')
    expect(source).toContain('pageSize: pageSize.value')
    expect(source).toContain('selection_scope_hint')
    expect(source).toContain('toggleCurrentPage')
    expect(source).not.toContain('.slice(')
  })

  it('previews references and performs CAS-protected partial batch deletion', () => {
    expect(source).toContain('workflowTransport.previewDeleteSources')
    expect(source).toContain('workflowTransport.deleteSources')
    expect(source).toContain('revision: row.revision')
    expect(source).toContain('sourceHash: row.sourceHash')
    expect(source).toContain('workflow.list.reference_${reference.kind}')
    expect(source).toContain("confirmText: t('common.delete')")
    expect(source).not.toContain("title: t('workflow.list.delete_result')")
  })
})
