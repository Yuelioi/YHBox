<template>
  <div
    class="workspace-page workspace-canvas flex h-full min-h-0 w-full flex-col overflow-hidden"
    data-testid="schedules-view"
  >
    <header
      class="workspace-page__header flex min-h-[72px] shrink-0 items-center justify-between gap-6 px-8 py-4 max-[900px]:flex-col max-[900px]:items-start max-[900px]:px-6"
    >
      <div class="min-w-0">
        <div class="flex items-center gap-3">
          <span
            class="workspace-page__mark flex size-10 shrink-0 items-center justify-center rounded-[10px] border border-primary/25 bg-primary/10 text-primary"
          >
            <UIcon name="i-tabler-calendar-time" class="size-5" />
          </span>
          <div class="flex min-w-0 items-center gap-2">
            <h1
              class="workspace-page__title truncate text-xl leading-tight font-semibold tracking-[-0.02em] text-highlighted"
            >
              {{ t('schedule.title') }}
            </h1>
            <UBadge color="neutral" variant="soft" size="sm">{{ store.list.length }}</UBadge>
          </div>
        </div>
      </div>
      <UButton icon="i-tabler-plus" data-testid="schedule-create" @click="onCreate">
        {{ t('schedule.create') }}
      </UButton>
    </header>

    <main
      class="flex min-h-0 flex-1 flex-col px-6 py-4"
      data-testid="schedule-library"
      data-mode="manage"
    >
      <section
        class="workspace-surface shrink-0 overflow-hidden rounded-t-lg border border-default"
      >
        <form
          class="flex items-center gap-2 border-b border-default p-3"
          role="search"
          @submit.prevent="applySearch"
        >
          <UInput
            v-model="searchInput"
            icon="i-tabler-search"
            :placeholder="t('schedule.search_all_placeholder')"
            class="min-w-0 flex-1"
          />
          <UButton type="submit" color="neutral" variant="soft" icon="i-tabler-search">
            {{ t('schedule.search_action') }}
          </UButton>
        </form>

        <LibrarySelectionToolbar
          v-if="selectedRows.length"
          :label="t('schedule.selected_count', { n: selectedRows.length })"
          :hint="t('batchMetadata.selection_hint')"
          :clear-label="t('schedule.clear_selection')"
          @clear="clearSelection"
        >
          <UButton
            size="sm"
            variant="soft"
            icon="i-tabler-category-plus"
            :disabled="batchBusy"
            data-testid="schedule-batch-metadata"
            @click="openBatchEdit"
          >
            {{ t('schedule.batch_edit') }}
          </UButton>
          <template #destructive>
            <UButton
              size="sm"
              color="error"
              variant="ghost"
              icon="i-tabler-trash"
              :loading="batchBusy"
              @click="deleteSelected"
            >
              {{ t('schedule.batch_delete') }}
            </UButton>
          </template>
        </LibrarySelectionToolbar>

        <div v-else class="flex flex-wrap items-center gap-2 p-3">
          <AdaptiveSelect
            v-model="statusFilter"
            :items="statusItems"
            icon="i-tabler-toggle-right"
            @update:model-value="queryChanged"
          />
          <AdaptiveSelect
            v-model="categoryFilter"
            :items="categoryFilterItems"
            icon="i-tabler-category"
            @update:model-value="queryChanged"
          />
          <UInputMenu
            v-model="tagFilters"
            :items="tagOptions"
            multiple
            class="min-w-56 max-w-md flex-1"
            icon="i-tabler-tags"
            :placeholder="t('schedule.all_tags')"
            @update:model-value="queryChanged"
          />
          <AdaptiveSelect
            v-model="createdRange"
            :items="createdRangeItems"
            icon="i-tabler-calendar-plus"
            @update:model-value="queryChanged"
          />
          <AdaptiveSelect
            v-model="updatedRange"
            :items="updatedRangeItems"
            icon="i-tabler-calendar-stats"
            @update:model-value="queryChanged"
          />
          <AdaptiveSelect
            v-model="sort"
            :items="sortItems"
            icon="i-tabler-arrows-sort"
            @update:model-value="queryChanged"
          />
          <UDropdownMenu :items="columnMenuItems">
            <UButton
              color="neutral"
              variant="soft"
              icon="i-tabler-columns-3"
              trailing-icon="i-tabler-chevron-down"
              :label="t('schedule.columns')"
            />
          </UDropdownMenu>
          <UButton
            v-if="hasFilters"
            color="neutral"
            variant="ghost"
            icon="i-tabler-filter-x"
            :label="t('schedule.reset_filters')"
            @click="resetFilters"
          />
        </div>
      </section>

      <div class="workspace-surface min-h-0 flex-1 overflow-auto border-x border-default">
        <div v-if="loading" class="space-y-px p-2" :aria-label="t('schedule.loading')">
          <USkeleton v-for="index in 10" :key="index" class="h-14 rounded-md" />
        </div>
        <EmptyState
          v-else-if="pageSchedules.length === 0"
          inset
          :icon="hasFilters ? 'i-tabler-filter-off' : 'i-tabler-calendar-off'"
          :title="t(hasFilters ? 'schedule.no_results' : 'schedule.empty')"
          :description="t(hasFilters ? 'schedule.no_results_hint' : 'schedule.empty_desc')"
        >
          <template #action>
            <UButton
              v-if="hasFilters"
              color="neutral"
              variant="soft"
              icon="i-tabler-filter-x"
              :label="t('schedule.reset_filters')"
              @click="resetFilters"
            />
            <UButton v-else icon="i-tabler-plus" :label="t('schedule.create')" @click="onCreate" />
          </template>
        </EmptyState>
        <ScheduleListPanel
          v-else
          :list="pageSchedules"
          :workflows="workflows"
          :running-id="runningId"
          :visible-columns="visibleColumns"
          :grid-template-columns="scheduleGridTemplate"
          :selected="selected"
          :all-selected="allCurrentPageSelected"
          @edit="onEdit"
          @delete="onDelete"
          @toggle="onToggle"
          @run="onRun"
          @repair="onRepair"
          @select="toggleSchedule"
          @select-page="toggleCurrentPage"
        />
      </div>

      <footer
        v-if="!loading && filteredSchedules.length > 0"
        class="workspace-surface flex min-h-14 shrink-0 items-center gap-4 rounded-b-lg border border-default px-3"
      >
        <p class="mr-auto text-xs text-dimmed">
          {{
            t('schedule.result_range', {
              start: resultStart,
              end: resultEnd,
              total: filteredSchedules.length,
            })
          }}
        </p>
        <UPagination
          :page="page"
          :total="filteredSchedules.length"
          :items-per-page="pageSize"
          :sibling-count="1"
          active-variant="subtle"
          show-edges
          @update:page="goToPage"
        />
        <span class="text-xs text-dimmed">{{ t('schedule.per_page') }}</span>
        <AdaptiveSelect
          v-model="pageSize"
          :items="pageSizeItems"
          class="w-24"
          width-mode="fixed"
          @update:model-value="queryChanged"
        />
      </footer>
    </main>

    <BaseModal
      :open="!!editing"
      :title="editing?.name ?? t('schedule.create')"
      icon="i-tabler-calendar-time"
      size="3xl"
      tall
      @update:open="(open) => !open && (editing = null)"
    >
      <ScheduleEditorPanel
        v-if="editing"
        :schedule="editing"
        :workflows="workflows"
        :category-options="categoryOptions"
        :tag-options="tagOptions"
        @save="onSaveEdit"
        @cancel="editing = null"
      />
    </BaseModal>

    <BaseModal
      v-model:open="batchEditing"
      :title="t('schedule.batch_edit_title', { n: selectedRows.length })"
      icon="i-tabler-category-plus"
      size="lg"
      :dismissible="!batchBusy"
    >
      <div class="space-y-5">
        <p class="text-sm text-muted">{{ t('batchMetadata.description') }}</p>
        <UFormField :label="t('common.category')">
          <div class="flex items-center gap-2">
            <AdaptiveSelect
              v-model="batchDraft.categoryMode"
              :items="categoryModeItems"
              class="w-36 shrink-0"
              width-mode="fixed"
            />
            <UInputMenu
              v-if="batchDraft.categoryMode === 'set'"
              v-model="batchDraft.category"
              :items="batchCategoryOptions"
              :create-item="'always'"
              class="min-w-0 flex-1"
              @create="createBatchCategory"
            />
          </div>
        </UFormField>
        <UFormField :label="t('common.tags')">
          <div class="flex items-start gap-2">
            <AdaptiveSelect
              v-model="batchDraft.tagMode"
              :items="tagModeItems"
              class="w-36 shrink-0"
              width-mode="fixed"
            />
            <UInputMenu
              v-if="tagModeNeedsValues"
              v-model="batchDraft.tags"
              :items="batchTagOptions"
              :create-item="'always'"
              multiple
              class="min-w-0 flex-1"
              @create="createBatchTag"
            />
          </div>
        </UFormField>
      </div>
      <template #footer>
        <UButton
          color="neutral"
          variant="ghost"
          :disabled="batchBusy"
          @click="batchEditing = false"
        >
          {{ t('common.cancel') }}
        </UButton>
        <UButton
          icon="i-tabler-check"
          :loading="batchBusy"
          :disabled="!batchDraftValid"
          @click="applyBatchEdit"
        >
          {{ t('batchMetadata.apply') }}
        </UButton>
      </template>
    </BaseModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, shallowRef, toRaw, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useToast } from '@nuxt/ui/composables'
import { useSchedulesStore } from '@/stores/schedules'
import { useConfirm } from '@/composables/useConfirm'
import { errorMessage } from '@/lib/invoke'
import {
  applyBatchMetadata,
  createBatchMetadataDraft,
  hasBatchMetadataChange,
  uniqueMetadataValues,
} from '@/lib/batchMetadata'
import type { Schedule } from '@/lib/backend'
import { workflowTransport, type SourceView } from '@/app/transport/workflow'
import { readinessOutcome, runReadinessMessage } from '@/app/run/runReadiness'
import ScheduleListPanel from '@/components/schedules/ScheduleListPanel.vue'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import ScheduleEditorPanel from '@/components/schedules/ScheduleEditorPanel.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LibrarySelectionToolbar from '@/components/library/LibrarySelectionToolbar.vue'

type ScheduleColumn =
  | 'category'
  | 'tags'
  | 'trigger'
  | 'targets'
  | 'createdAt'
  | 'updatedAt'
  | 'lastFiredAt'
type DateRange = 'all' | 'today' | '7d' | '30d' | '90d'

const defaultColumns: ScheduleColumn[] = [
  'category',
  'tags',
  'trigger',
  'targets',
  'createdAt',
  'updatedAt',
]
const allCategories = '__all__'
const { t } = useI18n()
const router = useRouter()
const store = useSchedulesStore()
const toast = useToast()
const { confirm } = useConfirm()
const editing = shallowRef<Schedule | null>(null)
const workflows = ref<SourceView[]>([])
const runningId = ref('')
const loading = ref(true)
const searchInput = ref('')
const search = ref('')
const statusFilter = ref<'all' | 'enabled' | 'disabled'>('all')
const categoryFilter = ref(allCategories)
const tagFilters = ref<string[]>([])
const createdRange = ref<DateRange>('all')
const updatedRange = ref<DateRange>('all')
const sort = ref('updated_desc')
const page = ref(1)
const pageSize = ref(20)
const visibleColumns = ref<ScheduleColumn[]>(loadColumns())
const selected = ref<Record<string, Schedule>>({})
const batchEditing = ref(false)
const batchBusy = ref(false)
const batchDraft = reactive(createBatchMetadataDraft())
const createdCategories = ref<string[]>([])
const createdTags = ref<string[]>([])

const categories = computed(() =>
  facetValues(store.list.flatMap((schedule) => [schedule.category ?? ''])),
)
const categoryOptions = computed(() => categories.value.map((item) => item.value))
const tags = computed(() => facetValues(store.list.flatMap((schedule) => schedule.tags ?? [])))
const tagOptions = computed(() => tags.value.map((item) => item.value))
const selectedRows = computed(() => Object.values(selected.value))
const categoryFilterItems = computed(() => [
  { label: t('schedule.all_categories'), value: allCategories },
  ...categories.value.map((item) => ({
    label: `${item.value} (${item.count})`,
    value: item.value,
  })),
])
const statusItems = computed(() => [
  { label: t('schedule.filter.all'), value: 'all' },
  { label: t('schedule.filter.enabled'), value: 'enabled' },
  { label: t('schedule.filter.disabled'), value: 'disabled' },
])
const createdRangeItems = computed(() => dateRangeItems('created'))
const updatedRangeItems = computed(() => dateRangeItems('updated'))
const sortItems = computed(() => [
  { label: t('schedule.sort_updated_desc'), value: 'updated_desc' },
  { label: t('schedule.sort_created_desc'), value: 'created_desc' },
  { label: t('schedule.sort_name_asc'), value: 'name_asc' },
  { label: t('schedule.sort_name_desc'), value: 'name_desc' },
  { label: t('schedule.sort_last_desc'), value: 'last_desc' },
])
const pageSizeItems = [
  { label: '20', value: 20 },
  { label: '50', value: 50 },
  { label: '100', value: 100 },
]
const columnOptions = computed<Array<{ key: ScheduleColumn; label: string }>>(() => [
  { key: 'category', label: t('common.category') },
  { key: 'tags', label: t('common.tags') },
  { key: 'trigger', label: t('schedule.table.trigger') },
  { key: 'targets', label: t('schedule.table.targets') },
  { key: 'createdAt', label: t('schedule.table.created') },
  { key: 'updatedAt', label: t('schedule.table.updated') },
  { key: 'lastFiredAt', label: t('schedule.table.last') },
])
const visibleColumnSet = computed(() => new Set(visibleColumns.value))
const columnMenuItems = computed(() => [
  columnOptions.value.map((column) => ({
    label: column.label,
    type: 'checkbox' as const,
    checked: visibleColumnSet.value.has(column.key),
    onUpdateChecked: (checked: boolean) => setColumnVisible(column.key, checked),
  })),
  [
    {
      label: t('schedule.reset_columns'),
      icon: 'i-tabler-restore',
      onSelect: () => {
        visibleColumns.value = [...defaultColumns]
      },
    },
  ],
])
const hasFilters = computed(() =>
  Boolean(
    search.value ||
    statusFilter.value !== 'all' ||
    categoryFilter.value !== allCategories ||
    tagFilters.value.length ||
    createdRange.value !== 'all' ||
    updatedRange.value !== 'all',
  ),
)
const filteredSchedules = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  const createdSince = rangeStart(createdRange.value)
  const updatedSince = rangeStart(updatedRange.value)
  return [...store.list]
    .filter((schedule) => {
      if (statusFilter.value === 'enabled' && !schedule.enabled) return false
      if (statusFilter.value === 'disabled' && schedule.enabled) return false
      if (
        categoryFilter.value !== allCategories &&
        (schedule.category ?? '').toLocaleLowerCase() !== categoryFilter.value.toLocaleLowerCase()
      )
        return false
      const scheduleTags = new Set((schedule.tags ?? []).map((tag) => tag.toLocaleLowerCase()))
      if (tagFilters.value.some((tag) => !scheduleTags.has(tag.toLocaleLowerCase()))) return false
      if (createdSince && Date.parse(schedule.createdAt) < createdSince) return false
      if (updatedSince && Date.parse(schedule.updatedAt) < updatedSince) return false
      if (!query) return true
      return [
        schedule.name,
        schedule.description,
        schedule.category,
        schedule.id,
        ...(schedule.tags ?? []),
      ]
        .filter(Boolean)
        .some((value) => value!.toLocaleLowerCase().includes(query))
    })
    .sort(compareSchedules)
})
const pageCount = computed(() =>
  Math.max(1, Math.ceil(filteredSchedules.value.length / pageSize.value)),
)
const pageSchedules = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredSchedules.value.slice(start, start + pageSize.value)
})
const resultStart = computed(() =>
  filteredSchedules.value.length ? (page.value - 1) * pageSize.value + 1 : 0,
)
const resultEnd = computed(() =>
  Math.min(page.value * pageSize.value, filteredSchedules.value.length),
)
const allCurrentPageSelected = computed(
  () =>
    pageSchedules.value.length > 0 &&
    pageSchedules.value.every((schedule) => selected.value[schedule.id]),
)
const scheduleGridTemplate = computed(() => {
  const columns = ['2rem', 'minmax(14rem, 2fr)']
  if (visibleColumnSet.value.has('category')) columns.push('minmax(7rem, 0.8fr)')
  if (visibleColumnSet.value.has('tags')) columns.push('minmax(10rem, 1.2fr)')
  if (visibleColumnSet.value.has('trigger')) columns.push('minmax(9rem, 1fr)')
  if (visibleColumnSet.value.has('targets')) columns.push('minmax(10rem, 1.2fr)')
  if (visibleColumnSet.value.has('createdAt')) columns.push('7.75rem')
  if (visibleColumnSet.value.has('updatedAt')) columns.push('7.75rem')
  if (visibleColumnSet.value.has('lastFiredAt')) columns.push('7.75rem')
  columns.push('6rem', '4.5rem')
  return columns.join(' ')
})
const categoryModeItems = computed(() => [
  { label: t('batchMetadata.keep'), value: 'keep' },
  { label: t('batchMetadata.set'), value: 'set' },
  { label: t('batchMetadata.clear'), value: 'clear' },
])
const tagModeItems = computed(() => [
  { label: t('batchMetadata.keep'), value: 'keep' },
  { label: t('batchMetadata.add'), value: 'add' },
  { label: t('batchMetadata.remove'), value: 'remove' },
  { label: t('batchMetadata.replace'), value: 'replace' },
  { label: t('batchMetadata.clear'), value: 'clear' },
])
const tagModeNeedsValues = computed(() => ['add', 'remove', 'replace'].includes(batchDraft.tagMode))
const batchDraftValid = computed(() => hasBatchMetadataChange(batchDraft))
const batchCategoryOptions = computed(() =>
  uniqueStrings([...categoryOptions.value, ...createdCategories.value, batchDraft.category]),
)
const batchTagOptions = computed(() =>
  uniqueStrings([...tagOptions.value, ...createdTags.value, ...batchDraft.tags]),
)

watch(
  visibleColumns,
  (value) => localStorage.setItem('yotta.schedule.columns', JSON.stringify(value)),
  { deep: true },
)
watch(pageCount, (count) => {
  if (page.value > count) page.value = count
})

onMounted(async () => {
  loading.value = true
  try {
    const [, sources] = await Promise.all([store.reload(), workflowTransport.listSources()])
    workflows.value = sources
  } catch (error) {
    showError(t('workflow.toast.list_failed'), error)
  } finally {
    loading.value = false
  }
})

async function onCreate() {
  try {
    editing.value = await store.createDraft(
      t('schedule.create_default_name', { n: store.list.length + 1 }),
    )
  } catch (error) {
    showError(t('toast.operation_failed'), error)
  }
}

function onEdit(schedule: Schedule) {
  editing.value = structuredClone(toRaw(schedule))
}

async function onSaveEdit(schedule: Schedule) {
  try {
    await store.save(schedule)
    editing.value = null
  } catch (error) {
    showError(t('toast.save_failed'), error)
  }
}

async function onToggle(schedule: Schedule, enabled: boolean) {
  try {
    await store.update(schedule.id, { enabled })
  } catch (error) {
    showError(t('toast.operation_failed'), error)
  }
}

async function onRun(schedule: Schedule) {
  if (runningId.value) return
  runningId.value = schedule.id
  try {
    const result = await store.fireNow(schedule.id)
    if (result.status === 'queued') return
    toast.add({
      title: t('schedule.run_not_started'),
      description: runReadinessMessage(readinessOutcome(result.readiness)),
      color: 'warning',
    })
  } catch (error) {
    showError(t('schedule.run_failed'), error)
  } finally {
    runningId.value = ''
  }
}

function onRepair(schedule: Schedule) {
  const readiness = schedule.lastReadiness
  const workflowId = readiness?.workflowId ?? schedule.targets[0]?.id
  if (!workflowId) return
  const query: Record<string, string> = {}
  if (readiness?.graphId) query.focusGraphPath = readiness.graphId
  if (readiness?.nodeId) query.focusNode = readiness.nodeId
  void router.push({ path: `/workflows/${workflowId}/edit`, query })
}

async function onDelete(schedule: Schedule) {
  const yes = await confirm({
    title: t('schedule.delete_title'),
    description: t('schedule.delete_desc', { name: schedule.name }),
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  try {
    await store.remove(schedule.id)
    const next = { ...selected.value }
    delete next[schedule.id]
    selected.value = next
  } catch (error) {
    showError(t('toast.operation_failed'), error)
  }
}

function applySearch(): void {
  search.value = searchInput.value.trim()
  page.value = 1
}

function queryChanged(): void {
  page.value = 1
}

function resetFilters(): void {
  searchInput.value = ''
  search.value = ''
  statusFilter.value = 'all'
  categoryFilter.value = allCategories
  tagFilters.value = []
  createdRange.value = 'all'
  updatedRange.value = 'all'
  page.value = 1
}

function goToPage(next: number): void {
  if (next < 1 || next > pageCount.value) return
  page.value = next
}

function toggleSchedule(schedule: Schedule, checked: boolean): void {
  const next = { ...selected.value }
  if (checked) next[schedule.id] = schedule
  else delete next[schedule.id]
  selected.value = next
}

function toggleCurrentPage(checked: boolean): void {
  const next = { ...selected.value }
  for (const schedule of pageSchedules.value) {
    if (checked) next[schedule.id] = schedule
    else delete next[schedule.id]
  }
  selected.value = next
}

function clearSelection(): void {
  selected.value = {}
}

function openBatchEdit(): void {
  Object.assign(batchDraft, createBatchMetadataDraft())
  batchEditing.value = true
}

async function applyBatchEdit(): Promise<void> {
  if (!batchDraftValid.value || batchBusy.value) return
  batchBusy.value = true
  try {
    await store.updateMany(
      selectedRows.value.map((schedule) => {
        const metadata = applyBatchMetadata(
          { category: schedule.category ?? '', tags: schedule.tags ?? [] },
          batchDraft,
        )
        return { id: schedule.id, patch: metadata }
      }),
    )
    clearSelection()
    batchEditing.value = false
  } catch (error) {
    showError(t('toast.operation_failed'), error)
  } finally {
    batchBusy.value = false
  }
}

async function deleteSelected(): Promise<void> {
  const yes = await confirm({
    title: t('schedule.batch_delete_title', { n: selectedRows.value.length }),
    description: t('schedule.batch_delete_desc'),
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  batchBusy.value = true
  try {
    await store.removeMany(selectedRows.value.map((schedule) => schedule.id))
    clearSelection()
  } catch (error) {
    showError(t('toast.operation_failed'), error)
  } finally {
    batchBusy.value = false
  }
}

function createBatchCategory(value: string): void {
  const category = value.trim()
  if (!category) return
  createdCategories.value = uniqueStrings([...createdCategories.value, category])
  batchDraft.category = category
}

function createBatchTag(value: string): void {
  const tag = value.trim()
  if (!tag) return
  createdTags.value = uniqueStrings([...createdTags.value, tag])
  batchDraft.tags = uniqueMetadataValues([...batchDraft.tags, tag])
}

function setColumnVisible(column: ScheduleColumn, visible: boolean): void {
  const current = new Set(visibleColumns.value)
  if (visible) current.add(column)
  else current.delete(column)
  visibleColumns.value = columnOptions.value
    .map((item) => item.key)
    .filter((key) => current.has(key))
}

function loadColumns(): ScheduleColumn[] {
  try {
    const raw = JSON.parse(localStorage.getItem('yotta.schedule.columns') ?? 'null')
    if (!Array.isArray(raw)) return [...defaultColumns]
    const allowed = new Set<ScheduleColumn>([
      'category',
      'tags',
      'trigger',
      'targets',
      'createdAt',
      'updatedAt',
      'lastFiredAt',
    ])
    const values = raw.filter((value): value is ScheduleColumn => allowed.has(value))
    return values.length ? values : [...defaultColumns]
  } catch {
    return [...defaultColumns]
  }
}

function dateRangeItems(kind: 'created' | 'updated') {
  const prefix = kind === 'created' ? 'created' : 'updated'
  return [
    { label: t(`schedule.${prefix}_any`), value: 'all' },
    { label: t(`schedule.${prefix}_today`), value: 'today' },
    { label: t(`schedule.${prefix}_days`, { n: 7 }), value: '7d' },
    { label: t(`schedule.${prefix}_days`, { n: 30 }), value: '30d' },
    { label: t(`schedule.${prefix}_days`, { n: 90 }), value: '90d' },
  ]
}

function rangeStart(range: DateRange): number {
  if (range === 'all') return 0
  const start = new Date()
  if (range === 'today') start.setHours(0, 0, 0, 0)
  else start.setDate(start.getDate() - Number.parseInt(range, 10))
  return start.getTime()
}

function compareSchedules(left: Schedule, right: Schedule): number {
  if (sort.value === 'name_asc') return left.name.localeCompare(right.name)
  if (sort.value === 'name_desc') return right.name.localeCompare(left.name)
  if (sort.value === 'created_desc') return Date.parse(right.createdAt) - Date.parse(left.createdAt)
  if (sort.value === 'last_desc')
    return Date.parse(right.lastFiredAt ?? '') - Date.parse(left.lastFiredAt ?? '')
  return Date.parse(right.updatedAt) - Date.parse(left.updatedAt)
}

function facetValues(values: string[]): Array<{ value: string; count: number }> {
  const facets = new Map<string, { value: string; count: number }>()
  for (const raw of values) {
    const value = raw.trim()
    const key = value.toLocaleLowerCase()
    if (!key) continue
    const current = facets.get(key)
    if (current) current.count += 1
    else facets.set(key, { value, count: 1 })
  }
  return [...facets.values()].sort((left, right) => left.value.localeCompare(right.value))
}

function uniqueStrings(values: string[]): string[] {
  return uniqueMetadataValues(values)
}

function showError(title: string, error: unknown): void {
  toast.add({ title, description: errorMessage(error), color: 'error' })
}
</script>
