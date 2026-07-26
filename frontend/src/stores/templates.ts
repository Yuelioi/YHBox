// stores/templates.ts — 全局资产 store (template kind).
// Phase 7: 从"容器 key map"换成"全局 guid map", 全部走 asset RPC.
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type AssetSummary } from '@/lib/backend'

export const useTemplatesStore = defineStore('templates', () => {
  // map: guid → AssetSummary (只含 template kind)
  const map = ref<Record<string, AssetSummary>>({})

  async function reload() {
    const summaries = await backend.assets.list()
    const next: Record<string, AssetSummary> = {}
    for (const s of summaries as AssetSummary[]) {
      if (s.kind === 'template') {
        next[s.guid] = s
      }
    }
    map.value = next
  }

  // save: 截图存为新模板资产，失败抛 typed RPCError。
  async function save(
    dataURL: string,
    name: string,
    category: string,
    tags: string[],
    recordedResolution: [number, number],
    region: [number, number, number, number],
  ): Promise<string> {
    const guid = await backend.assets.saveTemplateCapture(
      dataURL,
      name,
      category,
      tags,
      recordedResolution,
      region,
    )
    await reload()
    return guid as string
  }

  async function remove(guid: string): Promise<void> {
    await backend.assets.delete_(guid)
    await reload()
  }

  // updateMeta: 改名称/描述/分类/标签 (记录级, 不动变体/blob). 改完 reload 全局列表.
  async function updateMeta(
    guid: string,
    name: string,
    description: string,
    category: string,
    tags: string[],
  ): Promise<void> {
    await backend.assets.updateMeta(guid, name, description, category, tags)
    await reload()
  }

  return {
    map,
    reload,
    save,
    remove,
    updateMeta,
  }
})
