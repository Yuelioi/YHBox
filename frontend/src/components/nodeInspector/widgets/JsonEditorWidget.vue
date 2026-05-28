<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { InputSpec } from '@bindings/yhbox/internal/node'

const { t } = useI18n()

const props = defineProps<{ input: InputSpec; value: any }>()
const emit = defineEmits<{ (e: 'update', v: any): void }>()

const text = ref(JSON.stringify(props.value ?? {}, null, 2))
const error = ref<string | null>(null)

watch(() => props.value, (v) => {
  text.value = JSON.stringify(v ?? {}, null, 2)
})

function onInput(v: string) {
  text.value = v
  try {
    const parsed = JSON.parse(v)
    error.value = null
    emit('update', parsed)
  } catch (e: any) {
    error.value = e.message
  }
}

const rows = computed(() => Number(props.input.widget?.props?.rows ?? 8))
</script>
<template>
  <div>
    <UTextarea
      size="sm"
      :model-value="text"
      :rows="rows"
      class="font-mono"
      @update:model-value="onInput"
    />
    <p v-if="error" class="text-xs text-red-500 mt-1">{{ t('jsonEditor.error', { msg: error }) }}</p>
  </div>
</template>
