<template>
  <UTooltip :text="tooltip">
    <UButton
      :data-testid="testId"
      class="w-full"
      :class="showLabel ? 'justify-start' : 'justify-center'"
      :icon="icon"
      :label="showLabel ? label : undefined"
      color="neutral"
      variant="ghost"
      size="xs"
      :disabled="disabled"
      :loading="loading"
      :aria-label="label"
      @click="$emit('click')"
    />
  </UTooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  testId: string
  icon: string
  label: string
  showLabel: boolean
  disabled?: boolean
  loading?: boolean
  shortcut?: string
}>()
defineEmits<{ click: [] }>()
const tooltip = computed(() =>
  props.shortcut ? `${props.label} · ${props.shortcut}` : props.label,
)
</script>
