import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type TemplateMeta } from '@/lib/backend'

export const useTemplatesStore = defineStore('templates', () => {
  const containerId = ref<string>('')
  const map = ref<Record<string, TemplateMeta>>({})

  function setContainer(cid: string) {
    containerId.value = cid
    map.value = {}
  }

  async function reload() {
    if (!containerId.value) return
    const r = await backend.templates.list(containerId.value)
    if (r !== undefined) map.value = r as Record<string, TemplateMeta>
  }

  async function save(
    key: string,
    dataURL: string,
    name: string,
    description: string,
    recordedResolution: [number, number],
    region: [number, number, number, number],
  ): Promise<boolean> {
    if (!containerId.value) return false
    const r = await backend.templates.save(
      containerId.value,
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
    if (!containerId.value) return false
    const r = await backend.templates.delete_(containerId.value, key)
    if (r === undefined) return false
    await reload()
    return true
  }

  async function readPng(key: string): Promise<string | null> {
    if (!containerId.value) return null
    const r = await backend.templates.readPngDataURL(containerId.value, key)
    return typeof r === 'string' ? r : null
  }

  async function capture(): Promise<string | null> {
    if (!containerId.value) return null
    const r = await backend.templates.capture(containerId.value)
    return r === undefined ? null : (r as string)
  }

  return { containerId, map, setContainer, reload, save, remove, capture, readPng }
})
