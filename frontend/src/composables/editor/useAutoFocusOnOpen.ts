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
    // 死磕到搜索框真拿到焦点: 连续几帧反复 focus, 直到它真成 activeElement.
    // 起因是 pin 拖出开菜单走 vue-flow 拖线手势, 松手那下画布会把焦点抢回去, 比单次
    // focus 晚 → 抢不过. 这里每帧再抢一次 (最多 ~12 帧 ≈ 200ms), 谁晚抢都抢得回来.
    // 已聚焦时立即停, 对右键/双击等无竞争场景是一次成功、无副作用.
    let tries = 0
    const grab = () => {
      const el = inputRef.value?.input
      el?.focus?.()
      if (el && document.activeElement === el) return // 抢到了
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
