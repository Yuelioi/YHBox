import { createApp, reactive } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Schedule } from '@/lib/backend'
import ScheduleEditorPanel from './ScheduleEditorPanel.vue'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const mounted: Array<ReturnType<typeof createApp>> = []

afterEach(() => {
  for (const app of mounted.splice(0)) app.unmount()
  document.body.innerHTML = ''
})

describe('ScheduleEditorPanel', () => {
  it('opens from a reactive schedule returned by the management view', () => {
    const schedule = reactive({
      schemaVersion: '3.1',
      id: 'schedule-1',
      name: 'Morning run',
      enabled: true,
      targets: [],
      trigger: { kind: 'manual', subKind: '', at: '', everyMinutes: 0, hotkey: '' },
      timeoutMinutes: 0,
      onError: 'stop',
      createdAt: '2026-07-19T00:00:00Z',
      updatedAt: '2026-07-19T00:00:00Z',
    }) as unknown as Schedule
    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(ScheduleEditorPanel, { schedule, workflows: [] })
    app.config.warnHandler = () => undefined
    mounted.push(app)

    expect(() => app.mount(root)).not.toThrow()
    expect(root.querySelector('.schedule-editor')).toBeTruthy()
  })
})
