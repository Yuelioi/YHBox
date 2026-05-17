<template>
  <!-- Inline (toolbar 内嵌): dot + 简短分辨率 + refresh icon -->
  <button
    v-if="variant === 'inline'"
    type="button"
    class="inline-flex items-center gap-1.5 h-7 px-2 rounded text-[11px] text-toned hover:bg-elevated/50 transition-colors"
    :title="label + '（点击重新检测）'"
    :disabled="detecting"
    @click="handleDetect"
  >
    <span class="size-1.5 rounded-full shrink-0 transition-colors" :class="dotClass" />
    <span class="truncate max-w-[160px]">{{ inlineLabel }}</span>
    <UIcon
      name="i-tabler-refresh"
      class="size-3 text-dimmed transition-transform duration-500"
      :class="{ 'animate-spin': detecting }"
    />
  </button>

  <!-- Compact (collapsed sidebar): refresh icon + 角标 status dot -->
  <button
    v-else-if="compact"
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
import { computed, onMounted, ref } from 'vue'
import { useGameStore } from '@/stores/game'

withDefaults(
  defineProps<{ compact?: boolean; variant?: 'sidebar' | 'inline' }>(),
  { compact: false, variant: 'sidebar' },
)

const gameStore = useGameStore()
const detecting = ref(false)

// 跨窗口 store 是独立实例 (独立编辑器窗口的 game store 是 null 初值, 不与主窗口共享).
// mount 时主动 detect 一次, 让 inline/sidebar 都立刻拿到当前游戏状态.
// 即便主窗口已经检测过, 子窗口也需要自己 detect (cheap windows API call).
onMounted(() => {
  if (gameStore.status === null) {
    void gameStore.detect()
  }
})

const label = computed(() => {
  const s = gameStore.status
  if (!s) return '正在检测...'
  if (!s.ok) return '未检测到异环'
  return `${s.title}  ${s.w}×${s.h}`
})

// inline 用：只显示分辨率（标题太长挤不下 toolbar）
const inlineLabel = computed(() => {
  const s = gameStore.status
  if (!s) return '检测中'
  if (!s.ok) return '未检测到'
  return `${s.w}×${s.h}`
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
