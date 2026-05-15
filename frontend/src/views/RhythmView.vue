<template>
  <div class="px-8 py-6 space-y-6">
    <div class="flex items-center justify-between gap-4">
      <BotControls
        :state="rhythmStore.state"
        :can-start="canStart"
        :disabled-reason="disabledReason"
        @start="onStart"
        @pause="onPause"
        @resume="onResume"
        @stop="onStop"
      />
      <div v-if="rhythmStore.state !== 'idle'" class="text-sm text-muted">
        <span class="text-dimmed mr-2">已运行</span>
        <span class="tabular-nums text-default font-medium">{{ runtimeStr }}</span>
      </div>
    </div>

    <div class="rounded-xl bg-default border border-default p-5 space-y-2">
      <h3 class="text-sm font-medium text-highlighted">使用说明</h3>
      <p class="text-xs text-dimmed">进入"超强音"曲目后再点 [开始]，自动识别 4 轨命中按 D/F/J/K</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRhythmStore } from '@/stores/rhythm'
import { useGameStore } from '@/stores/game'
import BotControls from '@/components/BotControls.vue'

const rhythmStore = useRhythmStore()
const gameStore = useGameStore()

const canStart = computed(() => rhythmStore.state === 'idle' && !!gameStore.status?.ok)

const disabledReason = computed(() => {
  if (!gameStore.status?.ok) return '未检测到异环窗口'
  if (rhythmStore.state !== 'idle') return '已在运行'
  return ''
})

// now 是 ticker 驱动的 reactive 时间戳；startedAt 来自 store（跨导航/reload 保留）
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
  if (rhythmStore.state === 'idle' || !rhythmStore.startedAt) return '—'
  const sec = Math.max(0, Math.floor((now.value - rhythmStore.startedAt) / 1000))
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  return `${h}h ${String(m).padStart(2, '0')}m ${String(s).padStart(2, '0')}s`
})

async function onStart() {
  await rhythmStore.start()
}
async function onPause() {
  await rhythmStore.pause()
}
async function onResume() {
  await rhythmStore.resume()
}
async function onStop() {
  await rhythmStore.stop()
}
</script>
