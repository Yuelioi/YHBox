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
            region.unit === candidate.value
              ? 'bg-primary/10 text-primary ring-1 ring-inset ring-primary/25'
              : 'text-muted hover:bg-elevated hover:text-default'
          "
          @click="void setUnit(candidate.value)"
        >
          {{ candidate.label }}
        </button>
      </div>
      <UButton
        icon="i-tabler-crop"
        :label="compact ? undefined : t('workflow.inspector.pick_region')"
        color="primary"
        variant="soft"
        size="xs"
        :disabled="!targetSlot"
        :loading="picking"
        @click="pickRegion"
      />
    </div>
    <div class="grid grid-cols-2 gap-2">
      <UFormField v-for="field in fields" :key="field.key" :label="field.label">
        <UInputNumber
          :model-value="displayValue(field.key)"
          :min="0"
          :max="region.unit === 'ratio' ? 100 : undefined"
          :step="region.unit === 'ratio' ? 0.1 : 1"
          :size="compact ? 'xs' : 'sm'"
          class="w-full"
          @update:model-value="update(field.key, Number($event))"
        />
      </UFormField>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorMessage } from '@/lib/invoke'
import { useToast } from '@/composables/useAppToast'
import { pickTargetValue, targetDimensions, type TargetRegion } from './useTargetPicker'
import {
  regionValueFromTarget,
  type CoordinateUnit as Unit,
  type RegionValue,
} from './targetValueMapping'

type RegionField = 'x' | 'y' | 'width' | 'height'

const props = defineProps<{ modelValue: unknown; targetSlot?: string; compact?: boolean }>()
const emit = defineEmits<{ 'update:model-value': [value: RegionValue] }>()
const { t } = useI18n()
const toast = useToast()
const picking = ref(false)
const region = computed(() => normalize(props.modelValue))
const targetSlot = computed(() => props.targetSlot ?? '')
const fields = computed<Array<{ key: RegionField; label: string }>>(() => [
  { key: 'x', label: 'X' },
  { key: 'y', label: 'Y' },
  { key: 'width', label: t('workflow.inspector.region_width') },
  { key: 'height', label: t('workflow.inspector.region_height') },
])
const units: Array<{ label: string; value: Unit }> = [
  { label: '%', value: 'ratio' },
  { label: 'px', value: 'px' },
]

function displayValue(field: RegionField): number {
  return region.value[field] * (region.value.unit === 'ratio' ? 100 : 1)
}

function update(field: RegionField, display: number): void {
  if (!Number.isFinite(display)) return
  emit('update:model-value', {
    ...region.value,
    [field]: Math.max(0, display) / (region.value.unit === 'ratio' ? 100 : 1),
  })
}

async function setUnit(unit: Unit): Promise<void> {
  if (unit === region.value.unit) return
  try {
    const dimensions = await targetDimensions(targetSlot.value)
    emit(
      'update:model-value',
      region.value.unit === 'ratio'
        ? {
            x: Math.round(region.value.x * dimensions.width),
            y: Math.round(region.value.y * dimensions.height),
            width: Math.round(region.value.width * dimensions.width),
            height: Math.round(region.value.height * dimensions.height),
            unit: 'px',
          }
        : {
            x: region.value.x / dimensions.width,
            y: region.value.y / dimensions.height,
            width: region.value.width / dimensions.width,
            height: region.value.height / dimensions.height,
            unit: 'ratio',
          },
    )
  } catch (error) {
    showPickerError(error)
  }
}

async function pickRegion(): Promise<void> {
  if (!targetSlot.value || picking.value) return
  picking.value = true
  try {
    const picked = await pickTargetValue<TargetRegion>('rect', targetSlot.value)
    if (!picked) return
    emit('update:model-value', regionValueFromTarget(picked, region.value.unit))
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

function normalize(value: unknown): RegionValue {
  const candidate = value && typeof value === 'object' ? (value as Partial<RegionValue>) : {}
  return {
    x: finite(candidate.x),
    y: finite(candidate.y),
    width: finite(candidate.width, 1),
    height: finite(candidate.height, 1),
    unit: candidate.unit === 'px' ? 'px' : 'ratio',
  }
}

function finite(value: unknown, fallback = 0): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}
</script>
