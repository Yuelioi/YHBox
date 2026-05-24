<script setup lang="ts">
import type { InputSpec } from '@bindings/yhbox/internal/node'
defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: string): void }>()
</script>
<template>
  <UInput
    size="sm"
    type="password"
    :model-value="value ?? ''"
    @update:model-value="(v: string) => emit('update', v)"
  />
</template>
