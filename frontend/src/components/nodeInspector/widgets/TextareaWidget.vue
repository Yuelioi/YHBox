<script setup lang="ts">
import { computed } from 'vue'
import type { InputSpec } from '@bindings/yhbox/internal/node'
const props = defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: string): void }>()
const rows = computed(() => Number(props.input.widget?.props?.rows ?? 4))
const maxLength = computed(() => props.input.widget?.props?.maxLength as number | undefined)
</script>
<template>
  <UTextarea
    size="sm"
    :model-value="value ?? ''"
    :rows="rows"
    :maxlength="maxLength"
    @update:model-value="(v: string) => emit('update', v)"
  />
</template>
