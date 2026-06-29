import { describe, it, expect, beforeEach, vi } from 'vitest'
import { computed, ref } from 'vue'
import type { Container, Graph } from '@/lib/backend'
import { KIND_DEFAULTS } from '@/components/containers/pinSpec'
import { __resetForTests, register } from '@/components/containers/nodeRegistry/registry'

vi.mock('@vue-flow/core', () => ({
  useVueFlow: () => ({
    screenToFlowCoordinate: (p: { x: number; y: number }) => p,
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('./useInsertPoint', () => ({
  useInsertPoint: () => ({
    viewportCenterForNode: () => ({ x: 0, y: 0 }),
    screenPointToFlow: (p: { x: number; y: number }) => p,
  }),
}))

import { useNodeCreation } from './useNodeCreation'

function registerWindowTargetSpec() {
  register({
    kind: 'Win32WindowTarget',
    group: 'target',
    labelZh: 'node.Win32WindowTarget.label',
    description: '',
    example: '',
    visual: { icon: '', bg: '', border: '' },
    execIn: ['In'],
    execOut: ['Done'],
    dataIn: {},
    dataOut: {},
    fields: [],
    defaults: { literal: { Title: '', Class: '', ProcessName: '', TitleMatch: 'exact' } },
  })
  KIND_DEFAULTS.Win32WindowTarget = {
    literal: { Title: '', Class: '', ProcessName: '', TitleMatch: 'exact' },
  }
}

function setup() {
  const graph: Graph = { id: 'g', version: 1, nodes: [], edges: [] } as Graph
  const draft = ref<Container>({ id: 'c', name: 'c', graph } as Container)
  const activeGraph = computed<Graph | null>(() => graph)
  const selectedID = ref<string | null>(null)
  const api = useNodeCreation({
    draft,
    activeGraph,
    selectedID,
    applyDraftMutation: (mutator) => mutator(draft.value),
    syncFlowFromDraft: () => {},
    refreshSubgraphStore: async () => {},
    autoCreateSubgraphForNewNode: async () => true,
    toast: { add: () => {} },
  })
  return { api, graph }
}

describe('useNodeCreation default config cloning', () => {
  beforeEach(() => {
    __resetForTests()
    for (const k of Object.keys(KIND_DEFAULTS)) delete KIND_DEFAULTS[k]
    registerWindowTargetSpec()
  })

  it('onPickKind creates nodes with independent nested literal defaults', () => {
    const { api, graph } = setup()

    api.onPickKind('Win32WindowTarget')
    api.onPickKind('Win32WindowTarget')

    ;((graph.nodes[0].config as any).literal as Record<string, unknown>).Title = 'AE'

    expect((graph.nodes[1].config as any).literal.Title).toBe('')
    expect((graph.nodes[0].config as any).literal).not.toBe((graph.nodes[1].config as any).literal)
  })

  it('onAddNode creates nodes with independent nested literal defaults', async () => {
    const { api, graph } = setup()

    await api.onAddNode('Win32WindowTarget')
    await api.onAddNode('Win32WindowTarget')

    ;((graph.nodes[0].config as any).literal as Record<string, unknown>).Title = 'AE'

    expect((graph.nodes[1].config as any).literal.Title).toBe('')
    expect((graph.nodes[0].config as any).literal).not.toBe((graph.nodes[1].config as any).literal)
  })
})
