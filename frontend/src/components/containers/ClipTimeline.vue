<template>
  <div class="select-none">
    <div class="flex items-center justify-between text-[10px] text-dimmed mb-1">
      <span>0 ms</span>
      <span>{{ formatMs(durationMs) }}</span>
    </div>
    <div
      ref="trackRef"
      class="relative h-7 rounded-md bg-zinc-900 border border-default/60 cursor-crosshair overflow-hidden"
      @mousedown="onTrackMouseDown"
      @mousemove="onTrackMouseMove"
      @mouseleave="onTrackMouseLeave"
    >
      <!-- 整段背景 (灰 = 默认全播) -->
      <div v-if="ranges.length === 0" class="absolute inset-0 bg-blue-500/15" />

      <!-- 已选保留段 (蓝色高亮) -->
      <div
        v-for="(r, idx) in ranges"
        :key="idx"
        class="absolute top-0 bottom-0 bg-blue-500/35 border-x border-blue-400/80"
        :style="{ left: pct(r.fromMs) + '%', width: pct(r.toMs - r.fromMs) + '%' }"
        @mouseenter="hoverIdx = idx"
        @mouseleave="hoverIdx = null"
      >
        <!-- 拖把 (左/右) -->
        <div
          class="absolute -left-1 top-0 bottom-0 w-1.5 bg-blue-400 cursor-ew-resize"
          @mousedown.stop="onHandleMouseDown(idx, 'from', $event)"
        />
        <div
          class="absolute -right-1 top-0 bottom-0 w-1.5 bg-blue-400 cursor-ew-resize"
          @mousedown.stop="onHandleMouseDown(idx, 'to', $event)"
        />
        <!-- 删除小 X -->
        <button
          v-if="hoverIdx === idx"
          class="absolute top-0.5 right-1 size-3 rounded-full bg-error text-white text-[8px] flex items-center justify-center"
          @click.stop="$emit('remove', idx)"
        >×</button>
      </div>

      <!-- hover 提示 -->
      <div
        v-if="hoverMs !== null"
        class="absolute top-0 bottom-0 border-l border-primary/60 pointer-events-none"
        :style="{ left: pct(hoverMs) + '%' }"
      />
    </div>
    <div class="text-[10px] text-dimmed mt-1">
      {{ t('clipTimeline.hint') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface Range { fromMs: number; toMs: number }

const props = defineProps<{
  durationMs: number
  ranges: Range[]
}>()
const emit = defineEmits<{
  add: [range: Range]
  update: [idx: number, range: Range]
  remove: [idx: number]
}>()

const trackRef = ref<HTMLDivElement | null>(null)
const hoverMs = ref<number | null>(null)
const hoverIdx = ref<number | null>(null)

function pct(ms: number): number {
  if (props.durationMs <= 0) return 0
  return Math.max(0, Math.min(100, (ms / props.durationMs) * 100))
}

function clientXToMs(clientX: number): number {
  if (!trackRef.value || props.durationMs <= 0) return 0
  const rc = trackRef.value.getBoundingClientRect()
  const ratio = (clientX - rc.left) / rc.width
  return Math.round(Math.max(0, Math.min(1, ratio)) * props.durationMs)
}

function onTrackMouseMove(e: MouseEvent) {
  hoverMs.value = clientXToMs(e.clientX)
}

function onTrackMouseLeave() {
  if (!dragAdd) hoverMs.value = null
}

// 在空轨道拖 → 添加新段
let dragAdd: { start: number } | null = null
function onTrackMouseDown(e: MouseEvent) {
  // 不在已有 range 上才能新加
  const ms = clientXToMs(e.clientX)
  const insideExisting = props.ranges.some((r) => ms >= r.fromMs && ms <= r.toMs)
  if (insideExisting) return
  dragAdd = { start: ms }
  const onMove = (mv: MouseEvent) => {
    hoverMs.value = clientXToMs(mv.clientX)
  }
  const onUp = (uv: MouseEvent) => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
    if (!dragAdd) return
    const end = clientXToMs(uv.clientX)
    const fromMs = Math.min(dragAdd.start, end)
    const toMs = Math.max(dragAdd.start, end)
    if (toMs - fromMs > 50) emit('add', { fromMs, toMs })
    dragAdd = null
    hoverMs.value = null
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

// 拖把: from / to edge
function onHandleMouseDown(idx: number, edge: 'from' | 'to', e: MouseEvent) {
  e.preventDefault()
  const orig = { ...props.ranges[idx] }
  const onMove = (mv: MouseEvent) => {
    const ms = clientXToMs(mv.clientX)
    const next: Range = { ...orig }
    if (edge === 'from') next.fromMs = Math.min(ms, orig.toMs - 50)
    else next.toMs = Math.max(ms, orig.fromMs + 50)
    emit('update', idx, next)
  }
  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

function formatMs(ms: number): string {
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(1)} s`
}
</script>
