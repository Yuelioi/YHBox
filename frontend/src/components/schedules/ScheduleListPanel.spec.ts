import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  join(process.cwd(), 'src/components/schedules/ScheduleListPanel.vue'),
  'utf8',
)

describe('ScheduleListPanel structure', () => {
  it('keeps the table usable at compact widths', () => {
    expect(source).toContain('overflow-x-auto')
    expect(source).toContain('min-w-[760px]')
    expect(source).toContain('<caption class="sr-only">')
  })

  it('names edit and delete actions with schedule context', () => {
    expect(source).toContain(':aria-label="t(\'schedule.edit_action\', { name: s.name })"')
    expect(source).toContain(':aria-label="t(\'schedule.delete_action\', { name: s.name })"')
  })
})
