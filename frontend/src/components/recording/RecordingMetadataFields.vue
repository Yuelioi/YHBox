<template>
  <UFormField :label="t('recordingSave.name')" required>
    <UInput v-model="name" class="w-full" maxlength="80" autofocus />
  </UFormField>
  <UFormField :label="t('common.description')" :hint="t('common.optional')">
    <UTextarea v-model="description" class="w-full" :rows="2" />
  </UFormField>
  <div class="grid grid-cols-2 gap-3">
    <UFormField :label="t('common.category')" :hint="t('common.optional')">
      <UInputMenu
        v-model="category"
        class="w-full"
        :items="categoryOptions"
        :create-item="'always'"
        :placeholder="t('recordingSave.category_placeholder')"
        @create="createCategory"
      />
    </UFormField>
    <UFormField :label="t('common.tags')" :hint="t('recordingSave.tags_hint')">
      <UInputMenu
        v-model="tags"
        class="w-full"
        :items="tagOptions"
        :create-item="'always'"
        :placeholder="t('recordingSave.tags_placeholder')"
        multiple
        @create="createTag"
      />
    </UFormField>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  categories: string[]
  tagSuggestions: string[]
}>()
const name = defineModel<string>('name', { required: true })
const description = defineModel<string>('description', { required: true })
const category = defineModel<string>('category', { required: true })
const tags = defineModel<string[]>('tags', { required: true })
const { t } = useI18n()

const categoryOptions = computed(() => uniqueStrings([...props.categories, category.value]))
const tagOptions = computed(() => uniqueStrings([...props.tagSuggestions, ...tags.value]))

function createCategory(value: string): void {
  const created = value.trim()
  if (created) category.value = created
}

function createTag(value: string): void {
  const created = value.trim()
  if (created) tags.value = uniqueStrings([...tags.value, created])
}

function uniqueStrings(values: string[]): string[] {
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))]
}
</script>
