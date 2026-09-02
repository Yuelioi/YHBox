<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between gap-2">
      <div class="inline-flex rounded-md bg-muted p-0.5 text-[10px]">
        <button
          v-for="candidate in units"
          :key="candidate.value"
          type="button"
          class="rounded px-2 py-1 transition-colors"
          :class="
            point.unit === candidate.value
              ? 'bg-primary/10 text-primary ring-1 ring-inset ring-primary/25'
              : 'text-muted hover:bg-elevated hover:text-default'
          "
          @click="void setUnit(candidate.value)"
        >
          {{ candidate.label }}
        </button>
      </div>
      <UButton
        icon="i-tabler-pointer"
        :label="compact ? undefined : t('workflow.inspector.pick_point')"
        color="primary"
        variant="soft"
        size="xs"
        :disabled="!targetSlot"
        :loading="picking"
        @click="pickPoint"
      />
    </div>
    <div class="grid grid-cols-2 gap-2">
      <UFormField :label="`${t('workflow.inspector.point_x')} ${unitLabel}`">
        <UInputNumber
          :model-value="displayX"
          :min="0"
          :max="point.unit === 'ratio' ? 100 : undefined"
          :step="point.unit === 'ratio' ? 0.1 : 1"
          class="w-full"
          @update:model-value="update('x', Number($event))"
        />
      </UFormField>
      <UFormField :label="`${t('workflow.inspector.point_y')} ${unitLabel}`">
        <UInputNumber
          :model-value="displayY"
          :min="0"
          :max="point.unit === 'ratio' ? 100 : undefined"
          :step="point.unit === 'ratio' ? 0.1 : 1"
          class="w-full"
          @update:model-value="update('y', Number($event))"
        />
      </UFormField>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage } from '@/lib/invoke'
import { useToast } from '@nuxt/ui/composables'
import { pickTargetValue, targetDimensions, type TargetPoint } from './useTargetPicker'
import {
  pointValueFromTarget,
  type CoordinateUnit as Unit,
  type PointValue,
} from './targetValueMapping'

const props = defineProps<{ modelValue: unknown; targetSlot?: string; compact?: boolean }>()
const emit = defineEmits<{ 'update:model-value': [value: PointValue] }>()
const { t } = useI18n()
const toast = useToast()
const picking = ref(false)
const targetSlot = computed(() => props.targetSlot ?? '')
const point = computed<PointValue>(() => normalize(props.modelValue))
const displayX = computed(() => point.value.x * (point.value.unit === 'ratio' ? 100 : 1))
const displayY = computed(() => point.value.y * (point.value.unit === 'ratio' ? 100 : 1))
const unitLabel = computed(() => (point.value.unit === 'ratio' ? '%' : 'px'))
const units: Array<{ label: string; value: Unit }> = [
  { label: '%', value: 'ratio' },
  { label: 'px', value: 'px' },
]

function update(key: 'x' | 'y', display: number): void {
  if (!Number.isFinite(display)) return
  emit('update:model-value', {
    ...point.value,
    [key]: Math.max(0, display) / (point.value.unit === 'ratio' ? 100 : 1),
  })
}

async function setUnit(unit: Unit): Promise<void> {
  if (unit === point.value.unit) return
  if (point.value.x === 0 && point.value.y === 0) {
    emit('update:model-value', { ...point.value, unit })
    return
  }
  try {
    const dimensions = await targetDimensions(targetSlot.value)
    emit(
      'update:model-value',
      point.value.unit === 'ratio'
        ? {
            x: Math.round(point.value.x * dimensions.width),
            y: Math.round(point.value.y * dimensions.height),
            unit: 'px',
          }
        : {
            x: point.value.x / dimensions.width,
            y: point.value.y / dimensions.height,
            unit: 'ratio',
          },
    )
  } catch (error) {
    showPickerError(error)
  }
}

async function pickPoint(): Promise<void> {
  if (!targetSlot.value || picking.value) return
  picking.value = true
  try {
    const picked = await pickTargetValue<TargetPoint>('point', targetSlot.value)
    if (!picked) return
    emit('update:model-value', pointValueFromTarget(picked, point.value.unit))
  } catch (error) {
    showPickerError(error)
  } finally {
    picking.value = false
  }
}

function showPickerError(error: unknown): void {
  toast.add({
    title: t('workflow.inspector.pick_failed'),
    description: errorMessage(error),
    color: 'error',
  })
}

function normalize(value: unknown): PointValue {
  const candidate = value && typeof value === 'object' ? (value as Partial<PointValue>) : {}
  return {
    x: finite(candidate.x),
    y: finite(candidate.y),
    unit: candidate.unit === 'px' ? 'px' : 'ratio',
  }
}

function finite(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}
</script>
