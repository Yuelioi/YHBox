<template>
  <USelect
    v-model="model"
    v-bind="$attrs"
    :items="items"
    :label-key="labelKey"
    :value-key="valueKey"
    :style="widthStyle"
  />
</template>

<script setup lang="ts" generic="TValue extends SelectValue">
import { computed } from 'vue'
import type { SelectItem, SelectValue } from '@nuxt/ui'
import { adaptiveSelectWidth } from './adaptiveSelect'

defineOptions({ name: 'AdaptiveSelect', inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    items?: SelectItem[]
    labelKey?: string
    valueKey?: string
    widthMode?: 'content' | 'fill' | 'fixed'
    minWidth?: number
    maxWidth?: number
  }>(),
  {
    items: () => [],
    labelKey: 'label',
    valueKey: 'value',
    widthMode: 'content',
    minWidth: 12,
    maxWidth: 40,
  },
)

const model = defineModel<TValue>({ required: true })
const widthStyle = computed(() => {
  if (props.widthMode === 'fixed') return undefined
  if (props.widthMode === 'fill') return { width: '100%' }
  const width = adaptiveSelectWidth(props.items, props.labelKey, props.minWidth, props.maxWidth)
  return { width: `min(100%, ${width}ch)` }
})
</script>
