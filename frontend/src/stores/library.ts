// 子图库 store (2026-06-12 全局化): 库 = 全局子图池的视图, 不再有"导入/导出包"。
// 编辑器内的池由 containerEditor store 持有; 本 store 服务子图库 modal / 详情面板
// 等浏览消费方 — 自己拉 backend.subgraphs.list() 并订阅 subgraph:changed。
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type Subgraph, type SubgraphReferrer } from '@/lib/backend'

export const useLibraryStore = defineStore('library', () => {
  const pool = ref<Subgraph[]>([])
  const loading = ref(false)

  // 库浏览 = 具名子图 (匿名是 CollapsedNode 后备体, 实现细节不展示)。
  const subgraphs = computed<Subgraph[]>(() => pool.value.filter((s) => !s.isAnonymous))

  async function reload() {
    loading.value = true
    try {
      pool.value = ((await backend.subgraphs.list()) ?? []) as Subgraph[]
    } catch (e) {
      console.warn('subgraph pool reload failed', e)
    } finally {
      loading.value = false
    }
  }

  function byId(id: string): Subgraph | undefined {
    return pool.value.find((s) => s.id === id)
  }

  // referrers — 删除前警告 + 「被 N 个容器使用」(去重非空 containerID)。
  async function referrersOf(id: string): Promise<SubgraphReferrer[]> {
    return ((await backend.subgraphs.referrers(id)) ?? []) as SubgraphReferrer[]
  }
  function containerUseCount(refs: SubgraphReferrer[]): number {
    return new Set(refs.map((r) => r.containerID).filter(Boolean)).size
  }

  async function deleteSubgraph(id: string): Promise<boolean> {
    const sg = byId(id)
    try {
      await backend.subgraphs.delete_(id, sg?.rev ?? 0)
      await reload()
      return true
    } catch (e) {
      console.warn('subgraph delete failed', e)
      return false
    }
  }

  // 复制为新子图 (fork, ≈Blender Make Local)。返回副本。
  async function duplicateSubgraph(id: string): Promise<Subgraph | undefined> {
    const dup = await backend.subgraphs.duplicate(id)
    if (dup) await reload()
    return dup ?? undefined
  }

  // missingGlobalsFor — 插入引用前检缺变量: 该子图 + 传递闭包内全部 callee 的
  // requiredGlobals 名字, 减去 declared。Type/Default 由 caller 按目标容器现算 (这里只有名字)。
  function missingGlobalsFor(sgID: string, declared: Set<string>): string[] {
    const missing: string[] = []
    const seen = new Set<string>()
    const visited = new Set<string>()
    const queue = [sgID]
    while (queue.length > 0) {
      const id = queue.shift()!
      if (visited.has(id)) continue
      visited.add(id)
      const sg = pool.value.find((s) => s.id === id)
      if (!sg) continue
      for (const name of sg.requiredGlobals ?? []) {
        if (!declared.has(name) && !seen.has(name)) {
          seen.add(name)
          missing.push(name)
        }
      }
      for (const n of sg.graph?.nodes ?? []) {
        const ref = (n as { kind?: string; config?: Record<string, unknown> })
        const calleeID = ref.config?.SubgraphID as string | undefined
        if ((ref.kind === 'Subgraph' || ref.kind === 'CollapsedNode') && calleeID) queue.push(calleeID)
      }
    }
    return missing
  }

  Events.On('subgraph:changed', () => {
    void reload()
  })

  return {
    pool,
    subgraphs,
    loading,
    reload,
    byId,
    referrersOf,
    containerUseCount,
    deleteSubgraph,
    duplicateSubgraph,
    missingGlobalsFor,
  }
})
