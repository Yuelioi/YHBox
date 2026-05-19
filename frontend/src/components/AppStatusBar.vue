<template>
  <div
    class="h-7 shrink-0 flex items-center justify-between px-4 border-t border-default bg-default text-[11px] text-muted select-none"
  >
    <!-- LEFT — active bot status (互斥，只显示一个) -->
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
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useFishStore } from '@/stores/fish'
import { useCookStore } from '@/stores/cook'
import { usePianoStore } from '@/stores/piano'
import { useBattleStore } from '@/stores/battle'
import { useGameStore } from '@/stores/game'
import { useLogStore } from '@/stores/log'
import { useExecutionStore } from '@/stores/execution'
import { useContainersStore } from '@/stores/containers'

const fishStore = useFishStore()
const cookStore = useCookStore()
const pianoStore = usePianoStore()
const battleStore = useBattleStore()
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

// 每秒刷一次时间让 startedAt 时长 reactive 更新
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

type Active = {
  kind: 'fish' | 'cook' | 'piano' | 'battle' | 'container' | 'idle'
  state: 'idle' | 'running' | 'paused' | 'enabled'
  label: string
  metrics: string[]
}

const activeStatus = computed<Active>(() => {
  // 容器跑中优先显示运行 metrics, 否则显示空闲态.
  if (execStore.running) {
    const cur = containersStore.list.find((c) => c.id === execStore.currentTargetID)
    const name = cur?.name || execStore.currentTargetID.slice(0, 8) || '容器'
    const metrics: string[] = []
    if (execStore.targets.length > 1) {
      metrics.push(`target ${execStore.targetIdx + 1}/${execStore.targets.length}`)
    }
    return { kind: 'container', state: 'running', label: `▶ 跑中: ${name}`, metrics }
  }
  if (fishStore.state !== 'idle') {
    const s = fishStore.state === 'paused' ? '已暂停' : '正在钓鱼'
    const metrics: string[] = []
    const rt = fmtRuntime(fishStore.stats.startedAt)
    if (rt) metrics.push(rt)
    const total =
      fishStore.stats.commonCount + fishStore.stats.purpleCount + fishStore.stats.goldenCount
    if (total > 0) {
      metrics.push(`普 ${fishStore.stats.commonCount}`)
      metrics.push(`紫 ${fishStore.stats.purpleCount}`)
      metrics.push(`金 ${fishStore.stats.goldenCount}`)
    }
    return { kind: 'fish', state: fishStore.state, label: s, metrics }
  }
  if (cookStore.state !== 'idle') {
    const s = cookStore.state === 'paused' ? '店长已暂停' : '正在连点'
    return { kind: 'cook', state: cookStore.state, label: s, metrics: [] }
  }
  if (pianoStore.state !== 'idle') {
    const s = pianoStore.state === 'paused' ? '弹琴已暂停' : '正在演奏'
    const cur = fmtMs(pianoStore.progress.curMs)
    const total = fmtMs(pianoStore.progress.totalMs)
    return { kind: 'piano', state: pianoStore.state, label: s, metrics: [`${cur} / ${total}`] }
  }
  if (battleStore.status.enabled) {
    return {
      kind: 'battle',
      state: 'enabled',
      label: '战斗热键启用',
      metrics: [`${battleStore.status.mods}+1~6`],
    }
  }
  return { kind: 'idle', state: 'idle', label: '空闲', metrics: [] }
})

const leftDotClass = computed(() => {
  switch (activeStatus.value.state) {
    case 'running':
    case 'enabled':
      return 'bg-primary animate-pulse'
    case 'paused':
      return 'bg-warning'
    default:
      return 'bg-accented'
  }
})

const leftLabelClass = computed(() => {
  switch (activeStatus.value.state) {
    case 'running':
    case 'enabled':
      return 'text-primary'
    case 'paused':
      return 'text-warning'
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

function fmtRuntime(iso: string): string {
  if (!iso) return ''
  const start = new Date(iso).getTime()
  if (!Number.isFinite(start) || start <= 0) return ''
  const sec = Math.max(0, Math.floor((now.value - start) / 1000))
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  if (h > 0) return `${h}h ${String(m).padStart(2, '0')}m`
  if (m > 0) return `${m}m ${String(s).padStart(2, '0')}s`
  return `${s}s`
}

function fmtMs(ms: number): string {
  if (ms < 0) ms = 0
  const totalSec = Math.floor(ms / 1000)
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}:${String(s).padStart(2, '0')}`
}
</script>
