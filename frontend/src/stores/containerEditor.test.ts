// mergeSubgraphs 回归测 (录制雪崩根因③ — refresh 覆盖未落盘修改):
// refreshSubgraphStore 用后端磁盘快照同步子图时, 不能整体覆盖 —— 内存里正在编辑、还没落盘的子图
// 修改会被磁盘旧版盖掉、引用丢失。merge 语义: 后端列表为基准 (反映删除), 两边都有则保留内存版 (编辑真相)。
import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useContainerEditorStore } from './containerEditor'

function sg(id: string, nodes: any[] = []) {
  return {
    id,
    label: id,
    outputPins: [],
    entry: { nodeID: '' },
    graph: { id: 'g-' + id, version: 1, nodes, edges: [] },
  }
}

describe('useContainerEditorStore.mergeSubgraphs', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('两边都有的子图保留内存版 (不被后端快照覆盖编辑)', () => {
    const store = useContainerEditorStore()
    const memNode = { id: 'n', kind: 'Subgraph', x: 0, y: 0, config: { SubgraphID: 'sg-child' } }
    store.setActiveContainer('c1', [sg('sg-a', [memNode])] as any)
    // 后端快照里 sg-a 还是空的 (内存的新引用节点还没落盘)
    store.mergeSubgraphs('c1', [sg('sg-a', [])] as any)
    const a = store.subgraphsForCurrentContainer.find((s) => s.id === 'sg-a')
    expect(a?.graph.nodes).toHaveLength(1)
  })

  it('补进后端新增的子图', () => {
    const store = useContainerEditorStore()
    store.setActiveContainer('c1', [sg('sg-a')] as any)
    store.mergeSubgraphs('c1', [sg('sg-a'), sg('sg-new')] as any)
    expect(store.subgraphsForCurrentContainer.map((s) => s.id).sort()).toEqual(['sg-a', 'sg-new'])
  })

  it('移除后端已删的子图', () => {
    const store = useContainerEditorStore()
    store.setActiveContainer('c1', [sg('sg-a'), sg('sg-gone')] as any)
    store.mergeSubgraphs('c1', [sg('sg-a')] as any)
    expect(store.subgraphsForCurrentContainer.map((s) => s.id)).toEqual(['sg-a'])
  })
})
