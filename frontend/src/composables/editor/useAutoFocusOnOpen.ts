// modal 打开时自动 focus search input. 可选 onOpen 回调 (常用于 reset query/activeIdx).
// 5 modal 重复 (CommandPalette / InlineContextMenu / LibraryExplorer / NodeExplorer
// / NodeSearch) 收口到这里.
//
// Usage:
//   const searchInputRef = ref<any>(null)
//   useAutoFocusOnOpen(modelOpen, searchInputRef, {
//     onOpen: () => { query.value = ''; activeIdx.value = 0 },
//   })

import { nextTick, watch, type Ref } from 'vue'

interface AutoFocusOpts {
  onOpen?: () => void
}

export function useAutoFocusOnOpen(
  modelOpen: Ref<boolean>,
  inputRef: Ref<{ input?: HTMLInputElement | null } | null>,
  opts: AutoFocusOpts = {},
) {
  watch(modelOpen, async (v) => {
    if (!v) return
    opts.onOpen?.()
    await nextTick()
    inputRef.value?.input?.focus?.()
  })
}
