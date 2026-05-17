<template>
  <UModal :open="open" @update:open="onUpdateOpen" :ui="{ content: 'sm:max-w-[560px]' }">
    <template #content>
      <div class="bg-default flex flex-col">
        <header class="flex items-center gap-2 px-5 py-3 border-b border-default">
          <UIcon name="i-tabler-target" class="size-4 text-primary" />
          <h3 class="text-sm font-medium text-highlighted">鼠标 DPI 校准</h3>
          <span class="ml-auto" />
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-x"
            @click="onCancel"
          />
        </header>

        <div class="p-5 space-y-4">
          <!-- 状态分支 -->
          <div
            v-if="stage === 'waiting'"
            class="rounded-md border border-dashed border-default/60 bg-elevated/40 p-5 text-center space-y-3"
          >
            <UIcon name="i-tabler-keyboard" class="size-8 text-primary mx-auto" />
            <p class="text-sm text-highlighted">
              切到游戏，按
              <code class="bg-elevated/60 px-1.5 py-0.5 rounded text-toned">F8</code> 开始
            </p>
            <p class="text-[11px] text-dimmed">
              按下后 3 秒倒计时，期间最后调整姿态；倒计时结束自动开始累计
            </p>
          </div>

          <div
            v-else-if="stage === 'countingDown'"
            class="rounded-md border border-amber-500/40 bg-amber-500/10 p-5 text-center space-y-2"
          >
            <div class="text-6xl font-mono tabular-nums text-amber-400">{{ countdown }}</div>
            <p class="text-sm text-amber-300">即将开始累计，请就位</p>
            <p class="text-[10px] text-dimmed">提前按 F8 = 立即开始</p>
          </div>

          <div
            v-else-if="stage === 'accumulating'"
            class="rounded-md border border-emerald-500/40 bg-emerald-500/10 p-5 text-center space-y-2"
          >
            <div class="flex items-center justify-center gap-2 text-emerald-300">
              <span class="size-2 rounded-full bg-emerald-400 animate-pulse" />
              <span class="text-sm">累计中 · 原地转 360°</span>
            </div>
            <div class="text-4xl font-mono tabular-nums text-emerald-300">{{ liveAbsDx }}</div>
            <p class="text-[10px] text-dimmed font-mono">|dy| {{ liveAbsDy }}（垂直，仅参考）</p>
            <p class="text-[11px] text-emerald-300/80 pt-2">
              转完按 <code class="bg-emerald-500/20 px-1.5 py-0.5 rounded">F8</code> 停止
            </p>
          </div>

          <div
            v-else-if="stage === 'done'"
            class="rounded-md border border-primary/40 bg-primary/10 p-5 text-center space-y-2"
          >
            <UIcon name="i-tabler-circle-check" class="size-8 text-primary mx-auto" />
            <p class="text-sm text-highlighted">已记录</p>
            <div class="text-4xl font-mono tabular-nums text-primary">{{ liveAbsDx }}</div>
            <p class="text-[11px] text-dimmed">点下方「保存」写入本机基准；或按 F8 重测</p>
          </div>

          <p v-if="hotkeyWarn" class="text-[11px] text-warning">
            <UIcon name="i-tabler-alert-triangle" class="size-3 inline" />
            {{ hotkeyWarn }}
          </p>
        </div>

        <footer class="px-5 py-3 border-t border-default flex items-center gap-2">
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-refresh"
            :disabled="stage === 'waiting' || stage === 'countingDown'"
            @click="resetSession"
            >重测</UButton
          >
          <span class="ml-auto" />
          <UButton variant="ghost" color="neutral" @click="onCancel">取消</UButton>
          <UButton
            color="primary"
            icon="i-tabler-device-floppy"
            :disabled="stage !== 'done' || liveAbsDx === 0"
            @click="onSave"
          >
            保存（{{ liveAbsDx }}）
          </UButton>
        </footer>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  save: [counts: number]
}>()

type Stage = 'waiting' | 'countingDown' | 'accumulating' | 'done'

const stage = ref<Stage>('waiting')
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

async function openCalibrator() {
  hotkeyWarn.value = ''
  resetSession()
  // 订阅 F8 全局热键事件
  if (!unsubToggle) {
    const off = (await Events.On('calibration:toggle', onToggleHotkey)) as unknown as () => void
    unsubToggle = typeof off === 'function' ? off : null
  }
}

function resetSession() {
  stage.value = 'waiting'
  countdown.value = 3
  status.value = { active: false, absDx: 0, absDy: 0 }
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  // 已开着的 calibration 后端要先停
  void backend.calibration.stop()
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function onToggleHotkey() {
  if (!props.open) return
  switch (stage.value) {
    case 'waiting':
      beginCountdown()
      break
    case 'countingDown':
      // 提前按 F8 = 立即结束倒计时进入累计
      finishCountdown()
      break
    case 'accumulating':
      stopAccumulating()
      break
    case 'done':
      // 重测
      resetSession()
      // 立刻再进倒计时（用户连按 F8 → 重新开始一轮）
      beginCountdown()
      break
  }
}

function beginCountdown() {
  stage.value = 'countingDown'
  countdown.value = 3
  countdownTimer = setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0) finishCountdown()
  }, 1000)
}

async function finishCountdown() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  stage.value = 'accumulating'
  const ok = await backend.calibration.start()
  if (ok === undefined) {
    hotkeyWarn.value = '校准服务启动失败（端口被占？）'
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
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  const s = await backend.calibration.stop()
  if (s) status.value = s as any
  stage.value = 'done'
}

async function teardown() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  await backend.calibration.stop()
  if (unsubToggle) {
    unsubToggle()
    unsubToggle = null
  }
}

async function onCancel() {
  await teardown()
  emit('update:open', false)
}

async function onSave() {
  await teardown()
  const counts = liveAbsDx.value
  if (counts > 0) {
    emit('save', counts)
  }
  emit('update:open', false)
}

function onUpdateOpen(v: boolean) {
  emit('update:open', v)
  if (!v) onCancel()
}

// watch props.open 触发 open/teardown
watch(
  () => props.open,
  (v) => {
    if (v) {
      openCalibrator()
    } else {
      void teardown()
    }
  },
)

onUnmounted(() => {
  void teardown()
})
</script>
