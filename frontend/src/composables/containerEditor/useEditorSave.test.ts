// onSave 保存顺序 + GC 摘除回归测 (录制雪崩根因修):
//   ① 不再在自动保存里跑孤儿 GC — onSave 永不 deleteSubgraph.
//   ② 先落盘所有子图, 再保存校验主图 — 主图校验依赖子图存在, 顺序反了会 MISSING_SUBGRAPH 误炸.
//      子图落盘失败则不碰主图、保留 dirty 让用户重试.
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
    updateSilent: vi.fn(async () => { calls.push('main') }),
    updateSubgraphSilent: vi.fn(async (_cid: string, sgID: string) => { calls.push('sg:' + sgID) }),
    deleteSubgraph: vi.fn(async (_cid: string, sgID: string) => { calls.push('del:' + sgID) }),
  }
})
vi.mock('@/lib/backend', () => ({
  backend: {
    containers: {
      updateSilent: h.updateSilent,
      updateSubgraphSilent: h.updateSubgraphSilent,
      deleteSubgraph: h.deleteSubgraph,
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
    label: id,
    outputPins: [],
    entry: { nodeID: '' },
    graph: {
      id: 'g-' + id,
      version: 1,
      nodes: refSubID ? [{ id: 'n', kind: 'Subgraph', x: 0, y: 0, config: { SubgraphID: refSubID } }] : [],
      edges: [],
    },
  }
}

function setup(sgs: ReturnType<typeof sg>[], mainSubRefs: string[] = []) {
  setActivePinia(createPinia())
  const store = useContainerEditorStore()
  store.setActiveContainer('c1', sgs as any)
  const draft = ref<Container | null>({
    schemaVersion: 1,
    id: 'c1',
    name: 't',
    graph: {
      id: 'g-main',
      version: 1,
      nodes: mainSubRefs.map((s, i) => ({ id: 'm' + i, kind: 'Subgraph', x: 0, y: 0, config: { SubgraphID: s } })),
      edges: [],
    },
    createdAt: '',
    updatedAt: '',
  } as unknown as Container)
  const dirty = ref(true)
  const toast = { add: vi.fn() }
  const { onSave } = useEditorSave({ draft, dirty, toast })
  return { onSave, dirty, toast, store }
}

describe('useEditorSave.onSave', () => {
  beforeEach(() => {
    h.calls.length = 0
    vi.clearAllMocks()
  })

  it('先落盘所有子图, 再保存主图', async () => {
    const { onSave } = setup([sg('sg-a'), sg('sg-b')], ['sg-a'])
    const ok = await onSave()
    expect(ok).toBe(true)
    const mainIdx = h.calls.indexOf('main')
    expect(mainIdx).toBeGreaterThan(-1)
    expect(h.calls.indexOf('sg:sg-a')).toBeLessThan(mainIdx)
    expect(h.calls.indexOf('sg:sg-b')).toBeLessThan(mainIdx)
  })

  it('子图落盘失败 → 不保存主图, 保留 dirty, 返 false', async () => {
    h.updateSubgraphSilent.mockImplementationOnce(async () => { throw new Error('boom') })
    const { onSave, dirty } = setup([sg('sg-a')], ['sg-a'])
    const ok = await onSave()
    expect(ok).toBe(false)
    expect(h.updateSilent).not.toHaveBeenCalled()
    expect(dirty.value).toBe(true)
  })

  it('不删任何子图 (自动保存不跑孤儿 GC)', async () => {
    // sg-orphan 无任何引用 — 旧逻辑会 GC 删掉, 新逻辑必须留着.
    const { onSave } = setup([sg('sg-orphan')], [])
    await onSave()
    expect(h.deleteSubgraph).not.toHaveBeenCalled()
  })

  it('全部成功 → dirty=false, 返 true', async () => {
    const { onSave, dirty } = setup([sg('sg-a')], ['sg-a'])
    const ok = await onSave()
    expect(ok).toBe(true)
    expect(dirty.value).toBe(false)
  })
})
