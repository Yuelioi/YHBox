// useConfirm 全局确认对话框 API（Promise 返回）。
// 配合 App.vue 全局挂载的 ConfirmDialog 实例使用。
// 同时只能开一个对话框（单例 state）。
import { reactive } from 'vue'

export interface ConfirmOpts {
  title: string
  description?: string
  confirmText?: string
  cancelText?: string
  color?: 'primary' | 'error' | 'warning' | 'neutral'
  alternateText?: string
  alternateValue?: string
  alternateColor?: 'primary' | 'error' | 'warning' | 'neutral'
  /** 若设了 inputDefault，对话框显示 UInput，confirm 返回输入字符串；否则返 boolean */
  inputDefault?: string
  inputLabel?: string
  inputPlaceholder?: string
}

const state = reactive<{ open: boolean; opts: ConfirmOpts | null }>({
  open: false,
  opts: null,
})
let activeResolver: ((v: boolean | string) => void) | null = null

export function useConfirm() {
  function confirm(opts: ConfirmOpts): Promise<boolean | string> {
    return new Promise((resolve) => {
      activeResolver = resolve
      state.opts = opts
      state.open = true
    })
  }
  function resolveActive(v: boolean | string) {
    if (activeResolver) {
      activeResolver(v)
      activeResolver = null
    }
    state.open = false
    state.opts = null
  }
  return { confirm, state, resolveActive }
}
