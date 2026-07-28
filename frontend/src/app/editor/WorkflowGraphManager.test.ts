import { createApp, defineComponent, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Graph } from '../../../../contracts/workflow/current/workflow-source'
import zh from '../../i18n/zh'
import WorkflowGraphManager from './WorkflowGraphManager.vue'

afterEach(() => {
  document.body.replaceChildren()
})

describe('WorkflowGraphManager', () => {
  it('shows definition identity, reference counts, and locatable call sites', async () => {
    const located: string[][] = []
    const inserted: string[] = []
    const source = {
      entryGraph: 'main',
      graphs: [
        graph('main', 'main', '主图', [
          call('call-a', 'child-a', '第一次调用'),
          call('call-b', 'child-b', '第二次调用'),
        ]),
        graph('child-a', 'subgraph', '子图'),
        graph('child-b', 'subgraph', '子图', [call('nested-a', 'child-a', '嵌套调用')]),
        graph('orphan', 'subgraph', '未使用定义'),
      ],
    }

    const root = mount(
      source,
      (...args) => located.push(args),
      (graphId) => inserted.push(graphId),
    )
    await nextTick()
    const surface = root

    expect(surface.textContent).toContain('子图管理')
    expect(surface.textContent).toContain('2 个调用')
    expect(surface.textContent).toContain('child-a')
    expect(surface.textContent).toContain('child-b')
    expect(surface.querySelector('[data-testid="workflow-graph-new"]')).not.toBeNull()
    expect(surface.textContent).not.toContain('主图 · 第一次调用')
    expect(surface.querySelectorAll('[data-testid="workflow-graph-insert-call"]')).toHaveLength(3)
    expect(surface.querySelectorAll('button[aria-label*="更多操作"]')).toHaveLength(3)

    const callButton = [...surface.querySelectorAll<HTMLButtonElement>('button')].find((button) =>
      button.getAttribute('aria-label')?.includes('子图'),
    )
    callButton?.click()
    await nextTick()
    expect(inserted).toEqual(['child-a'])
    expect(located).toEqual([])
  })

  it('drags callable definitions as graph calls while keeping the current graph non-draggable', async () => {
    const source = {
      entryGraph: 'main',
      graphs: [
        graph('main', 'main', '主图'),
        graph('child-a', 'subgraph', '可调用子图'),
        graph('child-b', 'subgraph', '当前子图'),
      ],
    }
    const root = mount(
      source,
      () => undefined,
      () => undefined,
      'child-b',
      ['child-a'],
    )
    await nextTick()

    const buttons = [...root.querySelectorAll<HTMLButtonElement>('button')]
    const callable = buttons.find((button) => button.textContent?.includes('可调用子图'))
    const current = buttons.find((button) => button.textContent?.includes('当前子图'))
    expect(callable, root.innerHTML).toBeDefined()
    expect(current, root.innerHTML).toBeDefined()
    expect(callable?.getAttribute('draggable')).toBe('true')
    expect(current?.getAttribute('draggable')).toBe('false')

    const setData = vi.fn()
    const event = new Event('dragstart', { bubbles: true })
    Object.defineProperty(event, 'dataTransfer', {
      value: { effectAllowed: 'none', setData },
    })
    callable?.dispatchEvent(event)
    expect(setData).toHaveBeenCalledWith('application/x-yotta-graph-call', 'child-a')
  })
})

function mount(
  source: { entryGraph: string; graphs: Graph[] },
  locate: (...args: string[]) => void,
  insert: (graphId: string) => void,
  currentGraphId = 'main',
  callableGraphIds = source.graphs
    .filter((graph) => graph.kind === 'subgraph')
    .map((graph) => graph.id),
) {
  const root = document.createElement('div')
  document.body.append(root)
  const app = createApp(WorkflowGraphManager, {
    source,
    currentGraphId,
    callableGraphIds,
    onLocate: locate,
    onInsert: insert,
  })
  app.component('UIcon', defineComponent({ setup: () => () => h('i') }))
  app.component(
    'UDropdownMenu',
    defineComponent({
      setup(_, { slots }) {
        return () => h('div', slots.default?.())
      },
    }),
  )
  app.component(
    'UButton',
    defineComponent({
      inheritAttrs: true,
      props: {
        label: { type: String, default: '' },
        disabled: { type: Boolean, default: false },
      },
      setup(props, { attrs }) {
        return () => h('button', { ...attrs, disabled: props.disabled }, props.label)
      },
    }),
  )
  app.component(
    'UInput',
    defineComponent({
      inheritAttrs: true,
      props: { modelValue: { type: String, default: '' } },
      setup(props, { attrs }) {
        return () => h('input', { ...attrs, value: props.modelValue })
      },
    }),
  )
  app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh } }))
  app.use(
    createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: defineComponent({ render: () => null }) }],
    }),
  )
  app.mount(root)
  return root
}

function graph(
  id: string,
  kind: Graph['kind'],
  name: string,
  calls: NonNullable<Graph['calls']> = [],
): Graph {
  return {
    id,
    kind,
    name,
    nodes: [],
    calls,
    edges: [],
    inputs: [],
    outputs: [],
    entries: kind === 'subgraph' ? [{ nodeId: 'node', portId: 'in' }] : [],
    exits:
      kind === 'subgraph'
        ? [{ id: 'done', channel: 'exec', endpoint: { nodeId: 'node', portId: 'done' } }]
        : [],
    annotations: [],
  }
}

function call(id: string, graphId: string, label: string): NonNullable<Graph['calls']>[number] {
  return { id, graphId, label, position: { x: 0, y: 0 }, bindings: {} }
}
