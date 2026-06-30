// mergePool 回归测 (录制雪崩根因③ — refresh 覆盖未落盘修改, 全局化后池级同款语义):
// refreshSubgraphStore 用后端磁盘快照同步全局池时, 不能整体覆盖 —— 内存里正在编辑、还没落盘的
// 子图修改会被磁盘旧版盖掉、引用丢失。merge 语义: 后端列表为基准 (反映删除), 两边都有则保留内存版。
import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useContainerEditorStore } from './containerEditor'

function sg(id: string, nodes: any[] = []) {
  return {
    id,
    rev: 1,
    label: id,
    outputPins: [],
    entry: { nodeID: '' },
    graph: { id: 'g-' + id, schemaVersion: 1, nodes, edges: [] },
    createdAt: '',
  }
}

describe('useContainerEditorStore.mergePool', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('两边都有的子图保留内存版 (不被后端快照覆盖编辑)', () => {
    const store = useContainerEditorStore()
    const memNode = { id: 'n', kind: 'Subgraph', x: 0, y: 0, config: { SubgraphID: 'sg-child' } }
    store.setPool([sg('sg-a', [memNode])] as any)
    // 后端快照里 sg-a 还是空的 (内存的新引用节点还没落盘)
    store.mergePool([sg('sg-a', [])] as any)
    const a = store.subgraphById('sg-a')
    expect(a?.graph.nodes).toHaveLength(1)
  })

  it('补进后端新增的子图', () => {
    const store = useContainerEditorStore()
    store.setPool([sg('sg-a')] as any)
    store.mergePool([sg('sg-a'), sg('sg-new')] as any)
    expect(store.subgraphList.map((s) => s.id).sort()).toEqual(['sg-a', 'sg-new'])
  })

  it('移除后端已删的子图', () => {
    const store = useContainerEditorStore()
    store.setPool([sg('sg-a'), sg('sg-gone')] as any)
    store.mergePool([sg('sg-a')] as any)
    expect(store.subgraphList.map((s) => s.id)).toEqual(['sg-a'])
  })

  it('touch 归属按容器隔离, clearTouched 只清指定项', () => {
    const store = useContainerEditorStore()
    store.touchSubgraph('c1', 'sg-a')
    store.touchSubgraph('c1', 'sg-b')
    store.touchSubgraph('c2', 'sg-a')
    expect(store.touchedFor('c1').sort()).toEqual(['sg-a', 'sg-b'])
    expect(store.touchedFor('c2')).toEqual(['sg-a'])
    store.clearTouched('c1', ['sg-a'])
    expect(store.touchedFor('c1')).toEqual(['sg-b'])
    expect(store.touchedFor('c2')).toEqual(['sg-a'])
  })
})
