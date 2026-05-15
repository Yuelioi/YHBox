<template>
  <div class="px-8 py-6 space-y-6">
    <!-- Top row: run-time 在左，主操作按钮在右（统一布局） -->
    <div class="flex items-center justify-between gap-4">
      <div v-if="fishStore.state !== 'idle'" class="text-sm text-muted">
        <span class="text-dimmed mr-2">已运行</span>
        <span class="tabular-nums text-default font-medium">{{ runtimeStr }}</span>
      </div>
      <div v-else />
      <BotControls
        :state="fishStore.state"
        :can-start="canStart"
        :disabled-reason="disabledReason"
        @start="onStart"
        @pause="onPause"
        @resume="onResume"
        @stop="onStop"
      />
    </div>

    <!-- Config row: AutoSell switch -->
    <div class="rounded-xl bg-default border border-default p-5">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-sm font-medium text-highlighted">自动出售(建议开启)</h3>
          <p class="text-xs text-dimmed mt-1">鱼仓满(1000条)后自动出售</p>
          <p class="text-xs text-dimmed mt-1">没鱼饵会自动出售换钱购买</p>
          <p class="text-xs text-dimmed mt-1">注意: 会消耗都市体力</p>
        </div>
        <USwitch v-model="autoSell" @update:model-value="onAutoSellChange" />
      </div>
    </div>

    <!-- Config row: CaptureGolden switch -->
    <div class="rounded-xl bg-default border border-default p-5">
      <div class="flex items-center justify-between">
        <div>
          <h3 class="text-sm font-medium text-highlighted">金色鱼截图</h3>
          <p class="text-xs text-dimmed mt-1">钓上金色鱼时把结算画面存成 PNG</p>
          <p class="text-xs text-dimmed mt-1">
            保存到 exe 同目录的
            <code class="text-toned bg-elevated/60 px-1 py-0.5 rounded">captures/</code>
          </p>
        </div>
        <USwitch v-model="captureGolden" @update:model-value="onCaptureGoldenChange" />
      </div>
    </div>

    <!-- Stats grid -->
    <div>
      <h3 class="text-xs uppercase tracking-wider text-dimmed mb-3">本次统计</h3>
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <StatCard label="总数" :value="total" icon="i-tabler-hash" />
        <StatCard
          label="普通"
          :value="fishStore.stats.commonCount"
          color="sky"
          icon="i-tabler-fish"
        />
        <StatCard
          label="紫色"
          :value="fishStore.stats.purpleCount"
          color="violet"
          icon="i-tabler-fish"
        />
        <StatCard
          label="金色"
          :value="fishStore.stats.goldenCount"
          color="amber"
          icon="i-tabler-fish"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useToast } from '@nuxt/ui/composables'
import { useFishStore } from '@/stores/fish'
import { useGameStore } from '@/stores/game'
import { useSettingsStore } from '@/stores/settings'
import { backend } from '@/lib/backend'
import BotControls from '@/components/BotControls.vue'
import StatCard from '@/components/StatCard.vue'

const fishStore = useFishStore()
const gameStore = useGameStore()
const settingsStore = useSettingsStore()
const toast = useToast()

// AutoSell hydration: settings load → ref → switch
const autoSell = ref<boolean>(settingsStore.data?.fish.autoSell ?? true)
const captureGolden = ref<boolean>(settingsStore.data?.fish.captureGolden ?? false)

// Keep refs in sync if settings are patched elsewhere
const stopAutoSellWatch = watch(
  () => settingsStore.data?.fish.autoSell,
  (v) => {
    if (v !== undefined && v !== autoSell.value) {
      autoSell.value = v
    }
  },
)
const stopCaptureWatch = watch(
  () => settingsStore.data?.fish.captureGolden,
  (v) => {
    if (v !== undefined && v !== captureGolden.value) {
      captureGolden.value = v
    }
  },
)
onUnmounted(() => {
  stopAutoSellWatch()
  stopCaptureWatch()
})

const total = computed(
  () => fishStore.stats.commonCount + fishStore.stats.purpleCount + fishStore.stats.goldenCount,
)

const canStart = computed(() => fishStore.state === 'idle' && !!gameStore.status?.ok)

const disabledReason = computed(() => {
  if (!gameStore.status?.ok) return '未检测到异环窗口'
  if (fishStore.state !== 'idle') return '已在运行'
  return ''
})

// Runtime display (recompute every second while running)
const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  timer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
})
onUnmounted(() => {
  if (timer != null) clearInterval(timer)
})

const runtimeStr = computed(() => {
  const iso = fishStore.stats.startedAt
  if (!iso) return '—'
  const start = new Date(iso as unknown as string).getTime()
  if (!Number.isFinite(start) || start <= 0) return '—'
  const sec = Math.max(0, Math.floor((now.value - start) / 1000))
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  return `${h}h ${String(m).padStart(2, '0')}m ${String(s).padStart(2, '0')}s`
})

async function onStart() {
  await fishStore.start()
}
async function onPause() {
  await fishStore.pause()
}
async function onResume() {
  await fishStore.resume()
}
async function onStop() {
  await fishStore.stop()
}

async function onAutoSellChange(v: boolean) {
  autoSell.value = v
  const r = await backend.fish.setAutoSell(v)
  if (r === undefined) return
  // Reflect in local settings store immediately
  if (settingsStore.data) {
    settingsStore.data.fish.autoSell = v
  }
  toast.add({
    title: v ? '已开启自动出售' : '已关闭自动出售',
    color: v ? 'success' : 'neutral',
    icon: v ? 'i-tabler-check' : 'i-tabler-info-circle',
  })
}

async function onCaptureGoldenChange(v: boolean) {
  captureGolden.value = v
  const r = await backend.fish.setCaptureGolden(v)
  if (r === undefined) return
  if (settingsStore.data) {
    settingsStore.data.fish.captureGolden = v
  }
  toast.add({
    title: v ? '已开启金色鱼截图' : '已关闭金色鱼截图',
    description: v ? '保存到 exe 同目录的 captures/' : undefined,
    color: v ? 'success' : 'neutral',
    icon: v ? 'i-tabler-camera' : 'i-tabler-info-circle',
  })
}
</script>
