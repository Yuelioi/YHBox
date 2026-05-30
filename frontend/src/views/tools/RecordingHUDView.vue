<template>
  <!-- 录制控制 HUD: 260×96 Frameless + AlwaysOnTop + 透明背景 (BackgroundTypeTransparent).
       卡片外是真透明 (露桌面/游戏, 修圆角黑角); 卡片本体半透明玻璃 (bg-*/85 + backdrop-blur).
       态:
         - countdown: 单行大数字 3/2/1 (主窗口 'recording:countdown')
         - recording: 双行 — REC 红点+计时+模式 / 暂停·停止 / F12 hint
         - paused:    双行 — ‖ 已暂停+冻结计时+模式 / 继续·停止 / F12 hint
         - idle:      启动期兜底 (一般看不到) -->
  <div class="h-screen w-screen flex items-center justify-center select-none p-1.5">
    <!-- resume countdown (继续录制前的 3s 倒计时; 优先于 paused 卡片显示) -->
    <div
      v-if="resumeCountdown > 0"
      class="w-full rounded-xl bg-zinc-900/85 backdrop-blur border border-success/60 px-3 py-2 flex items-center gap-2 shadow-lg"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-player-play-filled" class="size-4 text-success animate-pulse shrink-0" />
      <span class="text-[11px] text-success tracking-wide shrink-0">继续录制</span>
      <span class="text-3xl text-success font-bold tabular-nums leading-none ml-0.5">{{ resumeCountdown }}</span>
      <span class="text-[11px] text-zinc-400 shrink-0">秒后继续 · 切到游戏</span>
    </div>

    <!-- countdown -->
    <div
      v-else-if="state === 'countdown'"
      class="w-full rounded-xl bg-zinc-900/85 backdrop-blur border border-primary/60 px-3 py-2 flex items-center gap-2 shadow-lg"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-circle-dot" class="size-4 text-primary animate-pulse shrink-0" />
      <span class="text-[11px] text-primary tracking-wide shrink-0">{{ modeLabel }}</span>
      <span class="text-3xl text-primary font-bold tabular-nums leading-none ml-0.5">{{ countdownSec }}</span>
      <span class="text-[11px] text-zinc-400 shrink-0">秒后开始 · 切到游戏</span>
    </div>

    <!-- recording / paused: 双行卡片 -->
    <div
      v-else-if="state === 'recording' || state === 'paused'"
      class="w-full rounded-xl bg-zinc-900/85 backdrop-blur px-3 py-2 flex flex-col gap-1.5 border shadow-lg"
      :class="state === 'paused' ? 'border-amber-500/50' : 'border-error/50'"
      style="--wails-draggable: drag"
    >
      <!-- 第一行: 状态 + 计时 + 模式 -->
      <div class="flex items-center gap-2">
        <template v-if="state === 'recording'">
          <span class="size-2.5 rounded-full bg-error animate-pulse shrink-0" />
          <span class="text-xs text-error font-semibold shrink-0">REC</span>
        </template>
        <template v-else>
          <UIcon name="i-tabler-player-pause-filled" class="size-3.5 text-amber-400 shrink-0" />
          <span class="text-xs text-amber-400 font-semibold shrink-0">已暂停</span>
        </template>
        <span class="text-sm text-zinc-100 font-mono tabular-nums">{{ elapsedLabel }}</span>
        <div class="flex-1" />
        <span v-if="modeLabel" class="text-[10px] text-zinc-400 shrink-0">{{ modeLabel }}</span>
      </div>

      <!-- 第二行: 暂停/继续 + 停止 -->
      <div class="flex items-center gap-1.5">
        <UButton
          v-if="state === 'recording'"
          size="xs"
          color="warning"
          variant="soft"
          icon="i-tabler-player-pause-filled"
          block
          class="flex-1"
          style="--wails-draggable: no-drag"
          @click="onPause"
        >暂停</UButton>
        <UButton
          v-else
          size="xs"
          color="success"
          variant="soft"
          icon="i-tabler-player-play-filled"
          block
          class="flex-1"
          style="--wails-draggable: no-drag"
          @click="onResume"
        >继续</UButton>
        <UButton
          size="xs"
          color="error"
          variant="solid"
          icon="i-tabler-player-stop-filled"
          block
          class="flex-1"
          style="--wails-draggable: no-drag"
          :title="`点此或按 ${stopKey} 停止`"
          @click="onStop"
        >停止</UButton>
      </div>

      <!-- 热键 hint (小字, 不进按钮防截断). 暂停/继续走热键不污染录制内容 -->
      <div class="text-[10px] text-zinc-500 text-center leading-none">{{ stopKey }} 停止 · {{ pauseKey }} 暂停/继续</div>
    </div>

    <!-- idle 兜底 -->
    <div
      v-else
      class="w-full rounded-xl bg-zinc-900/85 backdrop-blur border border-zinc-700/60 px-3 py-2 flex items-center gap-2 shadow-lg"
      style="--wails-draggable: drag"
    >
      <UIcon name="i-tabler-loader" class="size-3.5 text-zinc-400 animate-spin shrink-0" />
      <span class="text-[11px] text-zinc-400">准备录制...</span>
    </div>
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
</script>
