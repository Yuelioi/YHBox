// useCalibrationSession — 鼠标 DPI 校准的状态机 + F8 全局热键接线，抽出来给独立校准 HUD 窗复用。
//
// 流程 (用户按 F8 推进, F8 走后端自治 LL hook, 切到游戏也收得到):
//   waiting → (F8) → countingDown(3s) → accumulating(原地转 360°) → (F8) → done
//   done 时再按 F8 = 重测 (回到 countingDown)。
// 累积值 absDx 由后端 raw-input session 攒, 这里 80ms poll 读。
//
// open() 装钩 + 订阅 toggle; teardown() 停钩 + 停 session + 清定时器 (窗口关/卸载都要调)。
import { computed, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'

export type CalibrationStage = 'waiting' | 'countingDown' | 'accumulating' | 'done'

export function useCalibrationSession() {
  const stage = ref<CalibrationStage>('waiting')
  const countdown = ref(3)
  const hotkeyWarn = ref('')
  const status = ref<{ active: boolean; absDx: number; absDy: number }>({
    active: false,
    absDx: 0,
    absDy: 0,
  })
  const liveAbsDx = computed(() => status.value.absDx)
  const liveAbsDy = computed(() => status.value.absDy)

  let pollTimer: ReturnType<typeof setInterval> | null = null
  let countdownTimer: ReturnType<typeof setInterval> | null = null
  let unsubToggle: (() => void) | null = null

  function clearCountdown() {
    if (countdownTimer) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
  }
  function clearPoll() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  function resetSession() {
    stage.value = 'waiting'
    countdown.value = 3
    status.value = { active: false, absDx: 0, absDy: 0 }
    clearCountdown()
    void backend.calibration.stop() // 已开着的后端 session 先停
    clearPoll()
  }

  async function open() {
    hotkeyWarn.value = ''
    resetSession()
    if (!unsubToggle) {
      const off = (await Events.On('calibration:toggle', onToggleHotkey)) as unknown as () => void
      unsubToggle = typeof off === 'function' ? off : null
    }
    // 装 F8 全局 LL hook (切到游戏也收得到, 不走 OS RegisterHotKey)。失败 (杀软拦截) 提示。
    try {
      await backend.calibration.startHotkeyWatch()
    } catch {
      hotkeyWarn.value = 'service_failed'
    }
  }

  function onToggleHotkey() {
    switch (stage.value) {
      case 'waiting':
        beginCountdown()
        break
      case 'countingDown':
        finishCountdown() // 提前按 = 立即进累计
        break
      case 'accumulating':
        void stopAccumulating()
        break
      case 'done':
        resetSession()
        beginCountdown() // 连按 = 重测
        break
    }
  }

  function beginCountdown() {
    stage.value = 'countingDown'
    countdown.value = 3
    countdownTimer = setInterval(() => {
      countdown.value -= 1
      if (countdown.value <= 0) void finishCountdown()
    }, 1000)
  }

  async function finishCountdown() {
    clearCountdown()
    stage.value = 'accumulating'
    const ok = await backend.calibration.start()
    if (ok === undefined) {
      hotkeyWarn.value = 'service_failed'
      stage.value = 'waiting'
      return
    }
    pollTimer = setInterval(pollStatus, 80)
  }

  async function pollStatus() {
    const s = await backend.calibration.status()
    if (s) status.value = s as any
  }

  async function stopAccumulating() {
    clearPoll()
    const s = await backend.calibration.stop()
    if (s) status.value = s as any
    stage.value = 'done'
  }

  async function teardown() {
    clearCountdown()
    clearPoll()
    await backend.calibration.stop()
    void backend.calibration.stopHotkeyWatch()
    if (unsubToggle) {
      unsubToggle()
      unsubToggle = null
    }
  }

  return {
    stage,
    countdown,
    hotkeyWarn,
    liveAbsDx,
    liveAbsDy,
    open,
    teardown,
    resetSession,
  }
}
