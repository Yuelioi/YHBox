import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type Container } from '@/lib/backend'
import { useRecordingStore } from '@/stores/recording'

export const useContainersStore = defineStore('containers', () => {
  const list = ref<Container[]>([])

  // Plan B Task D.10：Draft 所有权 — 后端 wire_container_editor.go 在子窗口打开/关闭时 emit 事件
  // 这里订阅，给 ContainersView 卡片 disable + 锁图标 UI 用
  const editingContainerIDs = ref<Set<string>>(new Set())

  async function reload() {
    const r = await backend.containers.list()
    if (r !== undefined) list.value = r as unknown as Container[]
  }

  Events.On('container:editing-acquired', (data: any) => {
    const id = data?.data?.id ?? data?.id
    if (id) editingContainerIDs.value = new Set([...editingContainerIDs.value, id])
  })
  Events.On('container:editing-released', (data: any) => {
    const id = data?.data?.id ?? data?.id
    if (id) {
      const next = new Set(editingContainerIDs.value)
      next.delete(id)
      editingContainerIDs.value = next
    }
  })

  function isEditing(id: string): boolean {
    return editingContainerIDs.value.has(id)
  }

  // 容器 CRUD 后端 emit container:changed → 自动 reload list（跨窗口同步）
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
    return rec.isRecording && rec.activeTargetContainerID === id
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
    const r = await backend.containers.deleteMany(ids)
    // r 在批删失败时是 error string；undefined 即成功
    await reload()
    return r === undefined
  }

  async function run(id: string) {
    await backend.containers.run(id)
  }

  async function stopAll() {
    await backend.containers.stopAll()
  }

  return {
    list, reload, create, update, remove, deleteMany, run, stopAll,
    editingContainerIDs, isEditing, isRecordingLocked,
  }
})
