<template>
  <BaseModal
    :open="open"
    :title="snippetId ? t('workflow.snippets.edit_title') : t('workflow.snippets.create_title')"
    icon="i-tabler-bookmark-plus"
    size="lg"
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-3">
      <UFormField :label="t('workflow.snippets.name')" required>
        <UInput
          v-model="name"
          data-testid="workflow-snippet-name"
          autofocus
          maxlength="80"
          :placeholder="t('workflow.snippets.name_placeholder')"
          @keydown.enter.prevent="submit"
        />
      </UFormField>
      <UFormField :label="t('workflow.snippets.description')">
        <UTextarea v-model="description" :rows="3" maxlength="1000" />
      </UFormField>
      <div class="grid grid-cols-2 gap-3">
        <UFormField :label="t('workflow.snippets.category')">
          <UInput v-model="category" maxlength="80" />
        </UFormField>
        <UFormField :label="t('workflow.snippets.tags')">
          <UInput v-model="tags" :placeholder="t('workflow.snippets.tags_placeholder')" />
        </UFormField>
      </div>
      <div class="rounded-lg border border-default bg-elevated/25 px-3 py-2.5">
        <p class="text-[10px] font-medium text-toned">{{ t('workflow.snippets.payload') }}</p>
        <p class="mt-1 truncate font-mono text-[10px] text-dimmed">{{ nodeTypeId }}</p>
        <p class="mt-1 text-[10px] leading-4 text-muted">
          {{ t('workflow.snippets.payload_hint') }}
        </p>
      </div>
    </div>
    <template #footer>
      <UButton
        color="neutral"
        variant="ghost"
        :label="t('common.cancel')"
        @click="emit('update:open', false)"
      />
      <UButton
        data-testid="workflow-snippet-save"
        icon="i-tabler-device-floppy"
        :loading="busy"
        :disabled="!name.trim()"
        :label="t('common.save')"
        @click="submit"
      />
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseModal from '@/components/common/BaseModal.vue'

const props = defineProps<{
  open: boolean
  snippetId: string
  nodeTypeId: string
  initial?: { name: string; description?: string; category?: string; tags: string[] }
  busy?: boolean
}>()
const emit = defineEmits<{
  'update:open': [open: boolean]
  save: [value: { name: string; description: string; category: string; tags: string[] }]
}>()
const { t } = useI18n()
const name = ref('')
const description = ref('')
const category = ref('')
const tags = ref('')

watch(
  () => [props.open, props.initial] as const,
  ([open]) => {
    if (!open) return
    name.value = props.initial?.name ?? ''
    description.value = props.initial?.description ?? ''
    category.value = props.initial?.category ?? ''
    tags.value = props.initial?.tags.join(', ') ?? ''
  },
  { immediate: true },
)

function submit(): void {
  if (!name.value.trim() || props.busy) return
  emit('save', {
    name: name.value.trim(),
    description: description.value.trim(),
    category: category.value.trim(),
    tags: [
      ...new Set(
        tags.value
          .split(/[,，]/)
          .map((value) => value.trim())
          .filter(Boolean),
      ),
    ],
  })
}
</script>
