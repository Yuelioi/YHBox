import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/WorkflowsView.vue'), 'utf8')
const assetsSource = readFileSync(join(process.cwd(), 'src/views/AssetsView.vue'), 'utf8')

describe('WorkflowsView entry points', () => {
  it('makes each workflow name a direct editor destination', () => {
    expect(source).toContain(':to="`/workflows/${source.workflowId}/edit`"')
    expect(source).toContain("t('workflow.action.edit_named', { name: source.name })")
  })

  it('uses one create modal and a configurable list without grid or hotkey columns', () => {
    expect(source).toContain('data-testid="workflow-new-button"')
    expect(source).toContain('v-model:open="metadataModalOpen"')
    expect(source).toContain('<UPagination')
    expect(source).toContain('columnMenuItems')
    expect(source).toContain("key: 'createdAt'")
    expect(source).toContain("key: 'updatedAt'")
    expect(source).not.toContain("viewMode === 'grid'")
    expect(source).not.toContain("key: 'hotkey'")
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
    expect(source).toContain("categoryFilter.value === allCategories ? '' : categoryFilter.value")
    expect(source).toContain('tags: tagFilters.value')
    expect(source).toContain('createdSince: rangeStart(createdRange.value)')
    expect(source).toContain('updatedSince: rangeStart(updatedRange.value)')
    expect(source).toContain('page: page.value')
    expect(source).toContain('pageSize: pageSize.value')
    expect(source).toContain('toggleCurrentPage')
    expect(source).not.toContain('sources.value.slice(')
  })

  it('uses content-aware selects for templates and list controls', () => {
    expect(source).toContain("import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'")
    expect(source).toContain('v-model="metadataDraft.template"')
    expect(source).toContain(':items="templateItems"')
    const filterStart = source.indexOf('v-model="categoryFilter"')
    const filterEnd = source.indexOf('</section>', filterStart)
    expect(source.slice(filterStart, filterEnd)).not.toContain('width-mode="fixed"')
  })

  it('shows one editable workflow library without installation concepts', () => {
    expect(source).not.toContain('data-testid="workflow-installations"')
    expect(source).not.toContain('loadInstallations')
    expect(source).not.toContain('Installation')
    expect(source).not.toContain('readOnly')
    expect(assetsSource.match(/i-tabler-refresh/g)).toHaveLength(1)
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
