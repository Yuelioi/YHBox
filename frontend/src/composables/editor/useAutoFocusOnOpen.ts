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
  // 抢到焦点后全选内容 (命名框「默认值一字可替」用)。
  selectAll?: boolean
}

export function useAutoFocusOnOpen(
  modelOpen: Ref<boolean>,
  inputRef: Ref<{ inputRef?: HTMLInputElement | null } | null>,
  opts: AutoFocusOpts = {},
) {
  watch(modelOpen, async (v) => {
    if (!v) return
    opts.onOpen?.()
    await nextTick()
    // 抢搜索框焦点: focus 一次, 若被抢则连帧重试直到它真成 activeElement.
    // NuxtUI UInput 把原生 <input> 经 defineExpose({ inputRef }) 暴露成 .inputRef
    // (不是 .input — 早先按 .input 取永远 undefined, 焦点根本没设过). 无竞争场景一次成功;
    // 万一有谁晚抢 (最多 ~12 帧 ≈ 200ms) 也抢得回, 抢到立即停、无副作用.
    let tries = 0
    const grab = () => {
      const el = inputRef.value?.inputRef
      el?.focus?.()
      if (el && document.activeElement === el) {
        if (opts.selectAll) el.select()
        return // 抢到了
      }
      if (tries++ < 12) {
        requestAnimationFrame(grab)
      } else if (import.meta.env.DEV) {
        // 12 帧还没抢到 → 打出真相, 别再靠猜
        console.warn('[autofocus] 搜索框没拿到焦点; 当前焦点=', document.activeElement, '; input ref=', el)
      }
    }
    grab()
  })
}
