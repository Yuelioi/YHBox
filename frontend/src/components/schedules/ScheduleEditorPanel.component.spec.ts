import { createApp, nextTick, reactive } from 'vue'
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
      schemaVersion: '2',
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
    const app = createApp(ScheduleEditorPanel, { schedule, installations: [] })
    app.config.warnHandler = () => undefined
    mounted.push(app)

    expect(() => app.mount(root)).not.toThrow()
    expect(root.querySelector('.schedule-editor')).toBeTruthy()
    expect(root.querySelector('.schedule-editor__summary')).toBeNull()
  })

  it('persists the visible interval default when the number field is untouched', async () => {
    const schedule = {
      schemaVersion: '2',
      id: 'schedule-2',
      name: 'Interval run',
      enabled: true,
      targets: [{ kind: 'workflow-installation', id: 'installation-1' }],
      trigger: { kind: 'cron', subKind: 'interval' },
      timeoutMinutes: 0,
      onError: 'stop',
      createdAt: '2026-07-19T00:00:00Z',
      updatedAt: '2026-07-19T00:00:00Z',
    } as unknown as Schedule
    const save = vi.fn()
    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(ScheduleEditorPanel, {
      schedule,
      installations: [
        {
          installationId: 'installation-1',
          releaseId: 'sha256:1111111111111111111111111111111111111111111111111111111111111111',
          name: 'Workflow 1',
          lifecycle: 'active',
          createdAt: '2026-07-19T00:00:00Z',
          updatedAt: '2026-07-19T00:00:00Z',
        },
      ],
      onSave: save,
    })
    app.config.warnHandler = () => undefined
    mounted.push(app)
    app.mount(root)

    const saveButton = [...root.querySelectorAll('button, ubutton')].find((button) =>
      button.textContent?.includes('common.save'),
    )
    expect(saveButton).toBeTruthy()
    saveButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await nextTick()

    expect(save).toHaveBeenCalledOnce()
    expect(save.mock.calls[0]?.[0].trigger.everyMinutes).toBe(30)
  })
})
