import { createApp, defineComponent, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, describe, expect, it } from 'vitest'
import type { Graph } from '../../../../contracts/workflow/current/workflow-source'
import zh from '../../i18n/zh'
import WorkflowGraphInterfacePanel from './WorkflowGraphInterfacePanel.vue'
import type { GraphInterfaceCandidate } from './subgraphInterface'

afterEach(() => {
  document.body.replaceChildren()
})

describe('WorkflowGraphInterfacePanel', () => {
  it('edits display names while exposing stable IDs and protected references', async () => {
    const renamed: unknown[][] = []
    const removed: unknown[][] = []
    const moved: unknown[][] = []
    const added: unknown[][] = []
    const root = mount({
      onRename: (...args: unknown[]) => renamed.push(args),
      onRemove: (...args: unknown[]) => removed.push(args),
      onMove: (...args: unknown[]) => moved.push(args),
      onAdd: (...args: unknown[]) => added.push(args),
    })

    expect(root.textContent).toContain('ID stable-input')
    expect(root.textContent).toContain('2')

    const name = root.querySelector<HTMLInputElement>('input[aria-label="接口显示名称"]')!
    expect(name.value).toBe('目标文本')
    name.value = '新名称'
    name.dispatchEvent(new Event('change', { bubbles: true }))
    await nextTick()
    expect(renamed).toContainEqual(['input', 'stable-input', '新名称'])

    const protectedDelete = root.querySelector<HTMLButtonElement>(
      'button[aria-label="移除接口"][disabled]',
    )
    expect(protectedDelete).not.toBeNull()

    root.querySelector<HTMLButtonElement>('button[aria-label="下移接口"]')?.click()
    root.querySelector<HTMLButtonElement>('button[aria-label="解除子图入口"]')?.click()
    await nextTick()
    expect(moved).toContainEqual(['input', 'stable-input', 1])
    expect(removed).toContainEqual(['entry', ''])

    root.querySelector<HTMLButtonElement>('button[aria-label="添加输入"]')?.click()
    await nextTick()
    await nextTick()
    const candidate = [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')].find(
      (item) => item.textContent?.includes('第二个节点 · 备用文本'),
    )
    candidate?.click()
    await nextTick()
    expect(added).toEqual([['input:second:value']])
  })
})

function mount(handlers: Record<string, (...args: unknown[]) => void>): HTMLElement {
  const root = document.createElement('div')
  document.body.append(root)
  const app = createApp(WorkflowGraphInterfacePanel, {
    graph: graph(),
    candidates: candidates(),
    referenceCounts: { 'input:stable-input': 2 },
    ...handlers,
  })
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

function graph(): Graph {
  return {
    id: 'child',
    name: '子图',
    kind: 'subgraph',
    nodes: [],
    calls: [],
    edges: [],
    entries: [{ nodeId: 'first', portId: 'in' }],
    inputs: [
      {
        id: 'stable-input',
        name: '目标文本',
        type: stringType(),
        nodeId: 'first',
        portId: 'value',
      },
      {
        id: 'second-input',
        name: '第二文本',
        type: stringType(),
        nodeId: 'third',
        portId: 'value',
      },
    ],
    outputs: [],
    exits: [
      {
        id: 'stable-done',
        name: '完成',
        channel: 'exec',
        endpoint: { nodeId: 'first', portId: 'done' },
      },
    ],
    annotations: [],
  }
}

function candidates(): GraphInterfaceCandidate[] {
  return [
    {
      key: 'input:first:value',
      kind: 'input',
      endpoint: { nodeId: 'first', portId: 'value' },
      elementLabel: '第一个节点',
      name: '目标文本',
      type: stringType(),
      published: true,
    },
    {
      key: 'input:second:value',
      kind: 'input',
      endpoint: { nodeId: 'second', portId: 'value' },
      elementLabel: '第二个节点',
      name: '备用文本',
      type: stringType(),
      published: false,
    },
  ]
}

function stringType() {
  return {
    kind: 'ref' as const,
    ref: {
      typeId: 'https://schemas.yotta.dev/types/core/string/v1',
      semanticDigest: `sha256:${'1'.repeat(64)}`,
    },
  }
}
