<template>
  <HudShell
    icon="i-tabler-video"
    accent="error"
    :title="t('recordingHud.title')"
    :status="windowStatus"
    :status-active="state === 'recording'"
    :close-title="t('recordingHud.close_hint')"
    @close="onCloseHud"
  >
    <template #actions>
      <span
        class="grid size-7 place-items-center text-primary"
        :title="t('recordingHud.pinned_hint')"
        :aria-label="t('recordingHud.pinned_hint')"
      >
        <UIcon name="i-tabler-pin-filled" class="size-3.5" />
      </span>
    </template>
    <div class="flex min-h-0 flex-1 flex-col gap-3 p-3">
      <div class="live-hud__stage">
        <HudStatePanel
          v-if="resumeCountdown > 0"
          tone="success"
          icon="i-tabler-player-play-filled"
          :eyebrow="t('recordingHud.resuming')"
          :value="resumeCountdown"
          :hint="t('recordingHud.resume_hint')"
        />

        <HudStatePanel
          v-else-if="state === 'armed'"
          tone="primary"
          icon="i-tabler-keyboard"
          :eyebrow="modeLabel"
          :value="startKey"
          :hint="t('recordingHud.waiting_hint', { key: startKey })"
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
      </div>

      <div class="flex shrink-0 items-center gap-2 border-t border-default pt-3">
        <template v-if="state === 'armed'">
          <UButton
            size="sm"
            color="primary"
            icon="i-tabler-player-play-filled"
            class="flex-1"
            @click="onStartCountdown"
          >
            {{ t('recordingHud.start_countdown') }}
          </UButton>
          <UButton
            size="sm"
            color="error"
            variant="ghost"
            icon="i-tabler-x"
            class="flex-1"
            @click="cancelPreparation"
          >
            {{ t('common.cancel') }}
          </UButton>
        </template>
        <template v-else-if="state === 'countdown'">
          <UButton
            size="sm"
            color="error"
            variant="ghost"
            icon="i-tabler-x"
            class="flex-1"
            @click="cancelPreparation"
          >
            {{ t('recordingHud.cancel_countdown') }}
          </UButton>
        </template>
        <template v-else-if="state === 'recording' || state === 'paused'">
          <UButton
            v-if="state === 'recording'"
            size="sm"
            color="warning"
            variant="soft"
            icon="i-tabler-player-pause-filled"
            class="flex-1"
            @click="onPause"
          >
            <span>{{ t('recordingHud.pause') }}</span>
            <span class="recording-shortcut">{{ pauseKey }}</span>
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
            <span>{{ t('recordingHud.resume') }}</span>
            <span class="recording-shortcut">{{ pauseKey }}</span>
          </UButton>
          <UButton
            size="sm"
            color="primary"
            icon="i-tabler-player-stop-filled"
            class="flex-1"
            :title="t('recordingHud.stop_hint', { key: stopKey })"
            @click="onStop"
          >
            <span>{{ t('recordingHud.stop') }}</span>
            <span class="recording-shortcut">{{ stopKey }}</span>
          </UButton>
          <UButton
            size="sm"
            color="error"
            variant="ghost"
            icon="i-tabler-trash"
            class="flex-1"
            @click="armOrCancel"
          >
            <span>{{
              cancelArmed ? t('recordingHud.cancel_confirm') : t('recordingHud.cancel')
            }}</span>
            <span class="recording-shortcut">{{ cancelKey }}</span>
          </UButton>
        </template>
        <span v-else class="mx-auto text-xs text-dimmed">
          {{ t('recordingHud.preparing_hint') }}
        </span>
      </div>
    </div>
  </HudShell>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Events, Window } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { useHotkeysStore } from '@/stores/hotkeys'
import HudShell from '@/components/tools/HudShell.vue'
import HudStatePanel from '@/components/tools/HudStatePanel.vue'

type State = 'idle' | 'armed' | 'countdown' | 'recording' | 'paused'

const state = ref<State>('idle')
const { t } = useI18n()
const countdownSec = ref(0)
const countdownEndsAt = ref(0)
const resumeCountdown = ref(0) // >0 时显示"继续录制"倒计时 (优先于 paused 卡片)
const hotkeys = useHotkeysStore()
const startKey = computed(() => hotkeys.keyFor('recording.start', 'F10'))
const stopKey = computed(() => hotkeys.keyFor('recording.stop', 'F12'))
const pauseKey = computed(() => hotkeys.keyFor('recording.pause', 'F11'))
const cancelKey = computed(() => hotkeys.keyFor('recording.cancel', 'F7'))
const mode = ref<'simple' | 'precise'>('simple')
const elapsedMs = ref(0)
let revision = 0
// 计时基准: 录制时长 = now - startedAt - pausedMs (扣除累计暂停); 暂停态冻结值另算.
let startedAt = 0
let pausedMs = 0
let timer: ReturnType<typeof setInterval> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null
const cancelArmed = ref(false)
let cancelTimer: ReturnType<typeof setTimeout> | null = null

const modeLabel = computed(() => t(`recordingSave.mode_${mode.value}`))
const windowStatus = computed(() => {
  if (state.value === 'armed') return t('recordingHud.waiting')
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
function stopCountdownTimer() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}
function refreshCountdown() {
  countdownSec.value = Math.max(0, Math.ceil((countdownEndsAt.value - Date.now()) / 1000))
  if (countdownSec.value <= 0) stopCountdownTimer()
}
function startCountdownTimer() {
  stopCountdownTimer()
  refreshCountdown()
  countdownTimer = setInterval(refreshCountdown, 100)
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

// 后端权威状态机镜像. armed/countdown → 等待开始; recording → 计时; paused → 冻结计时;
// idle/finalizing (且之前在 session) → 自关.
// 即使主窗口没调 closeRecordingHUD (F12/异常停录), HUD 也不会残留.
const offState = Events.On('recording:state', (e: any) => {
  const st = e?.data?.[0] ?? e?.data ?? e
  applySnapshot(st)
}) as unknown as () => void

function applySnapshot(st: any) {
  const nextRevision = typeof st?.revision === 'number' ? st.revision : 0
  if (nextRevision < revision) return
  revision = nextRevision
  const phase = st?.phase
  if (st?.mode === 'simple' || st?.mode === 'precise') mode.value = st.mode
  if (phase === 'armed') {
    stopTimer()
    stopCountdownTimer()
    state.value = 'armed'
  } else if (phase === 'countdown') {
    stopTimer()
    state.value = 'countdown'
    countdownEndsAt.value = st?.countdownEndsAtMs ?? Date.now()
    startCountdownTimer()
  } else if (phase === 'recording') {
    stopCountdownTimer()
    startedAt = (st?.startedAtMs ?? 0) > 0 ? st.startedAtMs : Date.now()
    pausedMs = st?.pausedMs ?? 0
    state.value = 'recording'
    elapsedMs.value = Date.now() - startedAt - pausedMs
    startTimer() // 每次进 recording (含 resume) 重建 timer, 幂等
  } else if (phase === 'paused') {
    if ((st?.startedAtMs ?? 0) > 0) startedAt = st.startedAtMs
    pausedMs = st?.pausedMs ?? 0
    const pausedAt = (st?.pausedAtMs ?? 0) > 0 ? st.pausedAtMs : Date.now()
    state.value = 'paused'
    elapsedMs.value = pausedAt - startedAt - pausedMs // 冻结在暂停时刻
    stopTimer()
  } else if (
    state.value === 'armed' ||
    state.value === 'countdown' ||
    state.value === 'recording' ||
    state.value === 'paused'
  ) {
    stopTimer()
    stopCountdownTimer()
    Window.Close()
  }
}

onMounted(async () => {
  try {
    await hotkeys.reload()
    applySnapshot(await backend.recording.getState())
  } catch (error) {
    console.warn('recording HUD reconcile failed', error)
  }
})

const offHotkeyChanged = backend.events.onHotkeyChanged(() => {
  void hotkeys.reload()
})

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
  stopCountdownTimer()
  clearResumeTimer()
  clearCancelTimer()
  offState?.()
  offResumeHotkey?.()
  offHotkeyChanged?.()
})

async function onStartCountdown() {
  try {
    await backend.recording.beginCountdown()
  } catch (e) {
    console.warn('开始录制倒计时失败', e)
  }
}

async function cancelPreparation() {
  try {
    await backend.recording.cancel()
  } catch (e) {
    console.warn('取消录制准备失败', e)
  } finally {
    Window.Close()
  }
}

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
  if (state.value === 'armed' || state.value === 'countdown') {
    void cancelPreparation()
    return
  }
  Window.Close()
}
</script>

<style scoped>
.live-hud__stage {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1;
}

.recording-shortcut {
  margin-inline-start: 0.2rem;
  font-size: 9px;
  line-height: 1;
  font-weight: 500;
  opacity: 0.62;
}
</style>
