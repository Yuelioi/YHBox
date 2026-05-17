import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend, type Container } from '@/lib/backend'

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

  async function remove(id: string): Promise<boolean> {
    const r = await backend.containers.delete_(id)
    if (r === undefined) return false
    await reload()
    return true
  }

  async function deleteMany(ids: string[]): Promise<boolean> {
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
    editingContainerIDs, isEditing,
  }
})
