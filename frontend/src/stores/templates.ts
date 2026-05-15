import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type TemplateMeta } from '@/lib/backend'

export const useTemplatesStore = defineStore('templates', () => {
  // key → meta
  const map = ref<Record<string, TemplateMeta>>({})

  async function reload() {
    const r = await backend.templates.list()
    if (r !== undefined) map.value = r as unknown as Record<string, TemplateMeta>
  }

  async function save(
    key: string,
    dataURL: string,
    name: string,
    description: string,
    recordedResolution: [number, number],
    region: [number, number, number, number],
  ): Promise<boolean> {
    const r = await backend.templates.save(
      key,
      dataURL,
      name,
      description,
      recordedResolution,
      region,
    )
    if (r === undefined) return false
    await reload()
    return true
  }

  async function remove(key: string): Promise<boolean> {
    const r = await backend.templates.delete_(key)
    if (r === undefined) return false
    await reload()
    return true
  }

  async function updateMeta(key: string, name: string, description: string): Promise<boolean> {
    const r = await backend.templates.updateMeta(key, name, description)
    if (r === undefined) return false
    await reload()
    return true
  }

  async function capture(): Promise<string | null> {
    const r = await backend.templates.capture()
    return r === undefined ? null : (r as string)
  }

  return { map, reload, save, remove, capture, updateMeta }
})
