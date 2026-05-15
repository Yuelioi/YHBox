<template>
  <div class="px-8 py-6 space-y-6">
    <!-- Top row: 主操作按钮放右（统一布局） -->
    <div class="flex items-center justify-end gap-4">
      <BotControls
        :state="cookStore.state"
        :can-start="canStart"
        :disabled-reason="disabledReason"
        @start="cookStore.start()"
        @pause="cookStore.pause()"
        @resume="cookStore.resume()"
        @stop="cookStore.stop()"
      />
    </div>

    <!-- Config row: interval input -->
    <div class="rounded-xl bg-default border border-default p-5">
      <div class="flex items-center justify-between gap-6">
        <div>
          <h3 class="text-sm font-medium text-highlighted">点击间隔</h3>
          <p class="text-xs text-dimmed mt-1">每次锤子点击之间的等待时间。100 – 10000 ms。</p>
        </div>
        <div class="flex items-center gap-3">
          <UInputNumber
            v-model="intervalMs"
            :min="100"
            :max="10000"
            :step="100"
            class="w-32"
            @update:model-value="onIntervalCommit"
          />
          <span class="text-xs text-dimmed">ms</span>
        </div>
      </div>
    </div>

    <!-- Info card -->
    <div class="rounded-xl bg-default/50 border border-default/60 p-5">
      <div class="flex items-start gap-3">
        <UIcon name="i-tabler-info-circle" class="size-4 text-dimmed mt-0.5 shrink-0" />
        <div class="text-xs text-muted leading-relaxed">
          自动识别锤子图标位置并连点。本功能依赖前台截屏，运行时游戏窗口必须保持前台（最小化或失焦会停）。
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useCookStore } from '@/stores/cook'
import { useGameStore } from '@/stores/game'
import { useSettingsStore } from '@/stores/settings'
import BotControls from '@/components/BotControls.vue'

const cookStore = useCookStore()
const gameStore = useGameStore()
const settingsStore = useSettingsStore()

const intervalMs = ref<number>(settingsStore.data?.cook.intervalMs ?? 1500)

watch(
  () => settingsStore.data?.cook.intervalMs,
  (v) => {
    if (v !== undefined && v !== intervalMs.value) intervalMs.value = v
  },
)

const canStart = computed(() => cookStore.state === 'idle' && !!gameStore.status?.ok)
const disabledReason = computed(() => {
  if (!gameStore.status?.ok) return '未检测到异环窗口'
  if (cookStore.state !== 'idle') return '已在运行'
  return ''
})

async function onIntervalCommit(v: number) {
  const r = await cookStore.setIntervalMs(v)
  if (r === undefined) return
  if (settingsStore.data) settingsStore.data.cook.intervalMs = v
}
</script>
