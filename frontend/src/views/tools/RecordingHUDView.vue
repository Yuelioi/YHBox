<template>
  <!-- 录制控制 HUD: 360×200 实心盒子窗 (Frameless + AlwaysOnTop), 跟校准窗同风格.
       标题栏 / 大号状态 + 计时 / 底栏按钮.
       态: countdown(3/2/1) · recording · paused · idle(兜底). -->
  <div class="h-screen w-screen bg-default flex flex-col select-none">
    <header
      class="flex items-center gap-2 px-3 py-2 border-b border-default shrink-0"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-video" class="size-4 text-error" />
      <h3 class="text-xs font-medium text-highlighted">鼠标录制</h3>
      <span class="ml-auto" />
      <UButton
        size="xs"
        variant="ghost"
        color="neutral"
        icon="i-tabler-x"
        style="--wails-draggable: no-drag"
        title="关闭悬浮窗 (录制不停, 仍可按热键停)"
        @click="onCloseHud"
      />
    </header>

    <div class="flex-1 min-h-0 flex flex-col items-center justify-center px-3 py-3 gap-2">
      <!-- 状态面板: 深底 + 语义实彩描边 (跟输入校准 HUD 同风格) -->
      <!-- resume countdown (继续录制前的 3s 倒计时; 优先于 paused 显示) -->
      <div
        v-if="resumeCountdown > 0"
        class="w-full rounded-lg border border-success/40 bg-success/10 px-4 py-3 text-center space-y-1"
      >
        <UIcon name="i-tabler-player-play-filled" class="size-6 text-success animate-pulse mx-auto" />
        <div class="text-4xl font-mono tabular-nums text-success">{{ resumeCountdown }}</div>
        <p class="text-[11px] text-success/80">秒后继续 · 切到游戏</p>
      </div>

      <!-- countdown -->
      <div
        v-else-if="state === 'countdown'"
        class="w-full rounded-lg border border-primary/40 bg-primary/10 px-4 py-3 text-center space-y-1"
      >
        <span class="text-[11px] text-primary">{{ modeLabel }}</span>
        <div class="text-5xl font-mono tabular-nums text-primary">{{ countdownSec }}</div>
        <p class="text-[11px] text-dimmed">秒后开始 · 切到游戏</p>
      </div>

      <!-- recording -->
      <div
        v-else-if="state === 'recording'"
        class="w-full rounded-lg border border-error/40 bg-error/10 px-4 py-3 text-center space-y-1"
      >
        <div class="flex items-center justify-center gap-2">
          <span class="size-2.5 rounded-full bg-error animate-pulse" />
          <span class="text-xs text-error font-semibold">REC</span>
        </div>
        <div class="text-4xl font-mono tabular-nums text-highlighted">{{ elapsedLabel }}</div>
        <p v-if="modeLabel" class="text-[10px] text-dimmed">{{ modeLabel }}</p>
      </div>

      <!-- paused -->
      <div
        v-else-if="state === 'paused'"
        class="w-full rounded-lg border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-center space-y-1"
      >
        <div class="flex items-center justify-center gap-2">
          <UIcon name="i-tabler-player-pause-filled" class="size-3.5 text-amber-400" />
          <span class="text-xs text-amber-400 font-semibold">已暂停</span>
        </div>
        <div class="text-4xl font-mono tabular-nums text-highlighted">{{ elapsedLabel }}</div>
        <p v-if="modeLabel" class="text-[10px] text-dimmed">{{ modeLabel }}</p>
      </div>

      <!-- idle 兜底 -->
      <div
        v-else
        class="w-full rounded-lg border border-dashed border-default/60 bg-elevated/40 px-4 py-3 text-center space-y-1"
      >
        <UIcon name="i-tabler-loader" class="size-5 text-dimmed animate-spin mx-auto" />
        <p class="text-[11px] text-dimmed">准备录制...</p>
      </div>
    </div>

    <footer class="flex items-center gap-2 px-3 py-2 border-t border-default shrink-0">
      <template v-if="state === 'recording' || state === 'paused'">
        <UButton
          v-if="state === 'recording'"
          size="xs"
          color="warning"
          variant="soft"
          icon="i-tabler-player-pause-filled"
          class="flex-1 justify-center"
          @click="onPause"
        >暂停</UButton>
        <UButton
          v-else
          size="xs"
          color="success"
          variant="soft"
          icon="i-tabler-player-play-filled"
          class="flex-1 justify-center"
          @click="onResume"
        >继续</UButton>
        <UButton
          size="xs"
          color="error"
          variant="solid"
          icon="i-tabler-player-stop-filled"
          class="flex-1 justify-center"
          :title="`点此或按 ${stopKey} 停止`"
          @click="onStop"
        >停止</UButton>
        <span class="text-[10px] text-dimmed shrink-0 ml-0.5">{{ stopKey }}</span>
      </template>
      <span v-else class="text-[10px] text-dimmed mx-auto">{{ stopKey }} 停止 · {{ pauseKey }} 暂停/继续</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { Events, Window } from '@wailsio/runtime'
import { backend } from '@/lib/backend'

type State = 'idle' | 'countdown' | 'recording' | 'paused'

const state = ref<State>('idle')
const countdownSec = ref(0)
const resumeCountdown = ref(0) // >0 时显示"继续录制"倒计时 (优先于 paused 卡片)
const mode = ref<'precise' | 'simple'>('precise')
const stopKey = ref('F12') // 停录键标签, 由 recording:countdown 带来 (settings 配的)
const pauseKey = ref('F11') // 暂停/继续切换键标签
const elapsedMs = ref(0)
// 计时基准: 录制时长 = now - startedAt - pausedMs (扣除累计暂停); 暂停态冻结值另算.
let startedAt = 0
let pausedMs = 0
let timer: ReturnType<typeof setInterval> | null = null

const modeLabel = computed(() => (mode.value === 'precise' ? '精准录制' : '简易录制'))

const elapsedLabel = computed(() => {
  const s = Math.max(0, Math.floor(elapsedMs.value / 1000))
  const mm = String(Math.floor(s / 60)).padStart(2, '0')
  const ss = String(s % 60).padStart(2, '0')
  return `${mm}:${ss}`
})

function stopTimer() {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}
function startTimer() {
  stopTimer()
  timer = setInterval(() => {
    elapsedMs.value = Date.now() - startedAt - pausedMs
  }, 100)
}

function pickMode(fm: any) {
  if (fm === 'simple' || fm === 'precise') mode.value = fm
}

// 主窗口 useRecording 倒计时广播
const offCountdown = Events.On('recording:countdown', (e: any) => {
  const payload = e?.data?.[0] ?? e?.data ?? e
  const sec = payload?.sec ?? 0
  if (payload?.mode) mode.value = payload.mode
  if (payload?.stopKey) stopKey.value = payload.stopKey
  if (payload?.pauseKey) pauseKey.value = payload.pauseKey
  if (sec > 0) {
    state.value = 'countdown'
    countdownSec.value = sec
  } else if (state.value === 'countdown') {
    // 倒计时取消 (主窗口在循环里设 sec=0 退出) — HUD 自己关
    Window.Close()
  }
}) as unknown as () => void

// 后端权威状态机镜像. recording → 计时; paused → 冻结计时; idle/finalizing (且之前在 session) → 自关.
// 即使主窗口没调 closeRecordingHUD (F12/异常停录), HUD 也不会残留.
const offState = Events.On('recording:state', (e: any) => {
  const st = e?.data?.[0] ?? e?.data ?? e
  const phase = st?.phase
  if (phase === 'recording') {
    pickMode(st?.filterMode)
    startedAt = (st?.startedAtMs ?? 0) > 0 ? st.startedAtMs : Date.now()
    pausedMs = st?.pausedMs ?? 0
    state.value = 'recording'
    elapsedMs.value = Date.now() - startedAt - pausedMs
    startTimer() // 每次进 recording (含 resume) 重建 timer, 幂等
  } else if (phase === 'paused') {
    pickMode(st?.filterMode)
    if ((st?.startedAtMs ?? 0) > 0) startedAt = st.startedAtMs
    pausedMs = st?.pausedMs ?? 0
    const pausedAt = (st?.pausedAtMs ?? 0) > 0 ? st.pausedAtMs : Date.now()
    state.value = 'paused'
    elapsedMs.value = pausedAt - startedAt - pausedMs // 冻结在暂停时刻
    stopTimer()
  } else if (state.value === 'recording' || state.value === 'paused') {
    // idle / finalizing 且之前在 session → 录制结束, 关 HUD.
    stopTimer()
    Window.Close()
  }
}) as unknown as () => void

// 暂停热键命中(后端)且当前 paused → emit 'recording:resume-hotkey' → HUD 走继续倒计时.
const offResumeHotkey = Events.On('recording:resume-hotkey', () => {
  void startResumeCountdown()
}) as unknown as () => void

let resumeTimer: ReturnType<typeof setTimeout> | null = null
function clearResumeTimer() {
  if (resumeTimer) { clearTimeout(resumeTimer); resumeTimer = null }
}

// startResumeCountdown 继续录制前的 3s 倒计时 (HUD 按钮 / 暂停热键都走它).
// 倒计时期间仍 paused (recorder 不录), 倒计时完才调 resume RPC → 真正继续.
async function startResumeCountdown() {
  if (state.value !== 'paused' || resumeCountdown.value > 0) return
  for (let i = 3; i >= 1; i--) {
    if (state.value !== 'paused') { resumeCountdown.value = 0; return } // 期间停录 → 中止
    resumeCountdown.value = i
    await new Promise<void>((r) => { resumeTimer = setTimeout(r, 1000) })
  }
  resumeCountdown.value = 0
  if (state.value !== 'paused') return
  try { await backend.recording.resume() } catch (e) { console.warn('继续失败', e) }
}

onUnmounted(() => {
  stopTimer()
  clearResumeTimer()
  offCountdown?.()
  offState?.()
  offResumeHotkey?.()
})

async function onPause() {
  try { await backend.recording.pause() } catch (e) { console.warn('暂停失败', e) }
}
function onResume() {
  void startResumeCountdown()
}
async function onStop() {
  try {
    await backend.recording.stopAsync()
  } catch (e) {
    console.warn('录制停止失败', e)
  } finally {
    Window.Close()
  }
}

// 关 HUD 悬浮窗 (录制不停, 仍可按热键停). 录制结束时 recording:state 监听会自动关。
function onCloseHud() {
  Window.Close()
}
</script>
