<template>
  <div
    class="h-7 shrink-0 flex items-center justify-between px-4 border-t border-default bg-default text-[11px] text-muted select-none"
  >
    <!-- LEFT — active status -->
    <div class="flex items-center gap-3 min-w-0 flex-1">
      <span
        class="size-1.5 rounded-full shrink-0 transition-colors duration-300"
        :class="leftDotClass"
      />
      <span class="font-medium" :class="leftLabelClass">{{ activeStatus.label }}</span>
      <template v-if="activeStatus.metrics.length">
        <span class="text-dimmed">·</span>
        <span class="tabular-nums text-toned truncate">
          {{ activeStatus.metrics.join(' · ') }}
        </span>
      </template>
      <!-- 容器跑中 → 显示当前节点 + 一键停止按钮 -->
      <template v-if="activeStatus.kind === 'container'">
        <span v-if="currentNodeLabel" class="text-dimmed">·</span>
        <span v-if="currentNodeLabel" class="text-emerald-300 truncate">
          ▶ {{ currentNodeLabel }}
        </span>
        <button
          type="button"
          class="ml-2 px-2 py-0.5 rounded text-[10px] bg-error/15 border border-error/40 text-error hover:bg-error/25 transition-colors inline-flex items-center gap-1"
          title="停止当前运行 + 清队列 (Ctrl+Shift+F9)"
          @click="onStopAll"
        >
          <UIcon name="i-tabler-square" class="size-2.5" /> 停止
        </button>
      </template>
    </div>

    <!-- RIGHT — game window + log count -->
    <div class="flex items-center gap-4 shrink-0">
      <!-- Game window status + refresh -->
      <button
        type="button"
        class="flex items-center gap-1.5 hover:text-highlighted transition-colors duration-150"
        :title="gameTooltip + '（点击重新检测）'"
        :disabled="detecting"
        @click="onDetect"
      >
        <UIcon name="i-tabler-device-desktop" class="size-3" />
        <span
          class="size-1.5 rounded-full shrink-0 transition-colors duration-300"
          :class="gameDotClass"
        />
        <span :class="gameLabelClass">{{ gameLabel }}</span>
        <UIcon
          name="i-tabler-refresh"
          class="size-3 ml-0.5 text-dimmed"
          :class="{ 'animate-spin': detecting }"
        />
      </button>

      <!-- Log line count -->
      <div class="flex items-center gap-1.5" :title="`日志 ${logStore.lines.length}/500`">
        <UIcon name="i-tabler-terminal" class="size-3" />
        <span class="tabular-nums">{{ logStore.lines.length }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useGameStore } from '@/stores/game'
import { useLogStore } from '@/stores/log'
import { useExecutionStore } from '@/stores/execution'
import { useContainersStore } from '@/stores/containers'

const gameStore = useGameStore()
const logStore = useLogStore()
const execStore = useExecutionStore()
const containersStore = useContainersStore()

// 显示当前正在跑的节点（中文标签）
import { KIND_LABEL_ZH } from '@/components/containers/pinSpec'
const currentNodeLabel = computed(() => {
  if (!execStore.currentNodeKind) return ''
  return KIND_LABEL_ZH[execStore.currentNodeKind] ?? execStore.currentNodeKind
})

async function onStopAll() {
  await containersStore.stopAll()
}

type Active = {
  kind: 'container' | 'idle'
  state: 'idle' | 'running'
  label: string
  metrics: string[]
}

const activeStatus = computed<Active>(() => {
  if (execStore.running) {
    const cur = containersStore.list.find((c) => c.id === execStore.currentTargetID)
    const name = cur?.name || execStore.currentTargetID.slice(0, 8) || '容器'
    const metrics: string[] = []
    if (execStore.targets.length > 1) {
      metrics.push(`target ${execStore.targetIdx + 1}/${execStore.targets.length}`)
    }
    return { kind: 'container', state: 'running', label: `▶ 跑中: ${name}`, metrics }
  }
  return { kind: 'idle', state: 'idle', label: '空闲', metrics: [] }
})

const leftDotClass = computed(() => {
  switch (activeStatus.value.state) {
    case 'running':
      return 'bg-primary animate-pulse'
    default:
      return 'bg-accented'
  }
})

const leftLabelClass = computed(() => {
  switch (activeStatus.value.state) {
    case 'running':
      return 'text-primary'
    default:
      return 'text-dimmed'
  }
})

// Game window
const detecting = ref(false)
const gameLabel = computed(() => {
  const s = gameStore.status
  if (!s) return '检测中'
  if (!s.ok) return '未检测'
  return `${s.w}×${s.h}`
})
const gameLabelClass = computed(() => {
  const s = gameStore.status
  if (!s) return 'text-dimmed'
  if (!s.ok) return 'text-error'
  return 'text-primary'
})
const gameDotClass = computed(() => {
  const s = gameStore.status
  if (!s) return 'bg-accented'
  if (!s.ok) return 'bg-error'
  return 'bg-primary'
})
const gameTooltip = computed(() => {
  const s = gameStore.status
  if (!s) return '游戏窗口检测中'
  if (!s.ok) return '未检测到异环窗口'
  return `${s.title} (${s.w}×${s.h})`
})

async function onDetect() {
  if (detecting.value) return
  detecting.value = true
  try {
    await gameStore.detect()
  } finally {
    setTimeout(() => {
      detecting.value = false
    }, 400)
  }
}
</script>
