<template>
  <div class="space-y-2.5">
    <div class="flex items-center gap-2 rounded-lg border border-default bg-muted/25 p-2.5">
      <span
        class="size-9 shrink-0 rounded-md border border-white/15 shadow-inner"
        :style="{ background: previewGradient }"
        aria-hidden="true"
      />
      <div class="min-w-0 flex-1">
        <p class="truncate text-xs font-medium text-highlighted">{{ summary }}</p>
        <p class="text-[10px]" :class="fullRange ? 'text-warning' : 'text-muted'">
          {{
            t(
              fullRange
                ? 'workflow.inspector.color_unsampled_hint'
                : 'workflow.inspector.color_sample_hint',
            )
          }}
        </p>
      </div>
      <UButton
        icon="i-tabler-color-picker"
        :label="compact ? undefined : t('workflow.inspector.pick_color')"
        color="primary"
        variant="soft"
        size="xs"
        :disabled="!targetSlot"
        :loading="picking"
        @click="pickColor"
      />
    </div>

    <AdaptiveSelect
      :model-value="range.space"
      :items="spaces"
      value-key="value"
      label-key="label"
      width-mode="fill"
      @update:model-value="setSpace($event === 'hsv' ? 'hsv' : 'rgb')"
    />

    <UCollapsible v-model:open="advancedOpen">
      <UButton
        :label="t('workflow.inspector.color_channels')"
        :trailing-icon="advancedOpen ? 'i-tabler-chevron-up' : 'i-tabler-chevron-down'"
        color="neutral"
        variant="ghost"
        size="xs"
        class="w-full justify-between"
      />
      <template #content>
        <div class="grid grid-cols-3 gap-2 pt-2">
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
      </template>
    </UCollapsible>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import { pickTargetValue, type TargetColorRange } from './useTargetPicker'
import {
  colorRangeValueFromTarget,
  type ColorRangeValue as ColorRange,
  type ColorSpace,
} from './targetValueMapping'

const props = defineProps<{ modelValue: unknown; targetSlot?: string; compact?: boolean }>()
const emit = defineEmits<{ 'update:model-value': [value: ColorRange] }>()
const { t } = useI18n()
const toast = useToast()
const advancedOpen = ref(false)
const picking = ref(false)
const range = computed<ColorRange>(() => normalize(props.modelValue))
const targetSlot = computed(() => props.targetSlot ?? '')
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
  { label: t('workflow.inspector.color_hsv'), value: 'hsv' },
  { label: t('workflow.inspector.color_rgb'), value: 'rgb' },
])
const summary = computed(() => {
  const labels = range.value.space === 'hsv' ? ['H', 'S', 'V'] : ['R', 'G', 'B']
  return `${range.value.space.toUpperCase()} · ${labels
    .map((label, index) => `${label} ${range.value.minimum[index]}–${range.value.maximum[index]}`)
    .join(' · ')}`
})
const previewGradient = computed(() => {
  const minimum = rgbFor(range.value.space, range.value.minimum)
  const maximum = rgbFor(range.value.space, range.value.maximum)
  return `linear-gradient(135deg, rgb(${minimum.join(' ')}) 0 48%, rgb(${maximum.join(' ')}) 52% 100%)`
})
const fullRange = computed(() =>
  range.value.minimum.every(
    (minimum, index) => minimum === 0 && range.value.maximum[index] === limits.value[index],
  ),
)

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

async function pickColor(): Promise<void> {
  if (!targetSlot.value || picking.value) return
  picking.value = true
  try {
    const picked = await pickTargetValue<TargetColorRange>(
      'color',
      targetSlot.value,
      range.value.space,
    )
    if (!picked) return
    emit('update:model-value', colorRangeValueFromTarget(picked, range.value.space))
  } catch (error) {
    toast.add({
      title: t('workflow.inspector.pick_failed'),
      description: error instanceof Error ? error.message : String(error),
      color: 'error',
    })
  } finally {
    picking.value = false
  }
}

function normalize(value: unknown): ColorRange {
  if (!value || typeof value !== 'object')
    return { space: 'hsv', minimum: [0, 0, 0], maximum: [360, 100, 100] }
  const candidate = value as Partial<ColorRange>
  const space: ColorSpace = candidate.space === 'rgb' ? 'rgb' : 'hsv'
  const channelLimits: [number, number, number] =
    space === 'hsv' ? [360, 100, 100] : [255, 255, 255]
  return {
    space,
    minimum: normalizeChannels(candidate.minimum, [0, 0, 0], channelLimits),
    maximum: normalizeChannels(candidate.maximum, channelLimits, channelLimits),
  }
}

function normalizeChannels(
  value: unknown,
  fallback: [number, number, number],
  channelLimits: [number, number, number],
): [number, number, number] {
  if (!Array.isArray(value) || value.length !== 3) return [...fallback]
  return [0, 1, 2].map((index) => {
    const channel = Number(value[index])
    return Number.isFinite(channel)
      ? Math.max(0, Math.min(channelLimits[index], Math.trunc(channel)))
      : fallback[index]
  }) as [number, number, number]
}

function rgbFor(space: ColorSpace, channels: [number, number, number]): [number, number, number] {
  if (space === 'rgb') return channels
  const [hue, saturationValue, valueValue] = channels
  const saturation = saturationValue / 100
  const value = valueValue / 100
  const chroma = value * saturation
  const sector = (((hue % 360) + 360) % 360) / 60
  const x = chroma * (1 - Math.abs((sector % 2) - 1))
  const [red, green, blue] =
    sector < 1
      ? [chroma, x, 0]
      : sector < 2
        ? [x, chroma, 0]
        : sector < 3
          ? [0, chroma, x]
          : sector < 4
            ? [0, x, chroma]
            : sector < 5
              ? [x, 0, chroma]
              : [chroma, 0, x]
  const match = value - chroma
  return [red, green, blue].map((channel) => Math.round((channel + match) * 255)) as [
    number,
    number,
    number,
  ]
}
</script>
