import { createApp, defineComponent, h, nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { DebugSnapshot } from '@/app/transport/workflow'
import { DebugStatus } from '@bindings/github.com/yottaapp/yotta/internal/workflow/compiler/models.js'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params?.node ? `${key}:${String(params.node)}` : key,
    }),
  }
})

import WorkflowDebuggerPanel from './WorkflowDebuggerPanel.vue'

const mounted: Array<ReturnType<typeof createApp>> = []

afterEach(() => {
  for (const app of mounted.splice(0)) app.unmount()
  document.body.innerHTML = ''
})

describe('WorkflowDebuggerPanel', () => {
  it('puts Step and the next node at the top when the Run is paused', async () => {
    const root = await mountPanel(
      snapshot({ status: DebugStatus.DebugPaused, nodeId: 'play-clip' }),
    )

    expect(root.querySelector('[data-testid="workflow-debug-step"]')).toBeTruthy()
    expect(root.querySelector('[data-testid="workflow-debug-pause"]')).toBeNull()
    expect(root.textContent).toContain('workflow.debug.paused_before:Playback clip')
    expect(root.textContent).toContain('workflow.debug.will_execute')
  })

  it('uses terminal language instead of claiming that a node will still execute', async () => {
    const root = await mountPanel(
      snapshot({
        status: DebugStatus.DebugCompleted,
        runStatus: 'CANCELLED',
        nodeId: 'play-clip',
      }),
    )

    expect(root.textContent).toContain('workflow.debug.finished_cancelled')
    expect(root.textContent).toContain('workflow.debug.end_position')
    expect(root.textContent).not.toContain('workflow.debug.will_execute')
    expect(root.querySelector('[data-testid="workflow-debug-step"]')).toBeNull()
    expect(root.querySelector('[data-testid="workflow-debug-stop"]')).toBeNull()
  })
})

async function mountPanel(value: DebugSnapshot): Promise<HTMLElement> {
  const root = document.createElement('div')
  document.body.append(root)
  const app = createApp(WorkflowDebuggerPanel, {
    snapshot: value,
    embedded: true,
    nodeLabels: { 'play-clip': 'Playback clip' },
  })
  app.component(
    'UButton',
    defineComponent({
      inheritAttrs: false,
      props: { disabled: Boolean, label: String },
      emits: ['click'],
      setup(props, { attrs, emit, slots }) {
        return () =>
          h(
            'button',
            {
              ...attrs,
              disabled: props.disabled,
              onClick: () => emit('click'),
            },
            slots.default?.() ?? props.label,
          )
      },
    }),
  )
  app.component(
    'UBadge',
    defineComponent({
      setup:
        (_, { slots }) =>
        () =>
          h('span', slots.default?.()),
    }),
  )
  app.component('UIcon', defineComponent({ setup: () => () => h('span') }))
  mounted.push(app)
  app.mount(root)
  await nextTick()
  return root
}

function snapshot(patch: Partial<DebugSnapshot>): DebugSnapshot {
  return {
    status: DebugStatus.DebugRunning,
    runStatus: 'RUNNING',
    generation: 1,
    graphId: 'main',
    graphPath: ['main'],
    nodeId: '',
    previousNodeId: '',
    queue: [],
    inputs: {},
    outputs: {},
    state: {},
    ...patch,
  } as DebugSnapshot
}
