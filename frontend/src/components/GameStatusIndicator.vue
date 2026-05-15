<template>
  <!-- Compact (collapsed sidebar): refresh icon + 角标 status dot -->
  <button
    v-if="compact"
    type="button"
    class="h-10 shrink-0 flex items-center justify-center border-t border-default text-muted hover:text-highlighted hover:bg-elevated/40 transition-colors duration-150 group"
    :title="label + '（点击重新检测）'"
    :disabled="detecting"
    @click="handleDetect"
  >
    <span class="relative inline-flex items-center justify-center">
      <UIcon
        name="i-tabler-refresh"
        class="size-4 transition-transform duration-500"
        :class="{ 'animate-spin': detecting }"
      />
      <!-- 状态点叠在 icon 右上角，类似通知 badge -->
      <span
        class="absolute -top-0.5 -right-1 size-1.5 rounded-full ring-2 ring-default transition-colors duration-300"
        :class="dotClass"
      />
    </span>
  </button>

  <!-- Expanded: full label + refresh button -->
  <div v-else class="px-3 py-3 border-t border-default shrink-0">
    <div class="flex items-center gap-2 min-w-0">
      <span class="size-2 rounded-full shrink-0 transition-colors duration-300" :class="dotClass" />
      <span class="text-xs text-muted truncate flex-1 min-w-0" :title="label">{{ label }}</span>
      <button
        class="shrink-0 flex items-center justify-center size-5 rounded text-dimmed hover:text-toned hover:bg-elevated transition-colors duration-150"
        title="重新检测"
        @click="handleDetect"
      >
        <UIcon
          name="i-tabler-refresh"
          class="size-3 transition-transform duration-500"
          :class="{ 'animate-spin': detecting }"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useGameStore } from '@/stores/game'

withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })

const gameStore = useGameStore()
const detecting = ref(false)

const label = computed(() => {
  const s = gameStore.status
  if (!s) return '正在检测...'
  if (!s.ok) return '未检测到异环'
  return `${s.title}  ${s.w}×${s.h}`
})

const dotClass = computed(() => {
  const s = gameStore.status
  if (!s) return 'bg-accented'
  if (!s.ok) return 'bg-error'
  return 'bg-primary'
})

async function handleDetect() {
  if (detecting.value) return
  detecting.value = true
  try {
    await gameStore.detect()
  } finally {
    setTimeout(() => {
      detecting.value = false
    }, 600)
  }
}
</script>
