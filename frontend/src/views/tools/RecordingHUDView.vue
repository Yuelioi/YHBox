<template>
  <!-- 录制控制 HUD: 220×60 Frameless + AlwaysOnTop.
       三态:
         - countdown: 大数字 3/2/1 (主窗口启动倒计时时 emit 'recording:countdown')
         - recording: REC 红点 + 已用时间 + 停止按钮 (后端 'recording:state' phase=recording)
         - idle:      启动期空状态 (兜底, 一般看不到)
  -->
  <div
    class="h-screen w-screen flex items-center gap-2 px-2.5 select-none rounded-md transition-colors"
    :class="state === 'countdown'
      ? 'bg-primary/10 border-2 border-primary/70'
      : state === 'recording'
        ? 'bg-error/10 border border-error/60'
        : 'bg-default border border-default'"
    style="--wails-draggable: drag"
  >
    <template v-if="state === 'countdown'">
      <UIcon name="i-tabler-circle-dot" class="size-3.5 text-primary animate-pulse shrink-0" />
      <span class="text-[10px] text-primary tracking-wider shrink-0">{{ modeLabel }}</span>
      <span class="text-3xl text-primary font-bold tabular-nums leading-none ml-0.5">{{ countdownSec }}</span>
      <span class="text-[10px] text-toned shrink-0">秒后开始</span>
      <div class="flex-1" />
      <span class="text-[10px] text-dimmed shrink-0">切到游戏</span>
    </template>

    <template v-else-if="state === 'recording'">
      <span class="size-2.5 rounded-full bg-error animate-pulse shrink-0" />
      <span class="text-xs text-error font-semibold shrink-0">REC</span>
      <span v-if="modeLabel" class="text-[10px] text-dimmed shrink-0">{{ modeLabel }}</span>
      <span class="text-sm text-default font-mono tabular-nums ml-0.5">{{ elapsedLabel }}</span>
      <div class="flex-1" />
      <!-- 停录键提示 + 点击停止: 同一个按钮既是提示也是热区 -->
      <UButton
        size="xs"
        color="error"
        variant="solid"
        icon="i-tabler-player-stop-filled"
        style="--wails-draggable: no-drag"
        :title="`点此或按 ${stopKey} 停止录制`"
        @click="onStop"
      >
        <span class="font-mono">{{ stopKey }}</span>
      </UButton>
    </template>

    <template v-else>
      <UIcon name="i-tabler-loader" class="size-3 text-dimmed animate-spin shrink-0" />
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
const stopKey = ref('F12') // 停录键标签, 由 recording:countdown 带来 (settings 配的)
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
  if (payload?.stopKey) stopKey.value = payload.stopKey
  if (sec > 0) {
    state.value = 'countdown'
    countdownSec.value = sec
  } else if (state.value === 'countdown') {
    // 倒计时取消 (主窗口在循环里设 sec=0 退出) — HUD 自己关
    Window.Close()
  }
}) as unknown as () => void

// 后端权威状态机镜像. recording → 进录制视图 + 计时; idle/finalizing (且之前在录) → HUD 自关.
// 即使主窗口没调 closeRecordingHUD (F12/异常停录), HUD 也不会残留 "REC".
const offState = Events.On('recording:state', (e: any) => {
  const st = e?.data?.[0] ?? e?.data ?? e
  const phase = st?.phase
  if (phase === 'recording') {
    if (st?.filterMode === 'simple' || st?.filterMode === 'precise') mode.value = st.filterMode
    if (state.value !== 'recording') {
      state.value = 'recording'
      startedAt = st?.startedAtMs > 0 ? st.startedAtMs : Date.now()
      elapsedMs.value = Date.now() - startedAt
      if (timer) clearInterval(timer)
      timer = setInterval(() => {
        elapsedMs.value = Date.now() - startedAt
      }, 100)
    }
  } else if (state.value === 'recording') {
    // idle / finalizing 且之前在录 → 录制结束, 关 HUD.
    if (timer) { clearInterval(timer); timer = null }
    Window.Close()
  }
}) as unknown as () => void

// HUD 完全事件驱动: state 仅在 'recording:countdown' / 'recording:state' / Window.Close 时变.
// 启动期 race (HUD On 注册前主窗口 emit 第一次 countdown): wails3 Events 走主 process 队列,
// 子窗口注册前的事件应该被丢; 真出现这种 race 用户能看到 "准备录制..." 1 秒后倒计时进入正常状态.

onUnmounted(() => {
  if (timer) clearInterval(timer)
  offCountdown?.()
  offState?.()
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
