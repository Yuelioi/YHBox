<template>
  <UFormField :label="t('workflow.hotkey.label')" :hint="t('workflow.hotkey.hint')">
    <HotkeyCaptureInput
      :model-value="entry?.hotkeyStr ?? ''"
      :disabled="loading || saving"
      :aria-label="t('workflow.hotkey.capture_aria', { name })"
      @update:model-value="update"
    />
    <p v-if="entry?.problem" class="mt-1 text-xs text-error" role="alert">
      {{ t('workflow.hotkey.failed') }}：{{ errorMessage(entry.problem) }}
    </p>
  </UFormField>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { backend } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { useToast } from '@/composables/useAppToast'
import { useHotkeysStore } from '@/stores/hotkeys'
import HotkeyCaptureInput from './HotkeyCaptureInput.vue'

const props = defineProps<{ workflowId: string; name: string }>()
const { t } = useI18n()
const toast = useToast()
const store = useHotkeysStore()
const loading = ref(false)
const saving = ref(false)
const registryKey = computed(() => `action.${props.workflowId}`)
const entry = computed(() => store.list.find((item) => item.key === registryKey.value))

watch(
  () => props.workflowId,
  async (workflowId) => {
    if (!workflowId) return
    loading.value = true
    try {
      await backend.hotkeys.reconcile()
      await store.reload()
    } catch (error) {
      showError(error)
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

async function update(value: string) {
  if (!props.workflowId || saving.value) return
  saving.value = true
  try {
    await store.update(registryKey.value, value)
  } catch (error) {
    showError(error)
    await store.reload().catch(() => undefined)
  } finally {
    saving.value = false
  }
}

function showError(error: unknown) {
  toast.add({
    title: t('workflow.hotkey.update_failed'),
    description: errorMessage(error),
    color: 'error',
  })
}
</script>
