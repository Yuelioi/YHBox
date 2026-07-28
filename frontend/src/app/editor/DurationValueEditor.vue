<template>
  <div
    class="grid"
    :class="
      compact ? 'grid-cols-[minmax(0,1fr)_80px] gap-1.5' : 'grid-cols-[minmax(0,1fr)_92px] gap-2'
    "
  >
    <UInputNumber
      :model-value="displayValue"
      :min="0"
      :step="unit === 'ms' ? 10 : 0.1"
      :size="compact ? 'xs' : 'sm'"
      class="w-full"
      @update:model-value="updateValue(Number($event))"
    />
    <AdaptiveSelect
      :model-value="unit"
      :items="units"
      :size="compact ? 'xs' : 'sm'"
      width-mode="fill"
      @update:model-value="setUnit($event === 'min' ? 'min' : $event === 's' ? 's' : 'ms')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'

type DurationUnit = 'ms' | 's' | 'min'
const props = withDefaults(defineProps<{ modelValue: unknown; compact?: boolean }>(), {
  compact: false,
})
const emit = defineEmits<{ 'update:model-value': [value: number] }>()
const { t } = useI18n()
const unit = ref<DurationUnit>(suggestUnit(props.modelValue))
const units = computed(() => [
  { label: t('workflow.inspector.duration_ms'), value: 'ms' },
  { label: t('workflow.inspector.duration_s'), value: 's' },
  { label: t('workflow.inspector.duration_min'), value: 'min' },
])
const milliseconds = computed(() =>
  typeof props.modelValue === 'number' && Number.isFinite(props.modelValue)
    ? Math.max(0, props.modelValue)
    : 0,
)
const displayValue = computed(() => milliseconds.value / multiplier(unit.value))

function updateValue(value: number): void {
  if (!Number.isFinite(value)) return
  emit('update:model-value', Math.round(Math.max(0, value) * multiplier(unit.value)))
}

function setUnit(value: DurationUnit): void {
  unit.value = value
}

function multiplier(value: DurationUnit): number {
  return value === 'min' ? 60_000 : value === 's' ? 1_000 : 1
}

function suggestUnit(value: unknown): DurationUnit {
  if (typeof value !== 'number' || value < 1_000) return 'ms'
  return value >= 60_000 && value % 60_000 === 0 ? 'min' : 's'
}
</script>
