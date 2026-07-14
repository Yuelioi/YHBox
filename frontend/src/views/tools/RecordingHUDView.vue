<template>
  <HudShell
    icon="i-tabler-video"
    accent="error"
    :title="t('recordingHud.title')"
    :subtitle="t('recordingHud.subtitle')"
    :status="windowStatus"
    :status-active="state === 'recording'"
    :close-title="t('recordingHud.close_hint')"
    @close="onCloseHud"
  >
    <div class="flex min-h-0 flex-1 flex-col gap-3 p-3">
      <HudStatePanel
        v-if="resumeCountdown > 0"
        tone="success"
        icon="i-tabler-player-play-filled"
        :eyebrow="t('recordingHud.resuming')"
        :value="resumeCountdown"
        :hint="t('recordingHud.resume_hint')"
      />

      <HudStatePanel
        v-else-if="state === 'countdown'"
        tone="primary"
        icon="i-tabler-hourglass-high"
        :eyebrow="modeLabel"
        :value="countdownSec"
        :hint="t('recordingHud.countdown_hint')"
      />

      <HudStatePanel
        v-else-if="state === 'recording'"
        tone="error"
        active
        eyebrow="REC"
        :value="elapsedLabel"
        :hint="modeLabel"
      />

      <HudStatePanel
        v-else-if="state === 'paused'"
        tone="warning"
        icon="i-tabler-player-pause-filled"
        :eyebrow="t('recordingHud.paused')"
        :value="elapsedLabel"
        :hint="modeLabel"
      />

      <HudStatePanel
        v-else
        tone="neutral"
        icon="i-tabler-loader-2"
        :eyebrow="t('recordingHud.preparing')"
        :hint="t('recordingHud.preparing_hint')"
      />

      <div class="mt-auto flex items-center gap-2 border-t border-default pt-3">
        <template v-if="state === 'recording' || state === 'paused'">
          <UButton
            v-if="state === 'recording'"
            size="sm"
            color="warning"
            variant="soft"
            icon="i-tabler-player-pause-filled"
            class="flex-1"
            @click="onPause"
          >
            {{ t('recordingHud.pause') }}
          </UButton>
          <UButton
            v-else
            size="sm"
            color="success"
            variant="soft"
            icon="i-tabler-player-play-filled"
            class="flex-1"
            @click="onResume"
          >
            {{ t('recordingHud.resume') }}
          </UButton>
          <UButton
            size="sm"
            color="primary"
            icon="i-tabler-player-stop-filled"
            class="flex-1"
            :title="t('recordingHud.stop_hint', { key: stopKey })"
            @click="onStop"
          >
            {{ t('recordingHud.stop') }}
          </UButton>
          <UButton
            size="sm"
            color="error"
            variant="ghost"
            icon="i-tabler-trash"
            class="flex-1"
            @click="armOrCancel"
          >
            {{ cancelArmed ? t('recordingHud.cancel_confirm') : t('recordingHud.cancel') }}
          </UButton>
        </template>
        <span v-else class="mx-auto text-xs text-dimmed">
          {{ t('recordingHud.shortcut_hint', { stop: stopKey, pause: pauseKey }) }}
        </span>
      </div>
    </div>
  </HudShell>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Events, Window } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import HudShell from '@/components/tools/HudShell.vue'
import HudStatePanel from '@/components/tools/HudStatePanel.vue'

type State = 'idle' | 'countdown' | 'recording' | 'paused'

const state = ref<State>('idle')
const { t } = useI18n()
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
const cancelArmed = ref(false)
let cancelTimer: ReturnType<typeof setTimeout> | null = null

const modeLabel = computed(() =>
  t(mode.value === 'precise' ? 'recordingSave.mode_precise' : 'recordingSave.mode_simple'),
)
const windowStatus = computed(() => {
  if (resumeCountdown.value > 0 || state.value === 'countdown') return t('recordingHud.countdown')
  if (state.value === 'recording') return t('recordingHud.recording')
  if (state.value === 'paused') return t('recordingHud.paused')
  return t('recordingHud.preparing')
})

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
function clearCancelTimer() {
  if (cancelTimer) clearTimeout(cancelTimer)
  cancelTimer = null
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
  if (resumeTimer) {
    clearTimeout(resumeTimer)
    resumeTimer = null
  }
}

// startResumeCountdown 继续录制前的 3s 倒计时 (HUD 按钮 / 暂停热键都走它).
// 倒计时期间仍 paused (recorder 不录), 倒计时完才调 resume RPC → 真正继续.
async function startResumeCountdown() {
  if (state.value !== 'paused' || resumeCountdown.value > 0) return
  for (let i = 3; i >= 1; i--) {
    if (state.value !== 'paused') {
      resumeCountdown.value = 0
      return
    } // 期间停录 → 中止
    resumeCountdown.value = i
    await new Promise<void>((r) => {
      resumeTimer = setTimeout(r, 1000)
    })
  }
  resumeCountdown.value = 0
  if (state.value !== 'paused') return
  try {
    await backend.recording.resume()
  } catch (e) {
    console.warn('继续失败', e)
  }
}

onUnmounted(() => {
  stopTimer()
  clearResumeTimer()
  clearCancelTimer()
  offCountdown?.()
  offState?.()
  offResumeHotkey?.()
})

async function onPause() {
  try {
    await backend.recording.pause()
  } catch (e) {
    console.warn('暂停失败', e)
  }
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

async function armOrCancel() {
  if (!cancelArmed.value) {
    cancelArmed.value = true
    cancelTimer = setTimeout(() => {
      cancelArmed.value = false
    }, 4000)
    return
  }
  clearCancelTimer()
  try {
    await backend.recording.cancel()
  } catch (e) {
    console.warn('录制取消失败', e)
  } finally {
    Window.Close()
  }
}

// 关 HUD 悬浮窗 (录制不停, 仍可按热键停). 录制结束时 recording:state 监听会自动关。
function onCloseHud() {
  Window.Close()
}
</script>
