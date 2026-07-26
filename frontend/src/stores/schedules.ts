import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type Schedule } from '@/lib/backend'

export const useSchedulesStore = defineStore('schedules', () => {
  const list = ref<Schedule[]>([])

  async function reload() {
    list.value = await backend.schedules.list()
  }

  async function createDraft(name: string): Promise<Schedule> {
    return backend.schedules.create(name)
  }

  async function save(sc: Schedule): Promise<void> {
    await backend.schedules.save(sc)
    await reload()
  }

  async function update(id: string, patch: Partial<Schedule>): Promise<void> {
    await backend.schedules.update(id, JSON.stringify(patch))
    await reload()
  }

  async function remove(id: string): Promise<void> {
    await backend.schedules.delete_(id)
    await reload()
  }

  return { list, reload, createDraft, save, update, remove }
})
