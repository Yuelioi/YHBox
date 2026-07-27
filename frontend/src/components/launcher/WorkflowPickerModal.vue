<template>
  <BaseModal
    :open="open"
    :title="t('settingsLauncher.picker_title')"
    icon="i-tabler-layout-grid-add"
    size="2xl"
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-3">
      <UInput
        v-model="search"
        class="workflow-picker-search"
        icon="i-tabler-search"
        :placeholder="t('settingsLauncher.picker_search')"
        :aria-label="t('settingsLauncher.picker_search')"
      />

      <div class="grid gap-2 sm:grid-cols-2">
        <AdaptiveSelect
          v-model="category"
          :items="categoryItems"
          width-mode="fill"
          :placeholder="t('settingsLauncher.picker_all_categories')"
          :aria-label="t('settingsLauncher.picker_category')"
        />
        <USelectMenu
          v-model="tags"
          multiple
          :items="tagItems"
          value-key="value"
          label-key="label"
          :virtualize="tagItems.length > 40"
          :search-input="{ placeholder: t('settingsLauncher.picker_search_tags') }"
          :placeholder="t('settingsLauncher.picker_all_tags')"
          :aria-label="t('settingsLauncher.picker_tags')"
        />
      </div>

      <div class="workflow-picker-results" aria-live="polite">
        <div v-if="loading" class="space-y-2 p-2" aria-busy="true">
          <USkeleton v-for="index in 5" :key="index" class="h-14 w-full" />
        </div>

        <div
          v-else-if="issue"
          class="flex min-h-48 flex-col items-center justify-center gap-3 px-6 text-center"
          role="alert"
        >
          <UIcon name="i-tabler-alert-circle" class="size-5 text-error" />
          <p class="text-xs text-muted">{{ issue }}</p>
          <UButton
            size="xs"
            color="neutral"
            variant="soft"
            icon="i-tabler-refresh"
            @click="loadPage"
          >
            {{ t('settingsLauncher.picker_retry') }}
          </UButton>
        </div>

        <div
          v-else-if="!items.length"
          class="flex min-h-48 flex-col items-center justify-center gap-2 px-6 text-center"
        >
          <UIcon name="i-tabler-search-off" class="size-5 text-dimmed" />
          <p class="text-xs font-medium text-default">
            {{ t('settingsLauncher.picker_empty') }}
          </p>
          <p class="text-[11px] text-dimmed">
            {{ t('settingsLauncher.picker_empty_hint') }}
          </p>
        </div>

        <div
          v-else
          class="space-y-1 p-1"
          role="listbox"
          :aria-label="t('settingsLauncher.picker_results')"
        >
          <UButton
            v-for="workflow in items"
            :key="workflow.workflowId"
            color="neutral"
            variant="ghost"
            block
            class="workflow-picker-row"
            role="option"
            :aria-selected="isSelected(workflow.workflowId)"
            @click="toggle(workflow)"
          >
            <span
              class="inline-flex size-9 shrink-0 items-center justify-center rounded-lg border border-default bg-elevated"
            >
              <UIcon name="i-tabler-player-play" class="size-4 text-toned" />
            </span>
            <span class="min-w-0 flex-1 text-left">
              <span class="flex min-w-0 items-center gap-2">
                <span class="truncate text-xs font-semibold text-highlighted">
                  {{ workflow.name }}
                </span>
                <UBadge
                  v-if="addedCounts[workflow.workflowId]"
                  size="xs"
                  color="neutral"
                  variant="subtle"
                >
                  {{
                    t('settingsLauncher.picker_added_count', {
                      n: addedCounts[workflow.workflowId],
                    })
                  }}
                </UBadge>
              </span>
              <span class="mt-1 flex min-w-0 items-center gap-1.5 text-[11px] text-dimmed">
                <span v-if="workflow.category" class="shrink-0">{{ workflow.category }}</span>
                <span v-if="workflow.category && workflow.tags?.length" aria-hidden="true">·</span>
                <span v-if="workflow.tags?.length" class="truncate">
                  {{ workflow.tags.join(' · ') }}
                </span>
                <span v-if="!workflow.category && !workflow.tags?.length" class="truncate">
                  {{ workflow.workflowId }}
                </span>
              </span>
            </span>
            <span
              class="inline-flex size-7 shrink-0 items-center justify-center rounded-md border"
              :class="
                isSelected(workflow.workflowId)
                  ? 'border-primary bg-primary/15 text-primary'
                  : 'border-default text-dimmed'
              "
            >
              <UIcon
                :name="isSelected(workflow.workflowId) ? 'i-tabler-check' : 'i-tabler-plus'"
                class="size-4"
              />
            </span>
          </UButton>
        </div>
      </div>

      <div class="flex items-center justify-between gap-3 text-[11px] text-dimmed">
        <span>
          {{
            t('settingsLauncher.picker_range', {
              start: resultStart,
              end: resultEnd,
              total,
            })
          }}
        </span>
        <div class="flex items-center gap-1">
          <UButton
            size="xs"
            color="neutral"
            variant="ghost"
            icon="i-tabler-chevron-left"
            :disabled="page <= 1 || loading"
            :aria-label="t('settingsLauncher.picker_previous')"
            @click="page -= 1"
          />
          <span class="min-w-14 text-center">
            {{ t('settingsLauncher.picker_page', { page, pages: pageCount }) }}
          </span>
          <UButton
            size="xs"
            color="neutral"
            variant="ghost"
            icon="i-tabler-chevron-right"
            :disabled="page >= pageCount || loading"
            :aria-label="t('settingsLauncher.picker_next')"
            @click="page += 1"
          />
        </div>
      </div>
    </div>

    <template #footer>
      <span v-if="selectedCount" class="mr-auto text-xs text-muted">
        {{ t('settingsLauncher.picker_selected', { n: selectedCount }) }}
      </span>
      <UButton color="neutral" variant="ghost" @click="emit('update:open', false)">
        {{ t('common.cancel') }}
      </UButton>
      <UButton
        color="primary"
        icon="i-tabler-plus"
        :disabled="selectedCount === 0"
        @click="addSelected"
      >
        {{ t('settingsLauncher.picker_add_selected', { n: selectedCount }) }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { workflowTransport, type SourceView } from '@/app/transport/workflow'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import { errorMessage } from '@/lib/invoke'

const props = withDefaults(
  defineProps<{
    open: boolean
    addedCounts?: Record<string, number>
  }>(),
  { addedCounts: () => ({}) },
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  add: [workflows: SourceView[]]
}>()

const { t } = useI18n()
const search = ref('')
const category = ref('')
const tags = ref<string[]>([])
const page = ref(1)
const pageSize = 30
const items = ref<SourceView[]>([])
const total = ref(0)
const categories = ref<Array<{ value: string; count: number }>>([])
const availableTags = ref<Array<{ value: string; count: number }>>([])
const loading = ref(false)
const issue = ref('')
const selected = ref<Record<string, SourceView>>({})
let debounceTimer: ReturnType<typeof setTimeout> | undefined
let requestSequence = 0
let resetting = false

const categoryItems = computed(() => [
  { label: t('settingsLauncher.picker_all_categories'), value: '' },
  ...categories.value.map((item) => ({
    label: `${item.value} (${item.count})`,
    value: item.value,
  })),
])
const tagItems = computed(() =>
  availableTags.value.map((item) => ({
    label: `${item.value} (${item.count})`,
    value: item.value,
  })),
)
const selectedCount = computed(() => Object.keys(selected.value).length)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const resultStart = computed(() => (total.value ? (page.value - 1) * pageSize + 1 : 0))
const resultEnd = computed(() => Math.min(page.value * pageSize, total.value))

function focusSearch() {
  document.querySelector<HTMLInputElement>('.workflow-picker-search input')?.focus()
}

function isSelected(workflowId: string) {
  return Boolean(selected.value[workflowId])
}

function toggle(workflow: SourceView) {
  const next = { ...selected.value }
  if (next[workflow.workflowId]) delete next[workflow.workflowId]
  else next[workflow.workflowId] = workflow
  selected.value = next
}

function addSelected() {
  const workflows = Object.values(selected.value)
  if (!workflows.length) return
  emit('add', workflows)
  selected.value = {}
}

async function loadPage() {
  if (!props.open) return
  const sequence = ++requestSequence
  loading.value = true
  issue.value = ''
  try {
    const result = await workflowTransport.querySources({
      search: search.value.trim(),
      category: category.value,
      tags: tags.value,
      createdSince: '',
      updatedSince: '',
      sort: 'updated_desc',
      page: page.value,
      pageSize,
    })
    if (sequence !== requestSequence) return
    items.value = result.items
    total.value = result.total
    categories.value = result.categories
    availableTags.value = result.tags
    if (page.value > pageCount.value) page.value = pageCount.value
  } catch (error) {
    if (sequence !== requestSequence) return
    items.value = []
    total.value = 0
    issue.value = errorMessage(error)
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

watch(
  () => props.open,
  (open) => {
    clearTimeout(debounceTimer)
    if (!open) {
      requestSequence += 1
      return
    }
    resetting = true
    search.value = ''
    category.value = ''
    tags.value = []
    page.value = 1
    selected.value = {}
    void nextTick(() => {
      resetting = false
      void loadPage()
      focusSearch()
    })
  },
  { immediate: true },
)

watch(page, () => {
  if (!resetting) void loadPage()
})
watch([search, category, tags], () => {
  if (!props.open || resetting) return
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    if (page.value === 1) void loadPage()
    else page.value = 1
  }, 180)
})
</script>

<style scoped>
.workflow-picker-results {
  min-height: 304px;
  max-height: min(48vh, 440px);
  overflow-y: auto;
  border: 1px solid var(--ui-border);
  border-radius: 10px;
  background: var(--ui-bg);
}

.workflow-picker-row {
  min-height: 58px;
  justify-content: flex-start;
  gap: 10px;
  padding: 8px;
  text-align: left;
}
</style>
