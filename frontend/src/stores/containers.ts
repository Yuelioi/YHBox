import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type Container } from '@/lib/backend'
import { useRecordingStore } from '@/stores/recording'

export const useContainersStore = defineStore('containers', () => {
  const list = ref<Container[]>([])

  async function reload() {
    const r = await backend.containers.list()
    if (r !== undefined) list.value = r as unknown as Container[]
  }

  // 容器 CRUD 后端 emit container:changed → 自动 reload list（MCP / 外部改盘同步）
  Events.On('container:changed', () => {
    void reload()
  })

  async function create(name: string): Promise<Container | null> {
    const r = await backend.containers.create(name)
    if (r === undefined) return null
    await reload()
    return r as unknown as Container
  }

  async function update(id: string, patch: Partial<Container>): Promise<boolean> {
    const r = await backend.containers.update(id, JSON.stringify(patch))
    if (r === undefined) return false
    await reload()
    return true
  }

  // A2: 录制目标容器在录制态不可删 — 否则停录 SaveSubgraph 撞 "container not found".
  // 单一来源是 recordStore.activeTargetContainerID (本窗口). 这是 chokepoint backstop;
  // UI (ContainersTab) 另做一次带 toast 的前置拦截.
  function isRecordingLocked(id: string): boolean {
    const rec = useRecordingStore()
    // 暂停态 (isPaused) 仍是进行中的录制会话 — 同样锁删, 否则暂停期删容器 → 恢复/停录 SaveSubgraph 撞 not found.
    return (rec.isRecording || rec.isPaused) && rec.activeTargetContainerID === id
  }

  async function remove(id: string): Promise<boolean> {
    if (isRecordingLocked(id)) return false
    const r = await backend.containers.delete_(id)
    if (r === undefined) return false
    await reload()
    return true
  }

  async function deleteMany(ids: string[]): Promise<boolean> {
    if (ids.some((id) => isRecordingLocked(id))) return false
    try {
      await backend.containers.deleteMany(ids)
      await reload()
      return true
    } catch {
      await reload()
      return false
    }
  }

  async function exportPackage(id: string, destPath: string): Promise<boolean> {
    const r = await backend.containers.exportPackage(id, destPath)
    return r === true
  }

  async function run(id: string) {
    await backend.containers.run(id)
  }

  async function stopAll() {
    await backend.containers.stopAll()
  }

  return {
    list,
    reload,
    create,
    update,
    remove,
    deleteMany,
    exportPackage,
    run,
    stopAll,
    isRecordingLocked,
  }
})
