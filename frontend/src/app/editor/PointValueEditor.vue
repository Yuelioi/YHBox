<template>
  <div class="grid grid-cols-[1fr_1fr_92px] gap-2">
    <UInputNumber
      :model-value="point.x"
      :placeholder="t('workflow.inspector.point_x')"
      class="w-full"
      @update:model-value="update('x', Number($event))"
    />
    <UInputNumber
      :model-value="point.y"
      :placeholder="t('workflow.inspector.point_y')"
      class="w-full"
      @update:model-value="update('y', Number($event))"
    />
    <USelect
      :model-value="point.unit"
      :items="units"
      class="w-full"
      @update:model-value="update('unit', $event === 'px' ? 'px' : 'ratio')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

type PointValue = { x: number; y: number; unit: 'ratio' | 'px' }

const props = defineProps<{ modelValue: unknown }>()
const emit = defineEmits<{ 'update:model-value': [value: PointValue] }>()
const { t } = useI18n()

const point = computed<PointValue>(() => {
  if (!props.modelValue || typeof props.modelValue !== 'object') {
    return { x: 0, y: 0, unit: 'ratio' }
  }
  const value = props.modelValue as Record<string, unknown>
  return {
    x: typeof value.x === 'number' ? value.x : 0,
    y: typeof value.y === 'number' ? value.y : 0,
    unit: value.unit === 'px' ? 'px' : 'ratio',
  }
})

const units = computed(() => [
  { label: t('workflow.inspector.point_ratio'), value: 'ratio' },
  { label: t('workflow.inspector.point_px'), value: 'px' },
])

function update<Key extends keyof PointValue>(key: Key, value: PointValue[Key]): void {
  emit('update:model-value', { ...point.value, [key]: value })
}
</script>
