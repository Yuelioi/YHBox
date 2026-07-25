import { createApp, defineComponent, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it } from 'vitest'
import type { Graph } from '../../../../contracts/workflow/current/workflow-source'
import zh from '../../i18n/zh'
import WorkflowGraphManager from './WorkflowGraphManager.vue'

afterEach(() => {
  document.body.replaceChildren()
})

describe('WorkflowGraphManager', () => {
  it('shows definition identity, reference counts, and locatable call sites', async () => {
    const located: string[][] = []
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

    const root = mount(source, (...args) => located.push(args))
    await nextTick()
    const surface = root

    expect(surface.textContent).toContain('子图管理')
    expect(surface.textContent).toContain('2 个调用')
    expect(surface.textContent).toContain('主图 · 第一次调用')
    expect(surface.textContent).toContain('子图 · 嵌套调用')
    expect(surface.textContent).toContain('child-a')
    expect(surface.textContent).toContain('child-b')
    expect(surface.querySelector('[data-testid="workflow-graph-new"]')).not.toBeNull()

    const referencedDelete = [
      ...surface.querySelectorAll<HTMLButtonElement>('button[aria-label="删除子图定义"][disabled]'),
    ]
    expect(referencedDelete).toHaveLength(2)
    expect(
      [...surface.querySelectorAll<HTMLButtonElement>('button[aria-label="删除子图定义"]')].some(
        (button) => !button.disabled,
      ),
    ).toBe(true)
    expect(
      surface.querySelectorAll<HTMLButtonElement>('button[aria-label="复制子图定义"]'),
    ).toHaveLength(3)
    expect(
      surface.querySelectorAll<HTMLButtonElement>('button[aria-label="删除定义和全部调用"]'),
    ).toHaveLength(2)

    const callLocation = [...surface.querySelectorAll<HTMLButtonElement>('button')].find((button) =>
      button.textContent?.includes('主图 · 第一次调用'),
    )
    callLocation?.click()
    await nextTick()
    expect(located).toEqual([['main', 'call-a']])
  })
})

function mount(
  source: { entryGraph: string; graphs: Graph[] },
  locate: (...args: string[]) => void,
) {
  const root = document.createElement('div')
  document.body.append(root)
  const app = createApp(WorkflowGraphManager, {
    source,
    currentGraphId: 'main',
    onLocate: locate,
  })
  app.component('UIcon', defineComponent({ setup: () => () => h('i') }))
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
