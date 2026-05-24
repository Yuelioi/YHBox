<script setup lang="ts">
import { computed } from 'vue'
import type { InputSpec, EnumOption } from '@bindings/yhbox/internal/node'
const props = defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: any): void }>()
const options = computed<EnumOption[]>(() => (props.input.widget?.props?.options as EnumOption[]) ?? [])
const items = computed(() => options.value.map(o => ({ label: o.label, value: o.value })))
</script>
<template>
  <USelect
    size="sm"
    :model-value="value"
    :items="items"
    value-key="value"
    @update:model-value="(v: any) => emit('update', v)"
  />
</template>
