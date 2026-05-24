<script setup lang="ts">
import type { InputSpec } from '@bindings/yhbox/internal/node'
defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: boolean): void }>()
</script>
<template>
  <UCheckbox
    :model-value="!!value"
    @update:model-value="(v: boolean) => emit('update', v)"
  />
</template>
