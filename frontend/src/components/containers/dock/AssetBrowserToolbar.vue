<template>
  <div class="asset-browser-toolbar shrink-0 space-y-2">
    <div class="flex items-center gap-2">
      <UInput
        v-if="showSearch"
        ref="searchRef"
        v-model="query"
        icon="i-tabler-search"
        size="sm"
        class="min-w-0 flex-1"
        :placeholder="searchPlaceholder"
        :aria-label="searchPlaceholder"
      />

      <UPopover mode="click" :ui="{ content: 'w-72 p-3' }">
        <UButton
          size="sm"
          variant="soft"
          color="neutral"
          icon="i-tabler-adjustments-horizontal"
          class="shrink-0"
          :aria-label="t('assetBrowser.filters')"
        >
          <span class="asset-toolbar-label">{{ t('assetBrowser.filters') }}</span>
          <UBadge v-if="activeFilterCount" color="primary" variant="subtle" size="sm">
            {{ activeFilterCount }}
          </UBadge>
        </UButton>
        <template #content>
          <div class="space-y-3">
            <UFormField :label="t('common.category')">
              <USelectMenu
                v-model="category"
                :items="categoryItems"
                value-key="id"
                size="sm"
                :aria-label="t('common.category')"
              />
            </UFormField>
            <UFormField :label="t('library.detail.tags')">
              <UInputMenu
                v-model="tags"
                multiple
                :items="tagItems"
                size="sm"
                :placeholder="t('library.explorer.filter_tags')"
                :aria-label="t('library.explorer.filter_tags')"
              />
            </UFormField>
            <UFormField :label="t('assetBrowser.sortBy')">
              <div class="flex items-center gap-2">
                <USelect
                  v-model="sortKey"
                  :items="sortItems"
                  size="sm"
                  class="flex-1"
                  :aria-label="t('assetBrowser.sortBy')"
                />
                <UButton
                  size="sm"
                  variant="outline"
                  color="neutral"
                  :icon="sortDesc ? 'i-tabler-sort-descending' : 'i-tabler-sort-ascending'"
                  :aria-label="sortDesc ? t('assetBrowser.sortDesc') : t('assetBrowser.sortAsc')"
                  class="size-8 p-0"
                  @click="sortDesc = !sortDesc"
                />
              </div>
            </UFormField>
            <UButton
              v-if="activeFilterCount"
              block
              size="xs"
              color="neutral"
              variant="ghost"
              icon="i-tabler-filter-off"
              @click="clearFilters"
            >
              {{ t('assetBrowser.clearFilters') }}
            </UButton>
          </div>
        </template>
      </UPopover>

      <div v-if="allowViewSwitch" class="flex shrink-0 rounded-md bg-elevated/50 p-0.5">
        <UButton
          size="xs"
          color="neutral"
          :variant="view === 'grid' ? 'soft' : 'ghost'"
          icon="i-tabler-layout-grid"
          class="size-7 p-0"
          :aria-label="t('assetBrowser.gridView')"
          :aria-pressed="view === 'grid'"
          @click="view = 'grid'"
        />
        <UButton
          size="xs"
          color="neutral"
          :variant="view === 'list' ? 'soft' : 'ghost'"
          icon="i-tabler-list"
          class="size-7 p-0"
          :aria-label="t('assetBrowser.listView')"
          :aria-pressed="view === 'list'"
          @click="view = 'list'"
        />
      </div>
    </div>

    <div
      v-if="showCategoryScopes"
      class="category-scopes flex min-w-0 items-center gap-1 overflow-x-auto pb-0.5"
    >
      <UButton
        v-for="item in categoryItems"
        :key="item.id"
        size="xs"
        color="neutral"
        :variant="category === item.id ? 'soft' : 'ghost'"
        class="shrink-0 rounded-md"
        :class="category === item.id ? 'text-primary' : 'text-muted'"
        :aria-pressed="category === item.id"
        @click="category = item.id"
      >
        {{ item.label }}
      </UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'

interface SelectItem {
  label: string
  value: string
}

interface CategoryItem {
  label: string
  id: string
}

const {
  searchPlaceholder,
  categoryItems,
  tagItems,
  sortItems,
  allowViewSwitch = false,
  showCategoryScopes = true,
  showSearch = true,
} = defineProps<{
  searchPlaceholder: string
  categoryItems: CategoryItem[]
  tagItems: string[]
  sortItems: SelectItem[]
  allowViewSwitch?: boolean
  showCategoryScopes?: boolean
  showSearch?: boolean
}>()

const query = defineModel<string>('query', { required: true })
const category = defineModel<string>('category', { required: true })
const tags = defineModel<string[]>('tags', { required: true })
const sortKey = defineModel<string>('sortKey', { required: true })
const sortDesc = defineModel<boolean>('sortDesc', { required: true })
const view = defineModel<'grid' | 'list'>('view', { default: 'grid' })

const { t } = useI18n()
const searchRef = useTemplateRef<{ $el?: HTMLElement }>('searchRef')
const activeFilterCount = computed(() => (category.value === 'all' ? 0 : 1) + tags.value.length)

function clearFilters() {
  category.value = 'all'
  tags.value = []
}

async function focusSearch() {
  await nextTick()
  searchRef.value?.$el?.querySelector('input')?.focus()
}

defineExpose({ focusSearch })
</script>

<style scoped>
.asset-browser-toolbar {
  container-type: inline-size;
}

.category-scopes {
  scrollbar-width: none;
}

.category-scopes::-webkit-scrollbar {
  display: none;
}

@container (width < 500px) {
  .asset-toolbar-label {
    display: none;
  }
}
</style>
