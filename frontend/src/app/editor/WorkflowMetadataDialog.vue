<template>
  <BaseModal
    :open="open"
    :title="t('workflow.list.edit_metadata_title')"
    size="lg"
    :dismissible="!busy"
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-4">
      <UFormField :label="t('workflow.editor.workflow_name')">
        <UInput v-model="draft.name" class="w-full" autofocus />
      </UFormField>
      <UFormField :label="t('workflow.list.description_label')">
        <UTextarea v-model="draft.description" class="w-full" :rows="3" />
      </UFormField>
      <div class="grid grid-cols-2 gap-3">
        <UFormField :label="t('common.category')">
          <UInputMenu
            v-model="draft.category"
            class="w-full"
            :items="categoryOptions"
            :create-item="'always'"
            :placeholder="t('workflow.list.category_placeholder')"
            @create="createCategory"
          />
        </UFormField>
        <UFormField :label="t('common.tags')" :hint="t('workflow.editor.tags_hint')">
          <UInput v-model="tagsText" class="w-full" />
        </UFormField>
      </div>
      <p v-if="error" class="text-xs text-error" role="alert">{{ error }}</p>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton
          color="neutral"
          variant="ghost"
          :label="t('common.cancel')"
          :disabled="busy"
          @click="emit('update:open', false)"
        />
        <UButton
          :label="t('common.save')"
          :loading="busy"
          :disabled="!draft.name.trim()"
          @click="submit"
        />
      </div>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseModal from '@/components/common/BaseModal.vue'
import { addCreatedCategory, uniqueCategoryOptions } from '@/lib/categoryOptions'
import type { WorkflowMetadataDraft } from './EditorWorkflowMetadataController'

const props = defineProps<
  WorkflowMetadataDraft & { open: boolean; busy?: boolean; error?: string }
>()
const emit = defineEmits<{
  'update:open': [open: boolean]
  submit: [draft: WorkflowMetadataDraft]
}>()
const { t } = useI18n()
const draft = reactive<WorkflowMetadataDraft>({
  name: '',
  description: '',
  category: '',
  tags: [],
})
const tagsText = ref('')
const createdCategories = ref<string[]>([])
const categoryOptions = computed(() =>
  uniqueCategoryOptions([props.category, draft.category], createdCategories.value),
)

watch(
  () => [props.open, props.name, props.description, props.category, props.tags] as const,
  () => {
    if (!props.open) return
    draft.name = props.name
    draft.description = props.description
    draft.category = props.category
    draft.tags = [...props.tags]
    tagsText.value = props.tags.join(', ')
  },
  { immediate: true },
)

function createCategory(value: string): void {
  const result = addCreatedCategory(createdCategories.value, value)
  createdCategories.value = result.categories
  draft.category = result.value
}

function submit(): void {
  emit('submit', {
    name: draft.name.trim(),
    description: draft.description.trim(),
    category: draft.category.trim(),
    tags: [
      ...new Set(
        tagsText.value
          .split(/[,，]/)
          .map((tag) => tag.trim())
          .filter(Boolean),
      ),
    ],
  })
}
</script>
