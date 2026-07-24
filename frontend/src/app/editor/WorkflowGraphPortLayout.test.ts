import { createApp, defineComponent, h, type Component } from 'vue'
import { createI18n } from 'vue-i18n'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Graph, GraphCall } from '../../../../contracts/workflow/current/workflow-source'
import WorkflowGraphBoundary from './WorkflowGraphBoundary.vue'
import WorkflowGraphCall from './WorkflowGraphCall.vue'
import type { GraphBoundaryNodeData } from './workflowGraphBoundary'

vi.mock('@vue-flow/core', async () => {
  const vue = await import('vue')
  return {
    Handle: vue.defineComponent({
      inheritAttrs: true,
      props: { position: { type: String, required: true } },
      setup(props) {
        return () => vue.h('span', { 'data-handle-position': props.position })
      },
    }),
    Position: { Left: 'left', Right: 'right' },
  }
})

const iconStub = defineComponent({ setup: () => () => h('i') })
const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: {
    zh: {
      workflow: {
        graphs: {
          boundary_authoring: '接口点',
          boundary_entry: '子图入口',
          boundary_output: '子图输出',
          boundary_exit: '子图出口',
        },
      },
    },
  },
})

afterEach(() => {
  document.body.replaceChildren()
})

describe('workflow graph port layout', () => {
  it('reserves label space beside every graph-call handle', () => {
    const graph = {
      id: 'child',
      name: 'Child',
      kind: 'subgraph',
      nodes: [],
      edges: [],
      inputs: [],
      outputs: [],
      exits: [
        { id: 'exit_completed_1', channel: 'exec', endpoint: { nodeId: 'keys', portId: 'done' } },
        { id: 'exit_failed_2', channel: 'error', endpoint: { nodeId: 'keys', portId: 'failed' } },
      ],
    } as Graph
    const call = {
      id: 'call-child',
      graphId: graph.id,
      label: 'Child',
      position: { x: 0, y: 0 },
      bindings: {},
    } as GraphCall

    const root = mount(WorkflowGraphCall, { graph, call })

    expectHandleClearance(root)
  })

  it('reserves label space beside entry, exit, and data boundary handles', () => {
    for (const boundary of [
      { role: 'entry', graphId: 'child', inputs: [], outputs: [] },
      {
        role: 'exit',
        graphId: 'child',
        inputs: [],
        outputs: [],
        exit: {
          id: 'exit_failed_2',
          channel: 'error',
          endpoint: { nodeId: 'keys', portId: 'failed' },
        },
      },
      {
        role: 'output',
        graphId: 'child',
        inputs: [],
        outputs: [
          {
            id: 'result',
            type: { kind: 'ref', ref: { typeId: 'result', semanticDigest: 'sha256:result' } },
            nodeId: 'keys',
            portId: 'result',
          },
        ],
      },
    ] satisfies GraphBoundaryNodeData[]) {
      const root = mount(WorkflowGraphBoundary, { boundary })
      expectHandleClearance(root)
    }
  })
})

function mount(component: Component, props: Record<string, unknown>): HTMLElement {
  const root = document.createElement('div')
  document.body.append(root)
  const app = createApp(component, props)
  app.component('UIcon', iconStub)
  app.use(i18n)
  app.mount(root)
  return root
}

function expectHandleClearance(root: HTMLElement): void {
  const handles = [...root.querySelectorAll<HTMLElement>('[data-handle-position]')]
  expect(handles.length).toBeGreaterThan(0)
  for (const handle of handles) {
    const row = handle.closest<HTMLElement>('.relative')
    expect(row).not.toBeNull()
    expect(
      row?.classList.contains(handle.dataset.handlePosition === 'left' ? 'pl-3' : 'pr-3'),
    ).toBe(true)
    const label = [...(row?.querySelectorAll<HTMLElement>('span') ?? [])].find(
      (candidate) => !candidate.dataset.handlePosition,
    )
    expect(label).toBeDefined()
    expect(label?.classList.contains('min-w-0')).toBe(true)
    expect(label?.classList.contains('truncate')).toBe(true)
  }
}
