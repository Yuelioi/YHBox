import ui from '@nuxt/ui/vue-plugin'
import { createApp, defineComponent, h, nextTick, ref } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import LauncherSurface from './LauncherSurface.vue'
import type { ResolvedLauncherGroup } from './launcherModel'

const mounted: Array<ReturnType<typeof createApp>> = []

afterEach(() => {
  for (const app of mounted.splice(0)) app.unmount()
  document.body.innerHTML = ''
})

describe('LauncherSurface duplicate workflow entries', () => {
  it('activates duplicate entries by block id while running by workflow id', async () => {
    const groups: ResolvedLauncherGroup[] = [
      {
        id: 'group',
        label: '',
        items: [
          {
            id: 'first',
            workflowId: 'fishing',
            label: '自动钓鱼',
            icon: 'i-tabler-fish',
            shortcut: '',
            ordinal: 1,
          },
          {
            id: 'second',
            workflowId: 'fishing',
            label: '自动钓鱼',
            icon: 'i-tabler-fish',
            shortcut: '',
            ordinal: 2,
          },
        ],
      },
    ]
    const activeId = ref('first')
    const runs: string[] = []
    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(
      defineComponent({
        setup: () => () =>
          h(LauncherSurface, {
            groups,
            size: 'xsmall',
            activeId: activeId.value,
            emptyLabel: 'empty',
            runLabel: (name: string) => `run ${name}`,
            cancelLabel: (name: string) => `cancel ${name}`,
            statusLabels: {
              running: 'running',
              success: 'success',
              error: 'error',
              cancelled: 'cancelled',
            },
            staleLabel: 'stale',
            onSelect: (blockId: string) => {
              activeId.value = blockId
            },
            onRun: (workflowId: string) => runs.push(workflowId),
          }),
      }),
    )
    app.use(ui)
    mounted.push(app)
    app.mount(root)
    await nextTick()

    const commands = [...root.querySelectorAll<HTMLButtonElement>('.launcher-command')]
    expect(commands).toHaveLength(2)
    expect(commands[0]?.getAttribute('aria-current')).toBe('true')
    expect(commands[1]?.hasAttribute('aria-current')).toBe(false)
    expect(root.querySelector('.launcher-surface--xsmall')).toBeTruthy()

    commands[1]?.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true }))
    await nextTick()
    expect(commands[0]?.hasAttribute('aria-current')).toBe(false)
    expect(commands[1]?.getAttribute('aria-current')).toBe('true')

    commands[1]?.click()
    expect(runs).toEqual(['fishing'])
  })
})
