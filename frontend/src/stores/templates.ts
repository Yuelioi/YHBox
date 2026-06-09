// stores/templates.ts — 全局资产 store (template kind).
// Phase 7: 从"容器 key map"换成"全局 guid map", 全部走 asset RPC.
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type AssetSummary } from '@/lib/backend'

export const useTemplatesStore = defineStore('templates', () => {
  // containerId: 仍由 ContainerEditorView 调 setContainer 注入, 供 capture / ScreenPicker 定位目标窗口.
  // asset 列表本身是全局的, 与 containerId 无关.
  const containerId = ref<string>('')

  // map: guid → AssetSummary (只含 template kind)
  const map = ref<Record<string, AssetSummary>>({})

  function setContainer(cid: string) {
    containerId.value = cid
    // 切容器不清资产列表 — 资产是全局的, 不跟容器走.
  }

  async function reload() {
    const summaries = await backend.assets.list()
    if (summaries === undefined) return
    const next: Record<string, AssetSummary> = {}
    for (const s of summaries as AssetSummary[]) {
      if (s.kind === 'template') {
        next[s.guid] = s
      }
    }
    map.value = next
  }

  // save: 截图存为新模板资产, 返回新分配的 guid (成功) 或 null.
  async function save(
    dataURL: string,
    name: string,
    recordedResolution: [number, number],
    region: [number, number, number, number],
  ): Promise<string | null> {
    const guid = await backend.assets.saveTemplateCapture(dataURL, name, recordedResolution, region)
    if (guid === undefined || guid === null) return null
    await reload()
    return guid as string
  }

  async function remove(guid: string): Promise<boolean> {
    const r = await backend.assets.delete_(guid)
    if (r === undefined) return false
    await reload()
    return true
  }

  // readBlobDataURL: 按 AssetSummary.firstBlobSha 拉缩略图.
  async function readBlobDataURL(sha: string): Promise<string | null> {
    if (!sha) return null
    const r = await backend.assets.readBlobDataURL(sha)
    return typeof r === 'string' ? r : null
  }

  // capture: 截当前容器目标窗口帧, 返 data URL (用于 TemplateCapture/ScreenPicker 底图).
  async function capture(): Promise<string | null> {
    if (!containerId.value) return null
    const r = await backend.assets.capture(containerId.value)
    return r === undefined ? null : (r as string)
  }

  return { containerId, map, setContainer, reload, save, remove, capture, readBlobDataURL }
})
