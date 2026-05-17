<template>
  <!-- 录制控制 HUD: 220×60 Frameless + AlwaysOnTop.
       三态:
         - countdown: 大数字 3/2/1 (主窗口启动倒计时时 emit 'recording:countdown')
         - recording: REC 红点 + 已用时间 + 停止按钮 (emit 'recording:started' 后)
         - idle:      启动期空状态 (兜底, 一般看不到)
  -->
  <div
    class="h-screen w-screen bg-default flex items-center gap-2 px-2 select-none"
    :class="state === 'countdown' ? 'border-2 border-primary' : 'border border-error/60'"
    style="--wails-draggable: drag"
  >
    <template v-if="state === 'countdown'">
      <div class="flex items-center gap-1.5 shrink-0">
        <UIcon name="i-tabler-circle-dot" class="size-3 text-primary animate-pulse" />
        <span class="text-[10px] text-primary tracking-wider">{{ modeLabel }}</span>
      </div>
      <span class="text-3xl text-primary font-bold tabular-nums leading-none ml-1">{{ countdownSec }}</span>
      <span class="text-[10px] text-toned">秒后开始</span>
      <div class="flex-1" />
      <span class="text-[10px] text-dimmed">切到游戏</span>
    </template>

    <template v-else-if="state === 'recording'">
      <div class="flex items-center gap-1.5 shrink-0">
        <span class="size-2.5 rounded-full bg-error animate-pulse" />
        <span class="text-xs text-error font-medium">REC</span>
        <span v-if="modeLabel" class="text-[10px] text-dimmed">{{ modeLabel }}</span>
      </div>
      <span class="text-sm text-default font-mono tabular-nums">{{ elapsedLabel }}</span>
      <div class="flex-1" />
      <UButton
        size="xs"
        color="error"
        variant="solid"
        icon="i-tabler-square"
        style="--wails-draggable: no-drag"
        title="停止录制 (= 游戏前台按 F12)"
        @click="onStop"
      >停止</UButton>
    </template>

    <template v-else>
      <UIcon name="i-tabler-loader" class="size-3 text-dimmed animate-spin" />
      <span class="text-[11px] text-dimmed">准备录制...</span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Events, Window } from '@wailsio/runtime'
import { backend } from '@/lib/backend'

type State = 'idle' | 'countdown' | 'recording'

const state = ref<State>('idle')
const countdownSec = ref(0)
const mode = ref<'precise' | 'simple'>('precise')
const elapsedMs = ref(0)
let startedAt = 0
let timer: ReturnType<typeof setInterval> | null = null

const modeLabel = computed(() => (mode.value === 'precise' ? '精准录制' : '简易录制'))

const elapsedLabel = computed(() => {
  const s = Math.floor(elapsedMs.value / 1000)
  const mm = String(Math.floor(s / 60)).padStart(2, '0')
  const ss = String(s % 60).padStart(2, '0')
  return `${mm}:${ss}`
})

// 主窗口 useRecording 倒计时广播
const offCountdown = Events.On('recording:countdown', (e: any) => {
  const payload = e?.data?.[0] ?? e?.data ?? e
  const sec = payload?.sec ?? 0
  if (payload?.mode) mode.value = payload.mode
  if (sec > 0) {
    state.value = 'countdown'
    countdownSec.value = sec
  } else if (state.value === 'countdown') {
    // 倒计时取消 (主窗口在循环里设 sec=0 退出) — HUD 自己关
    Window.Close()
  }
}) as unknown as () => void

const offStarted = Events.On('recording:started', (e: any) => {
  const payload = e?.data?.[0] ?? e?.data ?? e
  if (payload?.mode) mode.value = payload.mode
  state.value = 'recording'
  startedAt = Date.now()
  elapsedMs.value = 0
  if (timer) clearInterval(timer)
  timer = setInterval(() => {
    elapsedMs.value = Date.now() - startedAt
  }, 100)
}) as unknown as () => void

// HUD 完全事件驱动: state 仅在 'recording:countdown' / 'recording:started' / Window.Close 时变.
// 启动期 race (HUD On 注册前主窗口 emit 第一次 countdown): wails3 Events 走主 process 队列,
// 子窗口注册前的事件应该被丢; 真出现这种 race 用户能看到 "准备录制..." 1 秒后倒计时进入正常状态.

onUnmounted(() => {
  if (timer) clearInterval(timer)
  offCountdown?.()
  offStarted?.()
})

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
