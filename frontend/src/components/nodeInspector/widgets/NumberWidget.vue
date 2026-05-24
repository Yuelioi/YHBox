<script setup lang="ts">
import type { InputSpec } from '@bindings/yhbox/internal/node'
defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: number): void }>()
</script>
<template>
  <UInputNumber
    size="sm"
    :model-value="Number(value ?? 0)"
    @update:model-value="(v: number) => emit('update', v)"
  />
</template>
