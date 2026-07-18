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

  it('offers browser onboarding as a target template without changing the source model', () => {
    expect(source).toContain("'generic' | 'windows' | 'android' | 'browser' | 'cross-target'")
    expect(source).toContain("t('workflow.list.template_browser')")
    expect(source).toContain("query: template === 'generic' ? {} : { template }")
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

  it('imports, replaces, and exports portable Workflow Source bundles without success toasts', () => {
    expect(source).toContain('workflowTransport.inspectSourceBundle')
    expect(source).toContain('workflowTransport.importSourceBundle')
    expect(source).toContain('workflowTransport.replaceSourceFromBundle')
    expect(source).toContain('source.revision')
    expect(source).toContain('source.sourceHash')
    expect(source).toContain('workflowTransport.exportSourceBundle')
    expect(source).toContain('workflowTransport.exportSourceBundles')
    expect(source).toContain('portabilityFeedback.value')
    expect(source).not.toContain("title: t('workflow.list.export_result')")
  })

  it('keeps a corrupt source isolated while exposing bounded repair and delete actions', () => {
    expect(source).toContain('workflowTransport.listSourceRecoveries')
    expect(source).toContain('data-testid="workflow-recovery-panel"')
    expect(source).toContain('workflowTransport.repairSourceRecovery')
    expect(source).toContain('workflowTransport.deleteSourceRecovery')
    expect(source).toContain("confirmText: t('common.delete')")
    expect(source).not.toContain('delete data')
  })
})
