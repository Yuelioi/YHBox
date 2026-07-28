<template>
  <section
    class="flex h-full min-h-0 flex-col bg-default"
    data-testid="workflow-resource-dock"
    :data-resource-kind="props.kind"
    :data-resource-scope="scope"
  >
    <header class="shrink-0 border-b border-default px-3 py-2.5">
      <div class="flex items-start gap-2">
        <div class="min-w-0 flex-1">
          <h2 class="text-xs font-semibold text-highlighted">{{ resourceTitle }}</h2>
          <p class="mt-1 text-[10px] leading-4 text-muted">{{ resourceHint }}</p>
        </div>
        <UButton
          icon="i-tabler-external-link"
          color="neutral"
          variant="ghost"
          size="xs"
          :aria-label="t('workflow.resources.open_library')"
          @click="emit('open-library')"
        />
      </div>

      <div
        class="mt-2 grid grid-cols-2 gap-1 rounded-lg border border-default bg-elevated/40 p-1"
        role="group"
        :aria-label="t('workflow.resources.title')"
      >
        <UButton
          v-for="candidate in scopeItems"
          :key="candidate.value"
          size="xs"
          :data-testid="`workflow-resource-scope-${candidate.value}`"
          :data-active="scope === candidate.value"
          :color="scope === candidate.value ? 'primary' : 'neutral'"
          :variant="scope === candidate.value ? 'soft' : 'ghost'"
          :class="[
            'min-w-0 justify-center px-1.5 transition-colors',
            scope === candidate.value
              ? 'ring-1 ring-inset ring-primary/45'
              : 'text-muted hover:text-highlighted',
          ]"
          :icon="candidate.icon"
          :label="candidate.label"
          :aria-pressed="scope === candidate.value"
          @click="scope = candidate.value"
        />
      </div>

      <div class="mt-2 flex items-center gap-2">
        <UButton
          v-if="props.kind !== 'template'"
          data-testid="workflow-resource-create"
          :icon="props.kind === 'macro' ? 'i-tabler-list-details' : 'i-tabler-route-alt-left'"
          :label="
            props.kind === 'macro'
              ? t('assets.recording.record_macro')
              : t('assets.recording.record_precise')
          "
          size="xs"
          class="min-w-0 flex-1 justify-center"
          :disabled="recordingPhase !== 'idle'"
          @click="emit('start-recording', props.kind === 'macro' ? 'simple' : 'precise')"
        />
        <UButton
          v-else
          data-testid="workflow-resource-create"
          icon="i-tabler-camera-plus"
          :label="t('assets.templates.capture')"
          size="xs"
          class="min-w-0 flex-1 justify-center"
          @click="emit('capture-template')"
        />
        <UButton
          v-if="scope === 'library'"
          icon="i-tabler-refresh"
          color="neutral"
          variant="ghost"
          size="xs"
          :loading="loading"
          :aria-label="t('common.refresh')"
          @click="loadLibrary(true)"
        />
      </div>

      <UInput
        v-model="searchInput"
        icon="i-tabler-search"
        size="sm"
        class="mt-2 w-full"
        :placeholder="t('assets.search_placeholder')"
        :aria-label="t('assets.search_action')"
      />
      <div data-testid="workflow-resource-filter-row" class="mt-2 flex min-w-0 gap-2">
        <AdaptiveSelect
          v-model="category"
          data-testid="workflow-resource-filter-category"
          :items="categoryItems"
          icon="i-tabler-category"
          size="sm"
          width-mode="fill"
          class="min-w-0 flex-1"
          :aria-label="t('common.category')"
          @update:model-value="changeQuery"
        />
        <AdaptiveSelect
          v-model="sort"
          data-testid="workflow-resource-filter-sort"
          :items="sortItems"
          icon="i-tabler-arrows-sort"
          size="sm"
          width-mode="fill"
          class="min-w-0 flex-1"
          :aria-label="t('workflow.list.sort_label')"
          @update:model-value="changeQuery"
        />
      </div>
      <UInputMenu
        v-model="tagFilters"
        :items="tagOptions"
        multiple
        icon="i-tabler-tags"
        size="sm"
        class="mt-2 w-full"
        :placeholder="t('assets.all_tags')"
        :aria-label="t('assets.tags_filter')"
        @update:model-value="changeQuery"
      />
    </header>

    <div
      v-if="visibleItems.length"
      data-testid="workflow-resource-selection-bar"
      class="flex h-9 shrink-0 items-center gap-2 border-b border-default px-3 text-[10px] text-muted"
      :class="selectedRows.length ? 'bg-primary/5' : ''"
      role="toolbar"
      :aria-label="t('workflow.resources.select_page')"
    >
      <UCheckbox
        :model-value="allCurrentPageSelected"
        :indeterminate="someCurrentPageSelected && !allCurrentPageSelected"
        :aria-label="t('assets.select_page')"
        @update:model-value="toggleCurrentPage(Boolean($event))"
      />
      <span class="min-w-0 flex-1 truncate">
        {{
          selectedRows.length
            ? t('workflow.resources.selected_count', { n: selectedRows.length })
            : t('workflow.resources.select_page')
        }}
      </span>
      <span v-if="!selectedRows.length" class="font-mono text-dimmed">{{
        visibleItems.length
      }}</span>
      <UButton
        v-if="selectedRows.length"
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-tabler-category-plus"
        :disabled="busy"
        :aria-label="t('assets.batch_edit')"
        :title="t('assets.batch_edit')"
        @click="openBatchEdit"
      />
      <UButton
        v-if="selectedRows.length"
        size="xs"
        color="error"
        variant="ghost"
        icon="i-tabler-trash"
        :loading="busy"
        :aria-label="t('assets.batch_delete')"
        :title="t('assets.batch_delete')"
        @click="deleteSelected"
      />
      <UButton
        v-if="selectedRows.length"
        size="xs"
        color="neutral"
        variant="ghost"
        icon="i-tabler-x"
        :aria-label="t('assets.clear_selection')"
        :title="t('assets.clear_selection')"
        @click="clearSelection"
      />
    </div>

    <div
      v-if="feedback"
      class="shrink-0 border-b px-3 py-2 text-[10px]"
      :class="
        feedback.tone === 'error'
          ? 'border-error/30 bg-error/10 text-error'
          : feedback.tone === 'warning'
            ? 'border-warning/30 bg-warning/10 text-warning'
            : 'border-success/30 bg-success/10 text-success'
      "
      role="status"
    >
      {{ feedback.message }}
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto p-2">
      <div v-if="loading" data-testid="workflow-resource-loading" class="space-y-2">
        <USkeleton v-for="index in 8" :key="index" class="h-16 rounded-lg" />
      </div>
      <AssetLibraryList
        v-else-if="visibleItems.length"
        :items="libraryItems"
        compact
        :draggable="scope === 'library'"
        :focused-id="focusedResourceId"
        @use="useLibraryItem"
        @dragstart="startResourceDrag"
      >
        <template #select="{ item }">
          <UCheckbox
            :model-value="Boolean(selected[item.id])"
            :aria-label="t('workflow.resources.select_named', { name: item.name })"
            @click.stop
            @update:model-value="toggleItem(item.id, Boolean($event))"
          />
        </template>
        <template #actions="{ item }">
          <div class="flex shrink-0 items-center gap-0.5">
            <UButton
              icon="i-tabler-plus"
              color="neutral"
              variant="ghost"
              size="xs"
              :disabled="busy"
              :aria-label="t('workflow.resources.use', { name: item.name })"
              @click.stop="useLibraryItem(item)"
            />
            <UDropdownMenu :items="itemMenu(item.id)">
              <UButton
                icon="i-tabler-dots"
                color="neutral"
                variant="ghost"
                size="xs"
                :aria-label="t('assets.asset_actions', { name: item.name })"
                @click.stop
              />
            </UDropdownMenu>
          </div>
        </template>
      </AssetLibraryList>
      <EmptyState
        v-else
        inset
        :icon="
          props.kind === 'macro'
            ? 'i-tabler-list-details'
            : props.kind === 'clip'
              ? 'i-tabler-route-alt-left'
              : 'i-tabler-photo-off'
        "
        :title="hasFilters ? t('assets.no_results') : t('workflow.resources.empty')"
        :description="hasFilters ? t('assets.no_results_hint') : t('workflow.resources.empty_hint')"
      />
    </div>

    <footer
      v-if="total > 0"
      class="flex min-h-11 shrink-0 flex-wrap items-center gap-2 border-t border-default px-3 py-1.5"
    >
      <span class="mr-auto text-[10px] text-dimmed">
        {{ t('assets.result_range', { start: resultStart, end: resultEnd, total }) }}
      </span>
      <AdaptiveSelect
        v-model="pageSize"
        :items="pageSizeItems"
        size="xs"
        width-mode="fixed"
        class="w-16"
        @update:model-value="changeQuery"
      />
      <UPagination
        :page="page"
        :total="total"
        :items-per-page="pageSize"
        :sibling-count="1"
        show-edges
        size="xs"
        @update:page="goToPage"
      />
    </footer>
  </section>

  <BaseModal
    :open="!!editing"
    :title="t('assets.edit_title')"
    icon="i-tabler-edit"
    size="md"
    @update:open="(open) => !open && (editing = null)"
  >
    <div class="space-y-3">
      <UFormField :label="t('common.name')" required>
        <UInput v-model="editDraft.name" maxlength="256" />
      </UFormField>
      <UFormField :label="t('common.description')">
        <UTextarea v-model="editDraft.description" :rows="2" />
      </UFormField>
      <UFormField :label="t('common.category')">
        <UInput v-model="editDraft.category" />
      </UFormField>
      <UFormField :label="t('common.tags')">
        <UInputMenu v-model="editDraft.tags" :items="tagOptions" :create-item="'always'" multiple />
      </UFormField>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="editing = null">{{
        t('common.cancel')
      }}</UButton>
      <UButton :loading="busy" :disabled="!editDraft.name.trim()" @click="saveEdit">{{
        t('common.save')
      }}</UButton>
    </template>
  </BaseModal>

  <BaseModal
    v-model:open="batchEditing"
    :title="t('workflow.resources.batch_edit_title', { n: selectedRows.length })"
    icon="i-tabler-tags"
    size="md"
  >
    <div class="space-y-3">
      <p class="text-xs leading-5 text-muted">{{ t('workflow.resources.batch_edit_hint') }}</p>
      <UFormField :label="t('common.category')">
        <UInput v-model="batchDraft.category" />
      </UFormField>
      <UFormField :label="t('common.tags')">
        <UInputMenu
          v-model="batchDraft.tags"
          :items="tagOptions"
          :create-item="'always'"
          multiple
        />
      </UFormField>
    </div>
    <template #footer>
      <UButton color="neutral" variant="ghost" @click="batchEditing = false">{{
        t('common.cancel')
      }}</UButton>
      <UButton :loading="busy" @click="saveBatchEdit">{{ t('common.save') }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AdaptiveSelect from '@/components/common/AdaptiveSelect.vue'
import AssetLibraryList, {
  type AssetLibraryListItem,
} from '@/components/assets/AssetLibraryList.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { backend, type AssetSummary } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'
import { useConfirm } from '@/composables/useConfirm'
import { useAssetsStore, type AssetPickerSelection } from '@/stores/assets'
import type {
  WorkflowResource,
  YottaWorkflowSource,
} from '../../../../contracts/workflow/current/workflow-source'
import { RESOURCE_DRAG_FORMAT, serializeWorkspaceResource } from './resourceDrag'
import { snapshotGlobalAsset } from './workflowResourceSnapshot'
import {
  projectWorkflowResourcePage,
  workflowResourceReferenceCount,
} from './workflowResourceLibrary'
import type { ResourceLocateRequest } from './resourceLocator'

type ResourceKind = 'macro' | 'clip' | 'template'
type ResourceScope = 'workflow' | 'library'
type EditableResource =
  | { scope: 'workflow'; resource: WorkflowResource }
  | { scope: 'library'; asset: AssetSummary }

const props = defineProps<{
  kind: ResourceKind
  source: YottaWorkflowSource
  recordingPhase: 'idle' | 'armed' | 'countdown' | 'recording' | 'paused' | 'finalizing' | 'pending'
  locateRequest?: ResourceLocateRequest | null
}>()
const emit = defineEmits<{
  'start-recording': [mode: 'simple' | 'precise']
  'capture-template': []
  'open-library': []
  edit: [asset: AssetSummary]
  use: [selection: AssetPickerSelection]
  'use-workflow': [resource: WorkflowResource, variantId: string]
  'import-workflow-resource': [resource: WorkflowResource]
  'edit-workflow-resource': [resource: WorkflowResource]
  'duplicate-workflow-resource': [resource: WorkflowResource]
  'update-workflow-resources': [
    payloads: Array<{
      resourceId: string
      name: string
      description: string
      category: string
      tags: string[]
    }>,
  ]
  'remove-workflow-resources': [resourceIds: string[]]
}>()
const { t } = useI18n()
const { confirm } = useConfirm()
const assets = useAssetsStore()
const allCategoriesValue = '__yotta_all_categories__'
const scope = ref<ResourceScope>('workflow')
const searchInput = ref('')
const search = ref('')
const category = ref(allCategoriesValue)
const tagFilters = ref<string[]>([])
const sort = ref('name_asc')
const page = ref(1)
const pageSize = ref(20)
const libraryTotal = ref(0)
const libraryAssets = ref<AssetSummary[]>([])
const libraryCategories = ref<Array<{ value: string; count: number }>>([])
const libraryTags = ref<Array<{ value: string; count: number }>>([])
const loading = ref(false)
const busy = ref(false)
const selected = ref<Record<string, WorkflowResource | AssetSummary>>({})
const editing = ref<EditableResource | null>(null)
const batchEditing = ref(false)
const editDraft = reactive({ name: '', description: '', category: '', tags: [] as string[] })
const batchDraft = reactive({ category: '', tags: [] as string[] })
const feedback = ref<{ tone: 'success' | 'warning' | 'error'; message: string } | null>(null)
const focusedResourceId = ref('')
let requestGeneration = 0
let searchTimer: ReturnType<typeof setTimeout> | undefined
let applyingLocate = false

const scopeItems = computed(() => [
  {
    label: t('workflow.resources.current_workflow'),
    value: 'workflow' as const,
    icon: 'i-tabler-file-code',
  },
  {
    label: t('workflow.resources.local_library'),
    value: 'library' as const,
    icon: 'i-tabler-database',
  },
])
const resourceTitle = computed(() =>
  t(
    props.kind === 'macro'
      ? 'assets.tabs.macros'
      : props.kind === 'clip'
        ? 'assets.tabs.clips'
        : 'assets.tabs.templates',
  ),
)
const resourceHint = computed(() => t(`workflow.resources.${props.kind}_hint`))
const workflowResourceKind = computed(() =>
  props.kind === 'template' ? 'image' : props.kind === 'clip' ? 'input-clip' : 'macro',
)
const workflowResources = computed(() =>
  props.source.resources.filter((resource) => resource.kind === workflowResourceKind.value),
)
const workflowPage = computed(() =>
  projectWorkflowResourcePage(workflowResources.value, {
    search: search.value,
    category: category.value,
    allCategoriesValue,
    tags: tagFilters.value,
    sort: sort.value === 'name_desc' ? 'name_desc' : 'name_asc',
    page: page.value,
    pageSize: pageSize.value,
  }),
)
const visibleItems = computed<Array<WorkflowResource | AssetSummary>>(() =>
  scope.value === 'workflow' ? workflowPage.value.items : libraryAssets.value,
)
const total = computed(() =>
  scope.value === 'workflow' ? workflowPage.value.total : libraryTotal.value,
)
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const resultStart = computed(() => (total.value ? (page.value - 1) * pageSize.value + 1 : 0))
const resultEnd = computed(() => Math.min(page.value * pageSize.value, total.value))
const selectedRows = computed(() => Object.values(selected.value))
const allCurrentPageSelected = computed(
  () =>
    visibleItems.value.length > 0 &&
    visibleItems.value.every((value) => Boolean(selected.value[sourceID(value)])),
)
const someCurrentPageSelected = computed(() =>
  visibleItems.value.some((value) => Boolean(selected.value[sourceID(value)])),
)
const libraryItems = computed<AssetLibraryListItem[]>(() =>
  visibleItems.value.map((value) =>
    isWorkflowResource(value)
      ? workflowListItem(value)
      : {
          id: value.guid,
          name: value.name,
          description: value.description ?? '',
          category: value.category ?? '',
          tags: value.tags ?? [],
          meta: assetMeta(value),
          icon: assetIcon(value),
          previewBlob: value.thumbnail,
        },
  ),
)
const categoryCounts = computed(() =>
  scope.value === 'workflow' ? workflowPage.value.categories : libraryCategories.value,
)
const tagCounts = computed(() =>
  scope.value === 'workflow' ? workflowPage.value.tags : libraryTags.value,
)
const categoryItems = computed(() => [
  { label: t('assets.all_categories'), value: allCategoriesValue },
  ...categoryCounts.value.map((item) => ({
    label: `${item.value} (${item.count})`,
    value: item.value,
  })),
])
const tagOptions = computed(() => tagCounts.value.map((item) => item.value))
const sortItems = computed(() => [
  { label: t('assets.sort_name_asc'), value: 'name_asc' },
  { label: t('assets.sort_name_desc'), value: 'name_desc' },
  ...(scope.value === 'library'
    ? [{ label: t('assets.sort_created_desc'), value: 'created_desc' }]
    : []),
])
const pageSizeItems = [
  { label: '20', value: 20 },
  { label: '50', value: 50 },
  { label: '100', value: 100 },
]
const hasFilters = computed(() =>
  Boolean(search.value || category.value !== allCategoriesValue || tagFilters.value.length),
)

onMounted(() => void loadLibrary())
onBeforeUnmount(() => {
  requestGeneration += 1
  if (searchTimer) clearTimeout(searchTimer)
})
watch(
  () => props.kind,
  () => resetView(),
)
watch(scope, () => {
  if (!applyingLocate) resetView()
})
watch(searchInput, (value) => {
  if (applyingLocate) return
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    search.value = value.trim()
    changeQuery()
  }, 250)
})
watch(
  () => [props.locateRequest?.requestId, props.kind] as const,
  () => void applyLocateRequest(),
  { immediate: true },
)
watch(
  () => assets.epoch,
  () => scope.value === 'library' && void loadLibrary(true),
)
watch(pageCount, (count) => {
  if (page.value > count) page.value = count
})

async function loadLibrary(force = false): Promise<void> {
  if (scope.value !== 'library') return
  const generation = ++requestGeneration
  loading.value = true
  try {
    const result = await assets.query(
      {
        search: search.value,
        kind: props.kind,
        category: category.value === allCategoriesValue ? '' : category.value,
        tags: [...tagFilters.value],
        sort: sort.value,
        page: page.value,
        pageSize: pageSize.value,
        thumbnailBudget: props.kind === 'template' ? 12 : 0,
        recentGUIDs: assets.recentGUIDs,
      },
      { force },
    )
    if (generation !== requestGeneration) return
    libraryAssets.value = result.items
    libraryTotal.value = result.total
    libraryCategories.value = result.categories
    libraryTags.value = result.tags
  } catch (error) {
    showFeedback('error', errorMessage(error))
  } finally {
    if (generation === requestGeneration) loading.value = false
  }
}

function resetView(): void {
  category.value = allCategoriesValue
  tagFilters.value = []
  sort.value = 'name_asc'
  page.value = 1
  focusedResourceId.value = ''
  clearSelection()
  if (scope.value === 'library') void loadLibrary()
}

function changeQuery(): void {
  page.value = 1
  focusedResourceId.value = ''
  clearSelection()
  if (scope.value === 'library') void loadLibrary()
}

async function applyLocateRequest(): Promise<void> {
  const request = props.locateRequest
  if (!request || request.kind !== props.kind || !request.id) return
  applyingLocate = true
  if (searchTimer) clearTimeout(searchTimer)
  scope.value = request.scope
  searchInput.value = request.id
  await nextTick()
  search.value = request.id
  category.value = allCategoriesValue
  tagFilters.value = []
  sort.value = 'name_asc'
  page.value = 1
  clearSelection()
  focusedResourceId.value = request.id
  applyingLocate = false

  if (request.scope === 'library') await loadLibrary(true)
  const found = visibleItems.value.some((value) => sourceID(value) === request.id)
  if (found) {
    showFeedback('success', t('workflow.resources.located', { id: request.id }))
  } else {
    focusedResourceId.value = ''
    showFeedback('warning', t('workflow.resources.locate_failed', { id: request.id }))
  }
}

function goToPage(next: number): void {
  if (next < 1 || next > pageCount.value || next === page.value) return
  page.value = next
  if (scope.value === 'library') void loadLibrary()
}

function clearSelection(): void {
  selected.value = {}
}

function toggleItem(id: string, checked: boolean): void {
  const next = { ...selected.value }
  const value = visibleSourceByID(id)
  if (checked && value) next[id] = value
  else delete next[id]
  selected.value = next
}

function toggleCurrentPage(checked: boolean): void {
  const next = { ...selected.value }
  for (const value of visibleItems.value) {
    const id = sourceID(value)
    if (checked) next[id] = value
    else delete next[id]
  }
  selected.value = next
}

function useLibraryItem(item: AssetLibraryListItem): void {
  if (busy.value) return
  if (scope.value === 'workflow') {
    const resource = workflowResources.value.find((candidate) => candidate.id === item.id)
    if (resource) {
      const variantId = resource.kind === 'image' ? (resource.image?.variants[0]?.id ?? '') : ''
      emit('use-workflow', resource, variantId)
    }
    return
  }
  const asset = libraryAssets.value.find((candidate) => candidate.guid === item.id)
  if (asset) void importAsset(asset)
}

async function importAsset(asset: AssetSummary): Promise<void> {
  busy.value = true
  try {
    emit('import-workflow-resource', await snapshotGlobalAsset(asset))
    assets.markUsed(asset.guid)
    showFeedback('success', t('workflow.resources.snapshot_created'))
  } catch (error) {
    showFeedback('error', errorMessage(error))
  } finally {
    busy.value = false
  }
}

function itemMenu(id: string) {
  const value = visibleSourceByID(id)
  if (!value) return []
  const name = isWorkflowResource(value) ? value.name : value.name
  return [
    [
      {
        label: t('common.edit'),
        icon: 'i-tabler-edit',
        onSelect: () => {
          if (value.kind !== 'macro') {
            openEdit(value)
            return
          }
          if (isWorkflowResource(value)) emit('edit-workflow-resource', value)
          else emit('edit', value)
        },
      },
      ...(isWorkflowResource(value)
        ? [
            ...(value.kind === 'input-clip'
              ? [
                  {
                    label: t('workflow.resources.edit_content'),
                    icon: 'i-tabler-route-alt-left',
                    onSelect: () => emit('edit-workflow-resource', value),
                  },
                ]
              : []),
            {
              label: t('workflow.resources.duplicate'),
              icon: 'i-tabler-copy',
              onSelect: () => void duplicateWorkflowResource(value),
            },
            {
              label: t('workflow.resources.promote'),
              icon: 'i-tabler-library-plus',
              onSelect: () => void promoteWorkflowResource(value),
            },
          ]
        : []),
    ],
    [
      {
        label: t('common.delete'),
        icon: 'i-tabler-trash',
        color: 'error' as const,
        disabled: isWorkflowResource(value) && referenceCount(value.id) > 0,
        onSelect: () => void deleteOne(value, name),
      },
    ],
  ]
}

async function duplicateWorkflowResource(resource: WorkflowResource): Promise<void> {
  if (busy.value) return
  busy.value = true
  try {
    const duplicate = await backend.workflowResources.duplicate(resource)
    emit('duplicate-workflow-resource', duplicate)
    showFeedback('success', t('workflow.resources.duplicated', { name: resource.name }))
  } catch (error) {
    showFeedback('error', errorMessage(error))
  } finally {
    busy.value = false
  }
}

async function promoteWorkflowResource(resource: WorkflowResource): Promise<void> {
  if (busy.value) return
  busy.value = true
  try {
    await backend.workflowResources.promote(resource)
    assets.invalidate()
    showFeedback('success', t('workflow.resources.promoted', { name: resource.name }))
  } catch (error) {
    showFeedback('error', errorMessage(error))
  } finally {
    busy.value = false
  }
}

function openEdit(value: WorkflowResource | AssetSummary): void {
  editing.value = isWorkflowResource(value)
    ? { scope: 'workflow', resource: value }
    : { scope: 'library', asset: value }
  editDraft.name = value.name
  editDraft.description = value.description ?? ''
  editDraft.category = value.category ?? ''
  editDraft.tags = [...(value.tags ?? [])]
}

async function saveEdit(): Promise<void> {
  const target = editing.value
  if (!target || !editDraft.name.trim()) return
  busy.value = true
  try {
    if (target.scope === 'workflow') {
      emit('update-workflow-resources', [
        {
          resourceId: target.resource.id,
          name: editDraft.name.trim(),
          description: editDraft.description.trim(),
          category: editDraft.category.trim(),
          tags: uniqueStrings(editDraft.tags),
        },
      ])
    } else {
      await backend.assets.updateMeta(
        target.asset.guid,
        editDraft.name.trim(),
        editDraft.description.trim(),
        editDraft.category.trim(),
        uniqueStrings(editDraft.tags),
      )
      assets.invalidate()
      await loadLibrary(true)
    }
    editing.value = null
    showFeedback('success', t('workflow.resources.saved'))
  } catch (error) {
    showFeedback('error', errorMessage(error))
  } finally {
    busy.value = false
  }
}

function openBatchEdit(): void {
  batchDraft.category = ''
  batchDraft.tags = []
  batchEditing.value = true
}

async function saveBatchEdit(): Promise<void> {
  if (!selectedRows.value.length) return
  busy.value = true
  try {
    const categoryValue = batchDraft.category.trim()
    const tags = uniqueStrings(batchDraft.tags)
    if (scope.value === 'workflow') {
      emit(
        'update-workflow-resources',
        selectedRows.value
          .filter((value): value is WorkflowResource => isWorkflowResource(value))
          .map((resource) => ({
            resourceId: resource.id,
            name: resource.name,
            description: resource.description ?? '',
            category: categoryValue,
            tags,
          })),
      )
    } else {
      await backend.assets.batchUpdateMeta(
        selectedRows.value
          .filter((value): value is AssetSummary => !isWorkflowResource(value))
          .map((asset) => ({ guid: asset.guid, category: categoryValue, tags })),
      )
      assets.invalidate()
      await loadLibrary(true)
    }
    batchEditing.value = false
    clearSelection()
    showFeedback('success', t('workflow.resources.batch_saved'))
  } catch (error) {
    showFeedback('error', errorMessage(error))
  } finally {
    busy.value = false
  }
}

async function deleteOne(value: WorkflowResource | AssetSummary, name: string): Promise<void> {
  if (isWorkflowResource(value) && referenceCount(value.id) > 0) return
  const accepted = await confirm({
    title: t('assets.delete_title', { name }),
    description: t('workflow.resources.delete_description'),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  busy.value = true
  try {
    if (isWorkflowResource(value)) emit('remove-workflow-resources', [value.id])
    else {
      await backend.assets.delete_(value.guid)
      assets.invalidate()
      await loadLibrary(true)
    }
    showFeedback('success', t('workflow.resources.deleted'))
  } catch (error) {
    showFeedback('error', errorMessage(error))
  } finally {
    busy.value = false
  }
}

async function deleteSelected(): Promise<void> {
  if (!selectedRows.value.length) return
  const blocked =
    scope.value === 'workflow'
      ? selectedRows.value.filter(
          (value) => isWorkflowResource(value) && referenceCount(value.id) > 0,
        ).length
      : 0
  const accepted = await confirm({
    title: t('workflow.resources.batch_delete_title', { n: selectedRows.value.length }),
    description: t('workflow.resources.batch_delete_description', {
      deletable: selectedRows.value.length - blocked,
      blocked,
    }),
    confirmText: t('common.delete'),
    color: 'error',
  })
  if (accepted !== true) return
  busy.value = true
  try {
    if (scope.value === 'workflow') {
      const ids = selectedRows.value
        .filter((value): value is WorkflowResource => isWorkflowResource(value))
        .filter((resource) => referenceCount(resource.id) === 0)
        .map((resource) => resource.id)
      if (ids.length) emit('remove-workflow-resources', ids)
    } else {
      await backend.assets.batchDelete(
        selectedRows.value
          .filter((value): value is AssetSummary => !isWorkflowResource(value))
          .map((asset) => asset.guid),
      )
      assets.invalidate()
      await loadLibrary(true)
    }
    clearSelection()
    showFeedback(
      blocked ? 'warning' : 'success',
      t('workflow.resources.batch_deleted', { blocked }),
    )
  } catch (error) {
    showFeedback('error', errorMessage(error))
  } finally {
    busy.value = false
  }
}

function startResourceDrag(event: DragEvent, item: AssetLibraryListItem): void {
  if (scope.value !== 'library') return
  const asset = libraryAssets.value.find((candidate) => candidate.guid === item.id)
  if (!asset || !event.dataTransfer) return
  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData(RESOURCE_DRAG_FORMAT, serializeWorkspaceResource(asset.guid))
}

function visibleSourceByID(id: string): WorkflowResource | AssetSummary | undefined {
  return scope.value === 'workflow'
    ? workflowResources.value.find((resource) => resource.id === id)
    : libraryAssets.value.find((asset) => asset.guid === id)
}

function sourceID(value: WorkflowResource | AssetSummary): string {
  return isWorkflowResource(value) ? value.id : value.guid
}

function workflowListItem(resource: WorkflowResource): AssetLibraryListItem {
  const references = referenceCount(resource.id)
  const summary = workflowResourceSummary(resource)
  const referenceSummary = references
    ? t('workflow.resources.reference_count', { n: references })
    : t('workflow.resources.unused')
  return {
    id: resource.id,
    name: resource.name,
    description: [resource.description ?? '', summary, referenceSummary]
      .filter(Boolean)
      .join(' · '),
    category: resource.category ?? '',
    tags: resource.tags ?? [],
    meta: `${summary} · ${referenceSummary}`,
    icon: assetIconForKind(resource.kind),
    previewBlob: resource.kind === 'image' ? resource.image?.variants[0]?.blob : undefined,
  }
}

function workflowResourceSummary(resource: WorkflowResource): string {
  if (resource.kind === 'image') {
    const variant = resource.image?.variants[0]
    return t('workflow.resources.image_summary', {
      variants: resource.image?.variants.length ?? 0,
      width: variant?.resolution[0] ?? 0,
      height: variant?.resolution[1] ?? 0,
    })
  }
  if (resource.kind === 'macro') {
    return t('workflow.resources.macro_summary', {
      actions: resource.macro?.actionCount ?? 0,
      duration: formatResourceDuration(resource.macro?.durationUs ?? 0),
    })
  }
  return t('workflow.resources.input_clip_summary', {
    events: resource.inputClip?.eventCount ?? 0,
    duration: formatResourceDuration(resource.inputClip?.durationUs ?? 0),
    counts: resource.inputClip?.mouseCounts360 || '—',
  })
}

function formatResourceDuration(durationUs: number): string {
  return `${(Math.max(0, durationUs) / 1_000_000).toFixed(2)}s`
}

function referenceCount(resourceId: string): number {
  return workflowResourceReferenceCount(props.source, resourceId)
}

function isWorkflowResource(value: WorkflowResource | AssetSummary): value is WorkflowResource {
  return 'id' in value
}

function uniqueStrings(values: string[]): string[] {
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))].sort()
}

function assetIcon(asset: AssetSummary): string {
  return assetIconForKind(asset.kind)
}

function assetIconForKind(kind: string): string {
  if (kind === 'template' || kind === 'image') return 'i-tabler-photo'
  if (kind === 'clip' || kind === 'input-clip') return 'i-tabler-route-alt-left'
  return 'i-tabler-list-details'
}

function assetMeta(asset: AssetSummary): string {
  if (asset.kind === 'template') return t('assets.templates.meta', { count: asset.variantCount })
  if (asset.kind === 'clip') return t('assetPicker.clip_size', { size: asset.blob?.size ?? 0 })
  return t('assetPicker.macro_size', { size: asset.blob?.size ?? 0 })
}

function showFeedback(tone: 'success' | 'warning' | 'error', message: string): void {
  feedback.value = { tone, message }
}
</script>
