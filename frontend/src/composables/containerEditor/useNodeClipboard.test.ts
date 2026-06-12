// 单元测 useNodeClipboard.onCopySelection 纯逻辑部分:
//   - 过滤 Start 节点
//   - 仅复制两端都在 copyset 里的边
//   - 全局化后 Subgraph 节点不再深拷贝子图 (粘贴 = 引用同一个全局子图)
import { describe, it, expect, vi } from 'vitest'
// composable 内 useI18n() 只用于 toast/label 文案; 单元测纯逻辑, 把 useI18n mock 成 identity t
// (否则非组件 setup 调 useI18n 抛 "Must be called at top of setup function")。
// 保留模块其余导出 (createI18n 等被 app i18n setup 经 import 链用到, 不能整模块替掉)。
vi.mock('vue-i18n', async (importActual) => ({
  ...(await importActual<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (k: string) => k }),
}))
import { computed, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useNodeClipboard } from './useNodeClipboard'
import type { Container, Graph, GraphNode } from '@/lib/backend'
import type { FlowNode } from './useContainerDraft'

function makeNode(id: string, kind: string, config?: Record<string, any>): GraphNode {
  return { id, kind, x: 100, y: 100, config: config ?? {} }
}

function setup(opts: {
  draft: Container
  selected: Array<{ id: string }>
  sgs?: Array<{ id: string; label: string; graph: Graph; outputPins?: any[] }>
}) {
  setActivePinia(createPinia())
  const store = useContainerEditorStore()
  store.setActiveContainer(opts.draft.id)
  store.setPool(
    (opts.sgs ?? []).map((s) => ({
      id: s.id,
      rev: 1,
      label: s.label,
      outputPins: s.outputPins ?? [],
      entry: { nodeID: 'test-entry' },
      graph: s.graph,
      createdAt: '',
    })) as any,
  )
  const draftRef = ref<Container | null>(opts.draft)
  const activeGraph = computed<Graph | null>(() => draftRef.value?.graph ?? null)
  const flowNodes = ref<FlowNode[]>([])
  const getSelectedNodes = ref(opts.selected as any[])
  const toast = { add: vi.fn() }

  const cb = useNodeClipboard({
    draft: draftRef,
    activeGraph,
    flowNodes,
    syncFlowFromDraft: () => {},
    refreshSubgraphStore: async () => {},
    getSelectedNodes,
    genID: () => 'genid',
    toast,
  })
  return { cb, toast }
}

describe('useNodeClipboard.onCopySelection', () => {
  it('过滤 Start 节点不复制', () => {
    const draft: Container = {
      schemaVersion: 1,
      id: 'c1',
      name: '',
      graph: {
        id: 'g',
        version: 1,
        nodes: [makeNode('start', 'Start'), makeNode('n1', 'Sleep'), makeNode('n2', 'KeyPress')],
        edges: [],
      },
      createdAt: '',
      updatedAt: '',
    }
    const { cb, toast } = setup({
      draft,
      selected: [{ id: 'start' }, { id: 'n1' }, { id: 'n2' }],
    })
    cb.onCopySelection()
    expect(cb.clipboard.value?.nodes.map((n) => n.id).sort()).toEqual(['n1', 'n2'])
    expect(toast.add).toHaveBeenCalledOnce()
  })

  it('只复制内部边 (两端都在选中集里)', () => {
    const draft: Container = {
      schemaVersion: 1,
      id: 'c1',
      name: '',
      graph: {
        id: 'g',
        version: 1,
        nodes: [makeNode('n1', 'Sleep'), makeNode('n2', 'KeyPress'), makeNode('n3', 'Log')],
        edges: [
          { from: 'n1.out', to: 'n2.in' }, // 内部, 应复制
          { from: 'n2.out', to: 'n3.in' }, // 外部出边, n3 不选, 不复制
          { from: 'n3.out', to: 'n1.in' }, // 外部入边, n3 不选, 不复制
        ],
      },
      createdAt: '',
      updatedAt: '',
    }
    const { cb } = setup({ draft, selected: [{ id: 'n1' }, { id: 'n2' }] })
    cb.onCopySelection()
    expect(cb.clipboard.value?.edges).toHaveLength(1)
    expect(cb.clipboard.value?.edges[0]).toEqual({ from: 'n1.out', to: 'n2.in' })
  })

  it('全局化: Subgraph 节点复制保留引用, 不深拷贝子图', () => {
    const draft: Container = {
      schemaVersion: 1,
      id: 'c1',
      name: '',
      graph: {
        id: 'g',
        version: 1,
        nodes: [makeNode('call', 'Subgraph', { SubgraphID: 'sg-x' })],
        edges: [],
      },
      createdAt: '',
      updatedAt: '',
    }
    const sgs = [
      {
        id: 'sg-x',
        label: 'sub-x',
        graph: { id: 'gx', version: 1, nodes: [makeNode('inner', 'Sleep')], edges: [] },
      },
    ]
    const { cb } = setup({ draft, selected: [{ id: 'call' }], sgs })
    cb.onCopySelection()
    expect(cb.clipboard.value?.subgraphsDeepCopy).toEqual({})
    expect(cb.clipboard.value?.nodes[0].config.SubgraphID).toBe('sg-x')
  })

  it('选中 0 节点 → clipboard 不动 + toast 不弹', () => {
    const draft: Container = {
      schemaVersion: 1,
      id: 'c1',
      name: '',
      graph: { id: 'g', version: 1, nodes: [makeNode('n1', 'Sleep')], edges: [] },
      createdAt: '',
      updatedAt: '',
    }
    const { cb, toast } = setup({ draft, selected: [] })
    cb.onCopySelection()
    expect(cb.clipboard.value).toBeNull()
    expect(toast.add).not.toHaveBeenCalled()
  })

  it('只选中 Start → clipboard 不动 (filter 后无可复制)', () => {
    const draft: Container = {
      schemaVersion: 1,
      id: 'c1',
      name: '',
      graph: { id: 'g', version: 1, nodes: [makeNode('start', 'Start')], edges: [] },
      createdAt: '',
      updatedAt: '',
    }
    const { cb, toast } = setup({ draft, selected: [{ id: 'start' }] })
    cb.onCopySelection()
    expect(cb.clipboard.value).toBeNull()
    expect(toast.add).not.toHaveBeenCalled()
  })
})
