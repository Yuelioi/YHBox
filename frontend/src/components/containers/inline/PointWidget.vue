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
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PointValue } from '@/components/containers/nodeRegistry/index'

const { t } = useI18n()

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

// 切单位: 保留框里数字不换算 → 数据层 x/y 随之改
function setUnit(u: 'percent' | 'px') {
  const next: PointValue = { ...safeValue.value }
  const curDisplayX = displayX.value
  const curDisplayY = displayY.value
  if (u === 'px') {
    next.unit = 'px'
    next.x = curDisplayX // 框里数字原样进 px
    next.y = curDisplayY
  } else {
    delete next.unit
    next.x = round4(curDisplayX / 100) // 框里数字回比例
    next.y = round4(curDisplayY / 100)
  }
  emit('update:modelValue', next)
}
</script>
