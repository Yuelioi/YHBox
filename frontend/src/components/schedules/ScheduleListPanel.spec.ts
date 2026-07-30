import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/schedules/ScheduleListPanel.vue'),
  'utf8',
)

describe('ScheduleListPanel structure', () => {
  it('uses the shared management table language with selectable rows', () => {
    expect(source).toContain('workspace-surface-strong')
    expect(source).toContain('role="table"')
    expect(source).toContain('workspace-table-row')
    expect(source).toContain('<UCheckbox')
    expect(source).toContain('<USwitch')
    expect(source).not.toContain('<table')
  })

  it('names edit and overflow actions with schedule context', () => {
    expect(source).toContain(':aria-label="t(\'schedule.run_action\', { name: schedule.name })"')
    expect(source).toContain(':aria-label="t(\'schedule.more_action\', { name: schedule.name })"')
    expect(source).toContain("label: t('schedule.edit_action', { name: schedule.name })")
    expect(source).toContain("label: t('schedule.delete_action', { name: schedule.name })")
  })

  it('keeps one-click run and the last readiness reason in the schedule row', () => {
    expect(source).toContain('data-testid="schedule-run"')
    expect(source).toContain('data-testid="schedule-readiness"')
    expect(source).toContain('data-testid="schedule-repair"')
    expect(source).toContain('runReadinessMessage(readinessOutcome(schedule.lastReadiness))')
  })

  it('supports metadata and date columns without a management-mode branch', () => {
    expect(source).toContain("isColumnVisible('category')")
    expect(source).toContain("isColumnVisible('tags')")
    expect(source).toContain("isColumnVisible('createdAt')")
    expect(source).toContain("isColumnVisible('updatedAt')")
    expect(source).not.toContain('manageMode')
  })
})
