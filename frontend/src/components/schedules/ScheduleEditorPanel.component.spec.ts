import { readFileSync } from 'node:fs'
import { join } from 'node:path'
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
  it('keeps identity visible while reserving advanced disclosure for runtime limits', () => {
    const source = readFileSync(
      join(process.cwd(), 'src/components/schedules/ScheduleEditorPanel.vue'),
      'utf8',
    )
    const primary = source.slice(0, source.indexOf('<UCollapsible'))
    const advanced = source.slice(source.indexOf('<UCollapsible'), source.indexOf('<footer'))

    expect(source).toContain('data-testid="schedule-add-target"')
    expect(source).toContain('v-model="draft.trigger.kind"')
    expect(source).toContain('data-testid="schedule-save"')
    expect(primary).toContain('id="schedule-name"')
    expect(primary).toContain('v-model="draft.description"')
    expect(primary).toContain('v-model="draft.category"')
    expect(primary).toContain('v-model="draft.tags"')
    expect(primary).toContain('v-model="draft.enabled"')
    expect(advanced).not.toContain('id="schedule-name"')
    expect(advanced).toContain(':model-value="draft.timeoutMinutes"')
    expect(advanced).toContain('v-model="draft.onError"')
  })

  it('opens from a reactive schedule returned by the management view', () => {
    const schedule = reactive({
      schemaVersion: '4',
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
    expect(root.querySelector('.schedule-editor__summary')).toBeNull()
  })

  it('persists the visible interval default when the number field is untouched', async () => {
    const schedule = {
      schemaVersion: '4',
      id: 'schedule-2',
      name: 'Interval run',
      enabled: true,
      targets: [{ kind: 'workflow', id: 'workflow-1' }],
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
      workflows: [
        {
          workflowId: 'workflow-1',
          name: 'Workflow 1',
          description: '',
          category: '',
          tags: [],
          nodeCount: 0,
          revision: 0,
          sourceHash: '',
          createdAt: '2026-07-19T00:00:00Z',
          updatedAt: '2026-07-19T00:00:00Z',
          sourceJson: '',
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
