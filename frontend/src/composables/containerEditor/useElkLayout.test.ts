import { beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, shallowRef, type ComputedRef } from 'vue'
import { createPinia, setActivePinia } from 'pinia'

const h = vi.hoisted(() => ({
  loadElk: vi.fn(),
  findNode: vi.fn(),
  fitView: vi.fn(),
}))

vi.mock('@vue-flow/core', () => ({
  useVueFlow: () => ({ findNode: h.findNode, fitView: h.fitView }),
}))

vi.mock('./elkConfig', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./elkConfig')>()),
  loadElk: h.loadElk,
}))

import type { Container, Graph, Subgraph } from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useElkLayout } from './useElkLayout'

type Deferred<T> = {
  promise: Promise<T>
  resolve: (value: T) => void
}

type LayoutResult = {
  children: Array<{ id: string; x: number; y: number; width: number; height: number }>
}

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => {
    resolve = done
  })
  return { promise, resolve }
}

function mainGraph(id: string): Graph {
  return {
    id,
    schemaVersion: 1,
    nodes: [
      { id: `${id}-start`, kind: 'Start', x: 0, y: 0, config: {} },
      { id: `${id}-log`, kind: 'Log', x: 300, y: 0, config: {} },
    ],
    edges: [{ from: `${id}-start.Done`, to: `${id}-log.In` }],
  } as Graph
}

function subgraph(id: string, graph: Graph, entryID: string, x: number): Subgraph {
  return {
    id,
    rev: 1,
    label: id,
    graph,
    entry: { nodeID: entryID, x, y: 0 },
    outputPins: [],
    createdAt: '2026-07-13T00:00:00Z',
  } as Subgraph
}

function markerGraph(id: string, entryID: string): Graph {
  return {
    id,
    schemaVersion: 1,
    nodes: [{ id: `${id}-body`, kind: 'Log', x: 100, y: 0, config: {} }],
    edges: [{ from: `${entryID}.Out`, to: `${id}-body.In` }],
  } as Graph
}

function setup(activeGraph: ComputedRef<Graph | null>) {
  const applyDraftMutation = vi.fn((mutator: (draft: Container) => void) => {
    mutator({} as Container)
  })
  const toast = { add: vi.fn() }
  const layout = useElkLayout({
    activeGraph,
    applyDraftMutation,
    toast,
    t: (key) => key,
  })
  return { ...layout, applyDraftMutation, toast }
}

describe('useElkLayout async editor context', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    h.findNode.mockReturnValue({ dimensions: { width: 220, height: 90 } })
    const store = useContainerEditorStore()
    store.setActiveContainer('container-a')
  })

  it('discards the old graph while the lazy ELK engine is loading', async () => {
    const engineLoad = deferred<{ layout: ReturnType<typeof vi.fn> }>()
    const engine = { layout: vi.fn() }
    h.loadElk.mockReturnValue(engineLoad.promise)
    const first = mainGraph('first')
    const second = mainGraph('second')
    const current = shallowRef<Graph | null>(first)
    const { autoLayout, applyDraftMutation, isLayouting, toast } = setup(
      computed(() => current.value),
    )

    const pending = autoLayout()
    expect(isLayouting.value).toBe(true)
    current.value = second
    engineLoad.resolve(engine)
    await pending

    expect(engine.layout).not.toHaveBeenCalled()
    expect(applyDraftMutation).not.toHaveBeenCalled()
    expect(toast.add).not.toHaveBeenCalled()
    expect(second.nodes.map(({ x, y }) => [x, y])).toEqual([
      [0, 0],
      [300, 0],
    ])
    expect(isLayouting.value).toBe(false)
  })

  it('discards old graph and marker results when the editor path changes during layout', async () => {
    const store = useContainerEditorStore()
    const graphA = markerGraph('graph-a', 'entry-a')
    const graphB = markerGraph('graph-b', 'entry-b')
    store.setPool([subgraph('sg-a', graphA, 'entry-a', 0), subgraph('sg-b', graphB, 'entry-b', 20)])
    store.setPath(['sg-a'])

    const layoutResult = deferred<LayoutResult>()
    const engine = { layout: vi.fn(() => layoutResult.promise) }
    h.loadElk.mockResolvedValue(engine)
    // 保持 graph 身份不变，单独证明 editor path 也是异步结果的有效性条件。
    const { autoLayout, applyDraftMutation, toast } = setup(computed(() => graphA))

    const pending = autoLayout()
    await vi.waitFor(() => expect(engine.layout).toHaveBeenCalledOnce())
    store.setPath(['sg-b'])
    layoutResult.resolve({
      children: [
        { id: 'entry-a', x: 50, y: 0, width: 220, height: 90 },
        { id: 'graph-a-body', x: 450, y: 0, width: 220, height: 90 },
      ],
    })
    await pending

    expect(applyDraftMutation).not.toHaveBeenCalled()
    expect(toast.add).not.toHaveBeenCalled()
    expect(graphA.nodes[0]).toMatchObject({ x: 100, y: 0 })
    expect(store.subgraphById('sg-a')?.entry).toMatchObject({ x: 0, y: 0 })
    expect(store.subgraphById('sg-b')?.entry).toMatchObject({ x: 20, y: 0 })
    expect(store.touchedFor('container-a')).toEqual([])
  })

  it('applies a current layout to the captured graph and marker owner', async () => {
    const store = useContainerEditorStore()
    const graph = markerGraph('graph-a', 'entry-a')
    store.setPool([subgraph('sg-a', graph, 'entry-a', 0)])
    store.setPath(['sg-a'])
    const engine = {
      layout: vi.fn().mockResolvedValue({
        children: [
          { id: 'entry-a', x: 50, y: 0, width: 220, height: 90 },
          { id: 'graph-a-body', x: 450, y: 0, width: 220, height: 90 },
        ],
      }),
    }
    h.loadElk.mockResolvedValue(engine)
    const { autoLayout, applyDraftMutation } = setup(computed(() => graph))

    await autoLayout('LR', { fitView: true })

    expect(applyDraftMutation).toHaveBeenCalledOnce()
    expect(graph.nodes[0]).toMatchObject({ x: 250, y: 0 })
    expect(store.subgraphById('sg-a')?.entry).toMatchObject({ x: -150, y: 0 })
    expect(store.touchedFor('container-a')).toEqual(['sg-a'])
    expect(h.fitView).toHaveBeenCalledOnce()
  })
})
