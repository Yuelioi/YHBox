// useWindowControls 自定义 Frameless 窗口的标题栏控件 (最小化 / 还原 / 关闭) 通用 composable.
//
// 适用: AppTitleBar (主窗口) + ContainerEditorView (独立编辑器窗口) + 任何 wails3 Frameless 窗口.
// 调用方: 自动启动 IsMaximised 轮询 (500ms), 自动清理 timer; close 行为可被 view 包装一层拦截
// (dirty 时弹保存确认等), 默认 closeImmediate 直接 Window.Close.
import { onMounted, onUnmounted, ref } from 'vue'
import { Window } from '@wailsio/runtime'

export function useWindowControls() {
  const isMaximised = ref(false)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  async function pollMax() {
    try {
      isMaximised.value = await Window.IsMaximised()
    } catch {
      /* ignore — webview2 偶尔启动期 race 报错, 下次 poll 自愈 */
    }
  }

  function onMinimise() {
    Window.Minimise()
  }

  function onToggleMaximise() {
    Window.ToggleMaximise()
    // wails3 ToggleMaximise 异步; 50ms 后再 poll 拿确切状态 (避免 icon 跳一帧)
    setTimeout(pollMax, 50)
  }

  function closeImmediate() {
    Window.Close()
  }

  onMounted(() => {
    void pollMax()
    pollTimer = setInterval(pollMax, 500)
  })
  onUnmounted(() => {
    if (pollTimer) clearInterval(pollTimer)
  })

  return { isMaximised, onMinimise, onToggleMaximise, closeImmediate }
}
