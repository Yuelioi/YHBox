import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type Schedule } from '@/lib/backend'

export const useSchedulesStore = defineStore('schedules', () => {
  const list = ref<Schedule[]>([])

  async function reload() {
    const r = await backend.schedules.list()
    if (r !== undefined) list.value = r as unknown as Schedule[]
  }

  async function createDraft(name: string): Promise<Schedule | null> {
    const r = await backend.schedules.create(name)
    return r === undefined ? null : (r as unknown as Schedule)
  }

  async function save(sc: Schedule): Promise<boolean> {
    const r = await backend.schedules.save(sc)
    if (r === undefined) return false
    await reload()
    return true
  }

  async function update(id: string, patch: Partial<Schedule>): Promise<boolean> {
    const r = await backend.schedules.update(id, JSON.stringify(patch))
    if (r === undefined) return false
    await reload()
    return true
  }

  async function remove(id: string): Promise<boolean> {
    const r = await backend.schedules.delete_(id)
    if (r === undefined) return false
    await reload()
    return true
  }

  return { list, reload, createDraft, save, update, remove }
})
