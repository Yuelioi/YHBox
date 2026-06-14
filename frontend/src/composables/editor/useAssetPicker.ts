// frontend/src/composables/editor/useAssetPicker.ts
import { ref } from 'vue'

export interface AssetPickRequest {
  /** 发起选择的节点 literal pin 名 (如 'Templates') */
  pin: string
  /** 当前已指派的模板 GUID 列表 */
  selected: string[]
}

// 模块级单例: 节点字段与停靠区资产 tab 共享同一份 request (仿 useSidebarPrefs).
const request = ref<AssetPickRequest | null>(null)

/** 节点字段点"选模板" → 请求把停靠区资产 tab 切到 pick 模式. */
function requestTemplatePick(pin: string, selected: string[]) {
  request.value = { pin, selected: [...selected] }
}

/** 停靠区里勾选/取消 → 更新当前选择 (字段会镜像回写到 config.literal). */
function updateSelection(selected: string[]) {
  if (!request.value) return
  request.value = { ...request.value, selected: [...selected] }
}

/** 取消 pick 上下文 (关停靠区 / 切节点). */
function cancel() {
  request.value = null
}

export function useAssetPicker() {
  return { request, requestTemplatePick, updateSelection, cancel }
}
