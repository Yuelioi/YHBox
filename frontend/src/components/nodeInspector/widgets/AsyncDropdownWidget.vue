<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { InputSpec, EnumOption } from '@bindings/yhbox/internal/node'
import { NodeService } from '@bindings/yhbox/internal/node'

const props = defineProps<{
  input: InputSpec
  value: any
  nodeID: string        // 节点 instance ID
  specKind: string      // 节点 Kind
  currentInputs: Record<string, any>  // 节点其他字段 (for context-aware dropdowns)
}>()
const emit = defineEmits<{ (e: 'update', v: any): void }>()

const asyncSource = computed(() => props.input.widget?.props?.asyncSource as string ?? '')
const options = ref<EnumOption[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

async function reload() {
  if (!asyncSource.value) return
  loading.value = true
  error.value = null
  try {
    options.value = await NodeService.AsyncOptions(props.nodeID, props.specKind, asyncSource.value, props.currentInputs)
  } catch (e: any) {
    error.value = String(e?.message ?? e)
    options.value = []
  } finally {
    loading.value = false
  }
}

watch([asyncSource, () => props.currentInputs], reload, { immediate: true, deep: true })

const items = computed(() => options.value.map(o => ({ label: o.label, value: o.value })))
</script>
<template>
  <div>
    <USelect
      size="sm"
      :model-value="value"
      :items="items"
      :loading="loading"
      value-key="value"
      @update:model-value="(v: any) => emit('update', v)"
    />
    <p v-if="error" class="text-xs text-red-500 mt-1">{{ error }}</p>
  </div>
</template>
