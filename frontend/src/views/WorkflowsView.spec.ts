import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/WorkflowsView.vue'), 'utf8')

describe('WorkflowsView entry points', () => {
  it('makes each workflow name a direct editor destination', () => {
    expect(source).toContain(':to="`/workflows/${source.workflowId}/edit`"')
    expect(source).toContain("t('workflow31.action.edit_named', { name: source.name })")
  })

  it('keeps creation and row actions reachable without a fixed four-column viewport', () => {
    expect(source).toContain('flex-col')
    expect(source).toContain('flex-wrap')
    expect(source).not.toContain('grid-cols-[minmax(220px,1fr)_88px_minmax(180px,0.8fr)_170px]')
    expect(source).not.toContain('overflow-hidden rounded-lg border border-default')
  })
})
