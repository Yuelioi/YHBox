import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type Container } from '@/lib/backend'

export const useContainersStore = defineStore('containers', () => {
  const list = ref<Container[]>([])

  async function reload() {
    const r = await backend.containers.list()
    if (r !== undefined) list.value = r as unknown as Container[]
  }

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

  async function run(id: string) {
    await backend.containers.run(id)
  }

  async function stopAll() {
    await backend.containers.stopAll()
  }

  return { list, reload, create, update, remove, run, stopAll }
})
