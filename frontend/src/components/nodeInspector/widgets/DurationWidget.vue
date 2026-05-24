<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { InputSpec } from '@bindings/yhbox/internal/node'

const props = defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: number): void }>()

// Go time.Duration 是纳秒 int64 (大数). FE 用 number ms 显示, 转换提交.
const ms = ref(typeof props.value === 'number' ? props.value / 1_000_000 : 0)
const unit = ref<'ms' | 's'>(props.input.widget?.props?.unit === 's' ? 's' : 'ms')

watch(() => props.value, (v) => {
  ms.value = typeof v === 'number' ? v / 1_000_000 : 0
})

function update(newMs: number) {
  ms.value = newMs
  emit('update', newMs * 1_000_000)  // 转纳秒
}

const display = computed(() => unit.value === 's' ? ms.value / 1000 : ms.value)

function onChange(v: number) {
  update(unit.value === 's' ? v * 1000 : v)
}
</script>
<template>
  <div class="flex items-center gap-2">
    <UInputNumber size="sm" :model-value="display" :step="unit === 's' ? 0.1 : 100"
      @update:model-value="onChange" class="flex-1" />
    <USelect size="sm" v-model="unit"
      :items="[{ label: 'ms', value: 'ms' }, { label: 's', value: 's' }]"
      value-key="value" class="w-16" />
  </div>
</template>
