// onSave 保存顺序 + 乐观锁回归测 (全局化版):
//   ① 只保存本容器 touch 过的子图 (归属隔离), onSave 永不 deleteSubgraph.
//   ② 先落盘子图, 再保存校验主图 — 主图校验依赖子图存在, 顺序反了会 MISSING_SUBGRAPH 误炸.
//      子图落盘失败则不碰主图、保留 dirty 让用户重试.
//   ③ 乐观锁被拒 (盘上 rev 更新) → staleSubgraphs 暴露给 view 弹"重载?", 不碰主图.
import { describe, it, expect, beforeEach, vi } from 'vitest'

// useI18n mock 成 identity t (非组件 setup 调 useI18n 否则抛错).
vi.mock('vue-i18n', async (importActual) => ({
  ...(await importActual<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (k: string) => k }),
}))

// backend mock — 用 hoisted 共享一个调用顺序记录数组.
const h = vi.hoisted(() => {
  const calls: string[] = []
  return {
    calls,
    updateSilent: vi.fn(async () => {
      calls.push('main')
    }),
    sgUpdateSilent: vi.fn(async (sgID: string) => {
      calls.push('sg:' + sgID)
    }),
    sgGet: vi.fn(async (sgID: string) => ({
      id: sgID,
      rev: 2,
      label: sgID,
      outputPins: [],
      entry: { nodeID: '' },
      graph: { id: 'g', schemaVersion: 1, nodes: [], edges: [] },
      createdAt: '',
    })),
    sgDelete: vi.fn(async (sgID: string) => {
      calls.push('del:' + sgID)
    }),
  }
})
vi.mock('@/lib/backend', () => ({
  backend: {
    containers: {
      updateSilent: h.updateSilent,
    },
    subgraphs: {
      updateSilent: h.sgUpdateSilent,
      get: h.sgGet,
      delete_: h.sgDelete,
    },
  },
}))
vi.mock('@/lib/invoke', () => ({ errorMessage: (e: any) => String(e?.message ?? e) }))

import { ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useEditorSave } from './useEditorSave'
import type { Container } from '@/lib/backend'

function sg(id: string, refSubID?: string) {
  return {
    id,
    rev: 1,
    label: id,
    outputPins: [],
    entry: { nodeID: '' },
    graph: {
      id: 'g-' + id,
      schemaVersion: 1,
      nodes: refSubID
        ? [{ id: 'n', kind: 'Subgraph', x: 0, y: 0, config: { SubgraphID: refSubID } }]
        : [],
      edges: [],
    },
    createdAt: '',
  }
}

function setup(sgs: ReturnType<typeof sg>[], mainSubRefs: string[] = [], touched: string[] = []) {
  setActivePinia(createPinia())
  const store = useContainerEditorStore()
  store.setActiveContainer('c1')
  store.setPool(sgs as any)
  for (const id of touched) store.touchSubgraph('c1', id)
  const draft = ref<Container | null>({
    schemaVersion: 1,
    id: 'c1',
    name: 't',
    graph: {
      id: 'g-main',
      schemaVersion: 1,
      nodes: mainSubRefs.map((s, i) => ({
        id: 'm' + i,
        kind: 'Subgraph',
        x: 0,
        y: 0,
        config: { SubgraphID: s },
      })),
      edges: [],
    },
    createdAt: '',
    updatedAt: '',
  } as unknown as Container)
  const dirty = ref(true)
  const toast = { add: vi.fn() }
  const save = useEditorSave({ containerID: 'c1', draft, dirty, toast })
  return { ...save, dirty, toast, store }
}

describe('useEditorSave.onSave', () => {
  beforeEach(() => {
    h.calls.length = 0
    vi.clearAllMocks()
  })

  it('先落盘 touch 过的子图, 再保存主图', async () => {
    const { onSave } = setup([sg('sg-a'), sg('sg-b')], ['sg-a'], ['sg-a', 'sg-b'])
    const ok = await onSave()
    expect(ok).toBe(true)
    const mainIdx = h.calls.indexOf('main')
    expect(mainIdx).toBeGreaterThan(-1)
    expect(h.calls.indexOf('sg:sg-a')).toBeLessThan(mainIdx)
    expect(h.calls.indexOf('sg:sg-b')).toBeLessThan(mainIdx)
  })

  it('没 touch 的子图不保存 (归属隔离, 不跨容器代保)', async () => {
    const { onSave } = setup([sg('sg-a'), sg('sg-other')], ['sg-a'], ['sg-a'])
    await onSave()
    expect(h.calls).toContain('sg:sg-a')
    expect(h.calls).not.toContain('sg:sg-other')
  })

  it('子图落盘失败 → 不保存主图, 保留 dirty, 返 false', async () => {
    h.sgUpdateSilent.mockImplementationOnce(async () => {
      throw new Error('boom')
    })
    const { onSave, dirty } = setup([sg('sg-a')], ['sg-a'], ['sg-a'])
    const ok = await onSave()
    expect(ok).toBe(false)
    expect(h.updateSilent).not.toHaveBeenCalled()
    expect(dirty.value).toBe(true)
  })

  it('乐观锁被拒 → staleSubgraphs 暴露, 不碰主图, 不弹错误 toast', async () => {
    h.sgUpdateSilent.mockImplementationOnce(async () => {
      throw new Error('subgraph rev stale: 盘上已有更新 (盘上 rev=3, 基准 rev=1)')
    })
    const { onSave, staleSubgraphs, toast } = setup([sg('sg-a')], ['sg-a'], ['sg-a'])
    const ok = await onSave()
    expect(ok).toBe(false)
    expect(staleSubgraphs.value).toEqual(['sg-a'])
    expect(h.updateSilent).not.toHaveBeenCalled()
    expect(toast.add).not.toHaveBeenCalled()
  })

  it('不删任何子图 (自动保存不跑孤儿 GC)', async () => {
    const { onSave } = setup([sg('sg-orphan')], [], ['sg-orphan'])
    await onSave()
    expect(h.sgDelete).not.toHaveBeenCalled()
  })

  it('全部成功 → dirty=false, 返 true, touched 清空', async () => {
    const { onSave, dirty, store } = setup([sg('sg-a')], ['sg-a'], ['sg-a'])
    const ok = await onSave()
    expect(ok).toBe(true)
    expect(dirty.value).toBe(false)
    expect(store.touchedFor('c1')).toEqual([])
  })
})
