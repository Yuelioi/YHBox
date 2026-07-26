import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/schedules/ScheduleListPanel.vue'),
  'utf8',
)

describe('ScheduleListPanel structure', () => {
  it('uses an operational list instead of a compact data table', () => {
    expect(source).toContain('class="schedule-list"')
    expect(source).toContain('role="list"')
    expect(source).toContain('class="schedule-row"')
    expect(source).toContain('<USwitch')
    expect(source).not.toContain('<table')
  })

  it('names edit and overflow actions with schedule context', () => {
    expect(source).toContain(':aria-label="t(\'schedule.edit_action\', { name: schedule.name })"')
    expect(source).toContain(':aria-label="t(\'schedule.more_action\', { name: schedule.name })"')
    expect(source).toContain("label: t('schedule.delete_action', { name: schedule.name })")
  })
})
