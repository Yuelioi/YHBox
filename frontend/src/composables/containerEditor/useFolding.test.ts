import { beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'

const h = vi.hoisted(() => ({
  persisted: null as any,
  create: vi.fn(),
  update: vi.fn(),
  get: vi.fn(),
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/composables/useConfirm', () => ({
  useConfirm: () => ({ confirm: async () => '折叠测试' }),
}))

vi.mock('@/lib/backend', () => ({
  backend: {
    subgraphs: {
      create: h.create,
      update: h.update,
      get: h.get,
    },
  },
}))

import type { Container, Graph, Subgraph } from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useFolding } from './useFolding'

function emptySubgraph(): Subgraph {
  return {
    id: 'sg-folded',
    rev: 1,
    label: '折叠测试',
    graph: { id: 'sg-graph', schemaVersion: 1, nodes: [], edges: [] },
    entry: { nodeID: 'entry' },
    outputPins: [{ id: 'done-decl', name: 'done', nodeID: 'output' }],
    createdAt: '2026-07-12T00:00:00Z',
  } as Subgraph
}

describe('useFolding', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    h.persisted = null
    vi.clearAllMocks()
    h.create.mockResolvedValue(emptySubgraph())
    h.update.mockImplementation(async (_id: string, patchJSON: string) => {
      const base = emptySubgraph()
      const patch = JSON.parse(patchJSON)
      h.persisted = { ...base, graph: patch.graph, rev: 2 }
    })
    h.get.mockImplementation(async () => h.persisted)
  })

  it('折叠后内存池保留移入子图的节点', async () => {
    const graph: Graph = {
      id: 'main',
      schemaVersion: 1,
      nodes: [{ id: 'click', kind: 'ClickAt', x: 120, y: 80, config: {} }],
      edges: [],
    } as Graph
    const draft = ref<Container>({ id: 'container', name: '测试', graph } as Container)
    const store = useContainerEditorStore()
    store.setActiveContainer('container')
    store.setPool([emptySubgraph()])

    const folding = useFolding({
      draft,
      activeGraph: computed(() => graph),
      refreshSubgraphStore: async () => store.mergePool([h.persisted]),
      syncFlowFromDraft: () => {},
      getSelectedNodes: ref([
        { id: 'click', data: { kind: 'ClickAt', config: {} }, position: { x: 120, y: 80 } },
      ]) as any,
      toast: { add: vi.fn() },
    })

    await folding.onFoldSelection()

    expect(h.persisted.graph.nodes.map((node: any) => node.id)).toContain('click')
    expect(store.subgraphById('sg-folded')?.graph.nodes.map((node) => node.id)).toContain('click')
  })
})
