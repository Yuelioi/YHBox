import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const viewSources = ['WorkflowsView.vue', 'AssetsView.vue', 'SchedulesView.vue'].map((name) =>
  readFileSync(join(process.cwd(), 'src/views', name), 'utf8'),
)

describe('management page shell', () => {
  it('uses one shared visual contract across primary library pages', () => {
    for (const source of viewSources) {
      expect(source).toContain('class="workspace-page"')
      expect(source).toContain('class="workspace-page__header"')
      expect(source).toContain('class="workspace-page__mark"')
      expect(source).toContain('class="workspace-page__eyebrow"')
      expect(source).toContain('class="workspace-page__title')
      expect(source).not.toContain('class="workspace-page__description"')
    }
  })

  it('keeps schedule creation and editing in a modal over the management list', () => {
    const source = viewSources[2] ?? ''
    expect(source).toContain('<BaseModal')
    expect(source).toContain(':open="!!editing"')
    expect(source).toContain('<ScheduleListPanel')
    expect(source).not.toContain('<template v-if="!editing">')
  })
})
