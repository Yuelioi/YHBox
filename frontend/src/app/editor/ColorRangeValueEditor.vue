<template>
  <div class="space-y-2">
    <AdaptiveSelect
      :model-value="range.space"
      :items="spaces"
      value-key="value"
      label-key="label"
      class="w-full"
      width-mode="fill"
      @update:model-value="setSpace($event === 'hsv' ? 'hsv' : 'rgb')"
    />
    <div class="grid grid-cols-3 gap-2">
      <div v-for="(channel, index) in channels" :key="channel" class="space-y-1">
        <p class="text-center text-[10px] font-medium text-muted">{{ channel }}</p>
        <UInputNumber
          :model-value="range.minimum[index]"
          :min="0"
          :max="limits[index]"
          :placeholder="t('workflow.inspector.color_minimum')"
          class="w-full"
          @update:model-value="setChannel('minimum', index, Number($event))"
        />
        <UInputNumber
          :model-value="range.maximum[index]"
          :min="0"
          :max="limits[index]"
          :placeholder="t('workflow.inspector.color_maximum')"
          class="w-full"
          @update:model-value="setChannel('maximum', index, Number($event))"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'

type ColorSpace = 'rgb' | 'hsv'
type ColorRange = {
  space: ColorSpace
  minimum: [number, number, number]
  maximum: [number, number, number]
}

const props = defineProps<{ modelValue: unknown }>()
const emit = defineEmits<{ 'update:model-value': [value: ColorRange] }>()
const { t } = useI18n()

const range = computed<ColorRange>(() => normalize(props.modelValue))
const channels = computed(() =>
  range.value.space === 'hsv'
    ? [
        t('workflow.inspector.color_hue'),
        t('workflow.inspector.color_saturation'),
        t('workflow.inspector.color_value'),
      ]
    : [
        t('workflow.inspector.color_red'),
        t('workflow.inspector.color_green'),
        t('workflow.inspector.color_blue'),
      ],
)
const limits = computed<[number, number, number]>(() =>
  range.value.space === 'hsv' ? [360, 100, 100] : [255, 255, 255],
)
const spaces = computed(() => [
  { label: t('workflow.inspector.color_rgb'), value: 'rgb' },
  { label: t('workflow.inspector.color_hsv'), value: 'hsv' },
])

function setSpace(space: ColorSpace): void {
  emit(
    'update:model-value',
    space === 'hsv'
      ? { space, minimum: [0, 0, 0], maximum: [360, 100, 100] }
      : { space, minimum: [0, 0, 0], maximum: [255, 255, 255] },
  )
}

function setChannel(bound: 'minimum' | 'maximum', index: number, value: number): void {
  if (!Number.isFinite(value)) return
  const next = normalize(range.value)
  next[bound][index] = Math.max(0, Math.min(limits.value[index], Math.trunc(value)))
  emit('update:model-value', next)
}

function normalize(value: unknown): ColorRange {
  if (!value || typeof value !== 'object')
    return { space: 'rgb', minimum: [0, 0, 0], maximum: [255, 255, 255] }
  const candidate = value as Partial<ColorRange>
  const space: ColorSpace = candidate.space === 'hsv' ? 'hsv' : 'rgb'
  const limits: [number, number, number] = space === 'hsv' ? [360, 100, 100] : [255, 255, 255]
  return {
    space,
    minimum: normalizeChannels(candidate.minimum, [0, 0, 0], limits),
    maximum: normalizeChannels(candidate.maximum, limits, limits),
  }
}

function normalizeChannels(
  value: unknown,
  fallback: [number, number, number],
  limits: [number, number, number],
): [number, number, number] {
  if (!Array.isArray(value) || value.length !== 3) return [...fallback]
  return [0, 1, 2].map((index) => {
    const channel = Number(value[index])
    return Number.isFinite(channel)
      ? Math.max(0, Math.min(limits[index], Math.trunc(channel)))
      : fallback[index]
  }) as [number, number, number]
}
</script>
