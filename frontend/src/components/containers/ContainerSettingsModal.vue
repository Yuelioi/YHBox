<!-- frontend/src/components/containers/ContainerSettingsModal.vue -->
<!-- 容器基本信息 — 从右抽屉抽出, 顶部 ⚙ 设置 / Ctrl+, 触发. -->
<template>
  <BaseModal v-model:open="modelOpen" :title="t('containers.settings_title')" icon="i-tabler-settings" size="lg">
    <div class="space-y-3">
      <UFormField :label="t('common.name')" required>
        <UInput v-model="form.name" :placeholder="t('containers.name_placeholder')" size="sm" />
      </UFormField>

      <UFormField :label="t('containers.hotkey_label')" :hint="t('containers.hotkey_hint')">
        <HotkeyCaptureInput v-model="form.hotkey" />
      </UFormField>

      <UFormField :label="t('common.description')">
        <UTextarea v-model="form.description" :rows="3" size="sm" :placeholder="t('containers.description_placeholder')" />
      </UFormField>

      <UFormField :label="t('common.tags')" :hint="t('containers.tags_hint')">
        <UInputMenu v-model="form.tags" multiple :items="allTags" :create-item="'always'" size="sm" @create="(v: string) => (form.tags = [...(form.tags ?? []), v])" />
      </UFormField>

      <UFormField :label="t('containers.input_backend_label')">
        <USelect v-model="form.inputBackend" :items="INPUT_BACKEND_OPTIONS" size="sm" />
        <template #help>
          <p class="text-xs text-muted">{{ t('containers.input_backend_hint') }}</p>
        </template>
      </UFormField>

      <UFormField :label="t('containers.capture_backend_label')">
        <USelect v-model="form.captureBackend" :items="CAPTURE_BACKEND_OPTIONS" size="sm" />
      </UFormField>

      <UFormField :label="t('containers.scale_tolerance_label')">
        <UInputNumber v-model="form.scaleTolerance" :min="1" :max="4" :step="0.1" size="sm" class="w-36" />
      </UFormField>
    </div>

    <template #footer>
      <UButton variant="ghost" color="neutral" @click="modelOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" icon="i-tabler-check" @click="onConfirm">{{ t('common.save') }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'
import BaseModal from '@/components/common/BaseModal.vue'

const { t } = useI18n()

interface FormState {
  name: string
  hotkey: string
  description: string
  tags: string[]
  inputBackend: string
  captureBackend: string
  scaleTolerance: number
}

const props = defineProps<{
  open: boolean
  initial: FormState
  allTags: string[]
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  save: [v: FormState]
}>()

const modelOpen = useDialogOpen(props, emit)

const form = ref<FormState>({ ...props.initial })
watch(() => props.initial, v => { form.value = { ...v } }, { deep: true })

const INPUT_BACKEND_OPTIONS = computed(() => [
  { value: 'postmessage', label: t('containers.input_backend_postmessage') },
  { value: 'sendinput', label: t('containers.input_backend_sendinput') },
])

const CAPTURE_BACKEND_OPTIONS = [
  { value: 'auto', label: 'auto' },
  { value: 'gdi', label: 'GDI' },
  { value: 'wgc', label: 'WGC' },
  { value: 'mock', label: 'Mock' },
]

function onConfirm() {
  emit('save', { ...form.value })
  modelOpen.value = false
}
</script>
