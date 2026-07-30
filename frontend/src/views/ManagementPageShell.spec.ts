import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const viewSources = ['WorkflowsView.vue', 'AssetsView.vue', 'SchedulesView.vue'].map((name) =>
  readFileSync(join(process.cwd(), 'src/views', name), 'utf8'),
)
const styles = readFileSync(join(process.cwd(), 'src/style.css'), 'utf8')
const assetList = readFileSync(
  join(process.cwd(), 'src/components/assets/AssetLibraryList.vue'),
  'utf8',
)

describe('management page shell', () => {
  it('uses one shared shell while allowing task-specific secondary context', () => {
    for (const source of viewSources) {
      expect(source).toContain('class="workspace-page workspace-canvas')
      expect(source).toMatch(/class="[^"]*\bworkspace-page__header\b[^"]*"/)
      expect(source).toMatch(/class="[^"]*\bworkspace-page__mark\b[^"]*"/)
      expect(source).toMatch(/class="[^"]*\bworkspace-page__title\b[^"]*"/)
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

  it('keeps schedules and assets directly manageable without legacy mode switches', () => {
    const source = viewSources[2] ?? ''
    expect(source).not.toContain('data-testid="schedule-manage-button"')
    expect(source).not.toContain('manageMode')
    expect(source).toContain('data-mode="manage"')
    expect(source).toContain('<UPagination')
    expect(source).toContain('columnMenuItems')
    expect(viewSources[1]).not.toContain('data-testid="asset-manage-button"')
    expect(viewSources[1]).not.toContain('managementMode')
    expect(viewSources[1]).toContain('data-mode="manage"')
  })

  it('uses one brighter surface for every management table', () => {
    expect(styles).toContain('--ui-surface:')
    expect(styles).toContain('@utility workspace-surface {')
    expect(viewSources[0]).toContain('class="workspace-surface')
    expect(viewSources[0]).toContain('class="workspace-table-row')
    expect(viewSources[1]).toContain('class="workspace-surface')
    expect(assetList).toContain('workspace-surface-strong')
    expect(assetList).toContain('workspace-table-row')
    expect(viewSources[2]).toContain('class="workspace-surface')
    expect(viewSources[2]).toContain('<ScheduleListPanel')
  })
})
