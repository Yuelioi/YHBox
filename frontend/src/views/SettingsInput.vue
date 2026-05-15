<template>
  <div class="p-6 space-y-6 max-w-2xl">
    <header>
      <h2 class="text-base font-medium text-highlighted">输入校准</h2>
      <p class="text-xs text-dimmed mt-1">
        鼠标硬件 DPI 影响"相对位移"类录制（camera/视角转动）的跨电脑回放。 校准后录制时会把基准写入
        Action 元数据，回放时按当前电脑基准按比例缩放。
      </p>
    </header>

    <section class="rounded-md border border-default bg-elevated/40 p-4 space-y-4">
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <h3 class="text-sm font-medium text-highlighted">本机 360° HID counts</h3>
          <p class="text-[11px] text-dimmed mt-0.5">原地转身 360° 鼠标硬件上报的累积 |dx|</p>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <UInputNumber
            v-model="manualCounts"
            :min="0"
            :max="999999"
            :step="100"
            class="w-32"
            @blur="onCommitManual"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-check"
            title="保存手填值"
            @click="onCommitManual"
          />
        </div>
      </div>

      <div class="flex items-center gap-2 flex-wrap">
        <UButton size="sm" color="primary" icon="i-tabler-target" @click="openCalibrator">
          {{ (settings?.ui.mouseCounts360 ?? 0) > 0 ? '重新校准' : '开始校准' }}
        </UButton>
        <UButton
          size="sm"
          variant="soft"
          color="neutral"
          icon="i-tabler-pointer"
          @click="openMouseHUD"
        >
          打开鼠标 HUD
        </UButton>
        <span class="ml-auto text-[11px] text-dimmed">
          也可以从其他电脑分享脚本附带的 counts，直接手填
        </span>
      </div>
    </section>

    <!-- 说明 -->
    <section
      class="rounded-md border border-default/60 bg-default/50 p-4 text-xs text-dimmed space-y-2"
    >
      <h4 class="text-xs uppercase tracking-wider text-toned">怎么用</h4>
      <ol class="list-decimal pl-5 space-y-1">
        <li>点「开始校准」打开对话框</li>
        <li>切到游戏，对准固定参照物，准备好</li>
        <li>
          按 <code class="bg-elevated/60 px-1 rounded text-toned">F8</code> 开始 3
          秒倒计时（不用回到本程序！）
        </li>
        <li>倒计时结束后开始累计 → 原地匀速转一整圈 360°</li>
        <li>转完再按一次 <code class="bg-elevated/60 px-1 rounded text-toned">F8</code> 停止</li>
        <li>切回程序点「保存」即可</li>
      </ol>
    </section>

    <!-- 校准 Modal -->
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
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { Events } from '@wailsio/runtime'
import { backend } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'

const settingsStore = useSettingsStore()
const settings = computed(() => settingsStore.data)

const manualCounts = ref<number>(0)
watch(
  () => settings.value?.ui.mouseCounts360,
  (v) => {
    manualCounts.value = v ?? 0
  },
  { immediate: true },
)

async function onCommitManual() {
  const v = Number(manualCounts.value)
  if (!Number.isFinite(v) || v < 0) return
  const cur = settings.value?.ui.mouseCounts360 ?? 0
  if (v === cur) return
  await settingsStore.patch({ ui: { mouseCounts360: Math.floor(v) } })
}

async function openMouseHUD() {
  await backend.tools.openMouseHUD()
}

type Stage = 'waiting' | 'countingDown' | 'accumulating' | 'done'

const open = ref(false)
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
  open.value = true
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
  if (!open.value) return
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
  open.value = false
}

async function onSave() {
  await teardown()
  const counts = liveAbsDx.value
  if (counts > 0) {
    await settingsStore.patch({ ui: { mouseCounts360: counts } })
  }
  open.value = false
}

function onUpdateOpen(v: boolean) {
  if (!v) onCancel()
}

onUnmounted(() => {
  void teardown()
})
</script>
