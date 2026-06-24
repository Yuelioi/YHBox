<template>
  <div class="space-y-2">
    <div class="flex items-center gap-1.5">
      <span class="text-[10px] text-dimmed flex-1">{{ isPx ? 'px' : '%' }}</span>
      <div data-testid="point-unit-toggle" class="flex gap-0.5">
        <button
          class="text-[10px] px-1.5 py-0.5 rounded"
          :class="!isPx ? 'bg-primary text-white' : 'text-dimmed hover:bg-elevated'"
          @click="setUnit('percent')"
        >{{ t('point_widget.unit_percent') }}</button>
        <button
          class="text-[10px] px-1.5 py-0.5 rounded"
          :class="isPx ? 'bg-primary text-white' : 'text-dimmed hover:bg-elevated'"
          @click="setUnit('px')"
        >{{ t('point_widget.unit_px') }}</button>
      </div>
    </div>
    <p class="text-[10px] text-dimmed leading-snug">{{ isPx ? t('point_widget.hint_px') : t('point_widget.hint_percent') }}</p>
    <div class="grid grid-cols-2 gap-1.5">
      <div class="space-y-0.5">
        <label class="text-[10px] text-dimmed">X {{ unitLabel }}</label>
        <UInputNumber
          :model-value="displayX"
          size="xs"
          class="w-full"
          :min="0"
          :max="isPx ? undefined : 100"
          :step="isPx ? 1 : 0.1"
          @update:model-value="(v: number) => onChange('x', v)"
        />
      </div>
      <div class="space-y-0.5">
        <label class="text-[10px] text-dimmed">Y {{ unitLabel }}</label>
        <UInputNumber
          :model-value="displayY"
          size="xs"
          class="w-full"
          :min="0"
          :max="isPx ? undefined : 100"
          :step="isPx ? 1 : 0.1"
          @update:model-value="(v: number) => onChange('y', v)"
        />
      </div>
    </div>
    <UButton
      size="xs" variant="soft" color="primary" icon="i-tabler-pointer"
      data-testid="point-pick-btn" :loading="picking"
      @click="onPickPoint"
    >
      {{ t('point_widget.pick_point') }}
    </UButton>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import type { PointValue } from '@/components/containers/nodeRegistry/index'
import { backend } from '@/lib/backend'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { useTemplatesStore } from '@/stores/templates'

const { t } = useI18n()
const toast = useToast()

const props = defineProps<{
  modelValue: PointValue | null
  fieldPath: string
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: PointValue): void }>()

function round4(n: number): number {
  return Math.round(n * 1e4) / 1e4
}

const safeValue = computed<PointValue>(() => {
  const v = props.modelValue
  if (!v || typeof v.x !== 'number' || typeof v.y !== 'number') return { x: 0, y: 0 }
  return { x: v.x, y: v.y, unit: v.unit }
})

const isPx = computed(() => safeValue.value.unit === 'px')
const unitLabel = computed(() => (isPx.value ? 'px' : '%'))

// 显示: px 原值; % ×100
const displayX = computed(() => (isPx.value ? safeValue.value.x : round4(safeValue.value.x * 100)))
const displayY = computed(() => (isPx.value ? safeValue.value.y : round4(safeValue.value.y * 100)))

function onChange(field: 'x' | 'y', displayVal: number) {
  const next: PointValue = { ...safeValue.value }
  next[field] = isPx.value ? displayVal : round4(displayVal / 100)
  emit('update:modelValue', next)
}

// 切单位: 调 mousePos 换算; 无窗口时保留框里数字并弹提示
async function setUnit(u: 'percent' | 'px') {
  const targetPx = u === 'px'
  if (targetPx === isPx.value) return // already that unit, no-op
  const cur = safeValue.value
  const info = await backend.tools.mousePos(tplStore.containerId, '')
  const hasSize = !!info?.hasGame && info.clientW > 0 && info.clientH > 0
  const next: PointValue = { x: cur.x, y: cur.y }
  if (targetPx) {
    next.unit = 'px'
    if (hasSize) {
      next.x = Math.round(cur.x * info.clientW) // ratio(0-1) → px
      next.y = Math.round(cur.y * info.clientH)
    } else {
      next.x = displayX.value // keep box number (= cur.x*100), no conversion
      next.y = displayY.value
      notifyNoSize()
    }
  } else {
    if (hasSize) {
      next.x = round4(cur.x / info.clientW) // px → ratio
      next.y = round4(cur.y / info.clientH)
    } else {
      next.x = round4(displayX.value / 100) // keep box number (= cur.x original px) as % display number
      next.y = round4(displayY.value / 100)
      notifyNoSize()
    }
  }
  emit('update:modelValue', next)
}

function notifyNoSize() {
  toast.add({
    title: t('point_widget.no_window_title'),
    description: t('point_widget.no_window_desc'),
    color: 'warning',
  })
}

// ─── 截图取点 ────────────────────────────────────────────────────────────────
type PointPayload = { xRatio: number; yRatio: number; screenW?: number; screenH?: number; cancelled?: boolean }

const tplStore = useTemplatesStore()
const picking = ref(false)

function genID(): string {
  return 'pick-pt-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
}

async function onPickPoint() {
  if (picking.value) return
  const id = genID()
  picking.value = true
  try {
    const waiter = awaitWailsEvent<{ id: string; payload: PointPayload }>('tools:picker-result', (p) => p?.id === id)
    const r = await backend.tools.openScreenPicker('point', id, tplStore.containerId)
    if (r === undefined) return
    const res = await waiter
    const p = res.payload
    if (!p || p.cancelled) return
    const next: PointValue = { ...safeValue.value }
    if (isPx.value && p.screenW && p.screenH) {
      next.unit = 'px'
      next.x = Math.round(p.xRatio * p.screenW)
      next.y = Math.round(p.yRatio * p.screenH)
    } else {
      delete next.unit
      next.x = round4(p.xRatio)
      next.y = round4(p.yRatio)
    }
    emitVal(next)
  } finally {
    picking.value = false
  }
}

function emitVal(v: PointValue) {
  emit('update:modelValue', v)
}
</script>
