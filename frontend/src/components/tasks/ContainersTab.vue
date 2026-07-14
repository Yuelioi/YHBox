<template>
  <div
    data-testid="containers-shell"
    class="flex h-full min-h-0 flex-1 flex-col gap-3 overflow-hidden"
  >
    <section class="workspace-metrics" :aria-label="t('containers.workspace.summary')">
      <div class="workspace-metric workspace-metric--primary">
        <span>{{ t('containers.workspace.total') }}</span>
        <strong>{{ store.list.length }}</strong>
      </div>
      <div class="workspace-metric">
        <span>{{ t('containers.workspace.running') }}</span>
        <strong>{{ runningCount }}</strong>
      </div>
      <div class="workspace-metric">
        <span>{{ t('containers.workspace.nodes') }}</span>
        <strong>{{ totalNodeCount }}</strong>
      </div>
      <div class="workspace-metric">
        <span>{{ t('containers.workspace.categories') }}</span>
        <strong>{{ categoryCount }}</strong>
      </div>
    </section>

    <header class="flex shrink-0 flex-col gap-2">
      <div data-testid="containers-toolbar" class="flex min-w-0 items-center justify-between gap-3">
        <UInput
          v-model="search"
          icon="i-tabler-search"
          :placeholder="t('containers.search_placeholder')"
          size="sm"
          class="min-w-0 flex-1 sm:max-w-sm"
        />

        <div class="flex shrink-0 items-center justify-end gap-2">
          <UButton
            size="sm"
            color="neutral"
            :variant="filtersOpen || activeFilterCount > 0 ? 'soft' : 'ghost'"
            icon="i-tabler-filter"
            :aria-expanded="filtersOpen"
            @click="filtersOpen = !filtersOpen"
          >
            <span class="hidden sm:inline">{{ t('containers.filters') }}</span>
            <UBadge v-if="activeFilterCount" size="xs" color="primary" variant="solid">{{
              activeFilterCount
            }}</UBadge>
          </UButton>
          <div class="view-switcher">
            <UButton
              size="sm"
              color="neutral"
              :variant="viewMode === 'cards' ? 'soft' : 'ghost'"
              icon="i-tabler-layout-grid"
              :aria-pressed="viewMode === 'cards'"
              :aria-label="t('containers.view.cards')"
              :title="t('containers.view.cards')"
              @click="viewMode = 'cards'"
            />
            <UButton
              size="sm"
              color="neutral"
              :variant="viewMode === 'list' ? 'soft' : 'ghost'"
              icon="i-tabler-list"
              :aria-pressed="viewMode === 'list'"
              :aria-label="t('containers.view.list')"
              :title="t('containers.view.list')"
              @click="viewMode = 'list'"
            />
          </div>
          <UButton
            color="primary"
            icon="i-tabler-plus"
            :aria-label="t('containers.create')"
            @click="onCreate"
          >
            <span class="hidden sm:inline">{{ t('containers.create') }}</span>
          </UButton>
        </div>
      </div>

      <div
        v-show="filtersOpen || activeFilterCount > 0"
        data-testid="containers-filterbar"
        class="container-filterbar flex shrink-0 flex-wrap items-center gap-2"
      >
        <UInputMenu
          v-model="selectedTags"
          multiple
          :items="allTagItems"
          icon="i-tabler-tags"
          size="xs"
          class="w-56"
          :placeholder="t('containers.filter_tags')"
        />
        <USelect
          v-model="categoryFilter"
          :items="categoryItems"
          icon="i-tabler-category"
          size="xs"
          class="w-40 shrink-0"
          :aria-label="t('containers.filter_category')"
        />
        <USelect
          v-model="sortKey"
          :items="sortItems"
          icon="i-tabler-arrows-sort"
          size="xs"
          class="w-40 shrink-0"
          :aria-label="t('containers.sort.label')"
        />
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          class="shrink-0"
          :icon="sortDesc ? 'i-tabler-sort-descending' : 'i-tabler-sort-ascending'"
          :aria-label="sortDirectionLabel"
          :title="sortDirectionLabel"
          @click="sortDesc = !sortDesc"
        />
        <UDropdownMenu v-if="viewMode === 'list'" :items="columnMenuItems">
          <UButton
            data-testid="containers-column-selector"
            size="xs"
            variant="ghost"
            color="neutral"
            class="shrink-0"
            icon="i-tabler-columns-3"
            trailing-icon="i-tabler-chevron-down"
            :aria-label="t('containers.columns.label')"
            :title="t('containers.columns.label')"
          />
        </UDropdownMenu>
      </div>
    </header>

    <main data-testid="containers-content" class="min-h-0 flex-1 overflow-y-auto pr-1">
      <EmptyState
        v-if="store.list.length === 0"
        icon="i-tabler-schema"
        :title="t('containers.empty_title')"
        :description="t('containers.empty_desc')"
      >
        <template #action>
          <UButton color="primary" icon="i-tabler-plus" @click="onCreate">
            {{ t('containers.empty_cta') }}
          </UButton>
        </template>
      </EmptyState>

      <EmptyState
        v-else-if="pageResult.total === 0"
        icon="i-tabler-filter-off"
        :title="t('containers.no_match_title')"
        :description="t('containers.no_match_desc')"
      >
        <template #action>
          <UButton color="neutral" variant="soft" icon="i-tabler-refresh" @click="resetFilters">
            {{ t('containers.reset_filters') }}
          </UButton>
        </template>
      </EmptyState>

      <div
        v-else-if="viewMode === 'cards'"
        class="grid gap-3 pb-1"
        style="grid-template-columns: repeat(auto-fill, minmax(min(260px, 100%), 1fr))"
      >
        <AppCard
          v-for="c in visibleContainers"
          :key="c.id"
          padding="panel"
          hover
          class="container-card group relative flex min-h-52 flex-col gap-4"
          :class="batch.isSelected(c.id) ? '!border-primary ring-2 ring-primary/40' : ''"
          @dblclick="onEdit(c)"
        >
          <UCheckbox
            :data-testid="`container-checkbox-${c.id}`"
            :model-value="batch.isSelected(c.id)"
            size="sm"
            class="absolute top-2 left-2"
            :aria-label="t('containers.batch_actions.select_one', { name: c.name })"
            @click.stop
            @update:model-value="batch.toggle(c.id)"
          />
          <div class="flex min-w-0 items-start gap-3 pl-7">
            <span class="container-card__icon">
              <UIcon name="i-tabler-route-square-2" class="size-5" />
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex items-center justify-between gap-2">
                <h3 class="truncate text-sm font-semibold text-highlighted">
                  {{ c.name || t('common.untitled') }}
                </h3>
                <StatusPill
                  :status="isRunning(c.id) ? 'online' : 'ready'"
                  :label="
                    isRunning(c.id) ? t('containers.status.running') : t('containers.status.idle')
                  "
                  :dot="isRunning(c.id)"
                  class="shrink-0"
                />
              </div>
              <p class="mt-1 line-clamp-2 min-h-8 text-xs leading-relaxed text-dimmed">
                {{ c.description || t('containers.workspace.no_description') }}
              </p>
            </div>
          </div>

          <div class="flex min-h-6 items-center gap-1.5 overflow-hidden">
            <span
              v-if="c.category"
              class="inline-flex shrink-0 items-center gap-1 rounded-md bg-elevated px-2 py-1 text-xs text-toned"
            >
              <UIcon name="i-tabler-category" class="size-3" />
              {{ c.category }}
            </span>
            <span
              v-for="tag in (c.tags ?? []).slice(0, 2)"
              :key="tag"
              class="shrink-0 rounded-md bg-primary/8 px-2 py-1 text-xs text-primary"
            >
              {{ tag }}
            </span>
            <span v-if="(c.tags?.length ?? 0) > 2" class="text-xs text-dimmed">
              +{{ (c.tags?.length ?? 0) - 2 }}
            </span>
          </div>

          <div class="container-card__facts">
            <span>
              <UIcon name="i-tabler-route" class="size-3.5" />
              {{ t('containers.node_count', { n: c.graph.nodes.length }) }}
            </span>
            <span>
              <UIcon name="i-tabler-history" class="size-3.5" />
              {{ t('containers.workspace.updated', { value: formatContainerDate(c.updatedAt) }) }}
            </span>
            <span v-if="c.hotkey">
              <UIcon name="i-tabler-keyboard" class="size-3.5" />
              <code>{{ c.hotkey }}</code>
            </span>
          </div>

          <div class="mt-auto flex items-center gap-1.5 border-t border-default/60 pt-3">
            <UButton
              v-if="!isRunning(c.id)"
              size="sm"
              color="primary"
              variant="soft"
              icon="i-tabler-player-play"
              @click.stop="onRun(c)"
              >{{ t('containers.run') }}</UButton
            >
            <UButton
              v-else
              size="sm"
              color="error"
              variant="soft"
              icon="i-tabler-square"
              @click.stop="onStop()"
              >{{ t('containers.stop') }}</UButton
            >
            <UButton
              size="sm"
              variant="ghost"
              color="neutral"
              icon="i-tabler-edit"
              @click.stop="onEdit(c)"
              >{{ t('containers.edit') }}</UButton
            >
            <div class="flex-1" />
            <UDropdownMenu :items="rowMenuItems(c)">
              <UButton
                size="sm"
                variant="ghost"
                color="neutral"
                icon="i-tabler-dots-vertical"
                :aria-label="t('containers.actions.more_for', { name: c.name })"
                @click.stop
              />
            </UDropdownMenu>
          </div>
        </AppCard>
      </div>

      <div v-else class="overflow-x-auto rounded-lg border border-default/60">
        <div data-testid="containers-list-table" class="w-full">
          <div
            data-testid="containers-list-header"
            class="sticky top-0 z-10 grid items-center gap-3 border-b border-default/60 bg-default px-3 py-2 text-xs font-medium text-muted"
            :style="{ gridTemplateColumns: listGridTemplate }"
          >
            <span />
            <span>{{ t('containers.list.name') }}</span>
            <span
              v-for="column in activeListColumns"
              :key="column.key"
              :class="column.align === 'right' ? 'text-right' : ''"
            >
              {{ column.label }}
            </span>
            <span class="text-right">{{ t('containers.list.actions') }}</span>
          </div>
          <div
            v-for="c in visibleContainers"
            :key="c.id"
            :data-testid="`container-row-${c.id}`"
            class="grid items-center gap-3 border-b border-default/40 px-3 py-2 last:border-b-0"
            :class="[
              'cursor-pointer hover:bg-elevated/40',
              batch.isSelected(c.id) ? 'bg-primary/10 ring-1 ring-inset ring-primary/40' : '',
            ]"
            :style="{ gridTemplateColumns: listGridTemplate }"
            @click="batch.toggle(c.id)"
            @dblclick="onEdit(c)"
          >
            <div class="flex items-center justify-center">
              <UCheckbox
                :data-testid="`container-checkbox-${c.id}`"
                :model-value="batch.isSelected(c.id)"
                size="sm"
                :aria-label="t('containers.batch_actions.select_one', { name: c.name })"
                @click.stop
                @update:model-value="batch.toggle(c.id)"
              />
            </div>
            <div class="min-w-0">
              <div class="truncate text-sm font-medium text-highlighted">
                {{ c.name || t('common.untitled') }}
              </div>
              <div v-if="c.description" class="mt-0.5 truncate text-xs text-dimmed">
                {{ c.description }}
              </div>
            </div>
            <StatusPill
              v-if="isColumnVisible('status')"
              :status="isRunning(c.id) ? 'online' : 'ready'"
              :label="
                isRunning(c.id) ? t('containers.status.running') : t('containers.status.idle')
              "
              :dot="isRunning(c.id)"
              class="w-fit"
            />
            <span v-if="isColumnVisible('category')" class="truncate text-xs text-toned">
              {{ c.category || '-' }}
            </span>
            <div v-if="isColumnVisible('tags')" class="flex min-w-0 flex-wrap gap-1">
              <span
                v-for="tag in (c.tags ?? []).slice(0, 4)"
                :key="tag"
                class="rounded bg-elevated/60 px-1.5 py-0.5 text-[11px] text-dimmed"
              >
                {{ tag }}
              </span>
              <span v-if="!c.tags || c.tags.length === 0" class="text-xs text-dimmed">-</span>
            </div>
            <span
              v-if="isColumnVisible('nodes')"
              class="font-mono text-xs tabular-nums text-toned"
              >{{ containerNodeCount(c) }}</span
            >
            <span v-if="isColumnVisible('createdAt')" class="font-mono text-xs text-dimmed">{{
              formatContainerDate(c.createdAt)
            }}</span>
            <span v-if="isColumnVisible('updatedAt')" class="font-mono text-xs text-dimmed">{{
              formatContainerDate(c.updatedAt)
            }}</span>
            <span v-if="isColumnVisible('hotkey')" class="truncate text-xs text-dimmed">
              <code v-if="c.hotkey" class="rounded bg-elevated/60 px-1 font-mono text-toned">{{
                c.hotkey
              }}</code>
              <span v-else>-</span>
            </span>
            <div class="flex items-center justify-end gap-1">
              <UButton
                v-if="!isRunning(c.id)"
                :data-testid="`container-row-run-${c.id}`"
                size="xs"
                color="primary"
                variant="soft"
                icon="i-tabler-player-play"
                :aria-label="t('containers.run')"
                @click.stop="onRun(c)"
              />
              <UButton
                v-else
                :data-testid="`container-row-run-${c.id}`"
                size="xs"
                color="error"
                variant="soft"
                icon="i-tabler-square"
                :aria-label="t('containers.stop')"
                @click.stop="onStop()"
              />
              <UDropdownMenu :items="rowMenuItems(c)">
                <UButton
                  :data-testid="`container-row-more-${c.id}`"
                  size="xs"
                  variant="ghost"
                  color="neutral"
                  icon="i-tabler-dots"
                  :aria-label="t('containers.actions.more_for', { name: c.name })"
                  @click.stop
                />
              </UDropdownMenu>
            </div>
          </div>
        </div>
      </div>
    </main>

    <footer
      v-if="store.list.length > 0"
      data-testid="containers-pagination"
      class="flex shrink-0 flex-col gap-3 border-t border-default/60 bg-default py-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <div class="flex flex-wrap items-center gap-2">
        <UDropdownMenu :items="batchMenuItems">
          <UButton
            data-testid="containers-batch-actions"
            size="xs"
            variant="soft"
            color="neutral"
            icon="i-tabler-stack-2"
            trailing-icon="i-tabler-chevron-down"
            :disabled="visibleContainers.length === 0"
          >
            {{ t('containers.batch_actions.menu') }}
          </UButton>
        </UDropdownMenu>
        <span v-if="batch.count.value > 0" class="text-xs text-toned">
          {{ t('containers.batch_actions.selected', { n: batch.count.value }) }}
        </span>
        <span class="text-xs text-dimmed">{{ pageTotalLabel }}</span>
      </div>
      <div class="flex items-center gap-2">
        <UPagination
          v-if="pageResult.totalPages > 1"
          v-model:page="page"
          :total="pageResult.total"
          :items-per-page="pageSize"
          :sibling-count="1"
          size="xs"
        />
        <USelect v-model="pageSize" :items="pageSizeItems" size="xs" class="w-28" />
      </div>
    </footer>
  </div>

  <BaseModal
    v-model:open="createDialogOpen"
    :title="t('containers.create_dialog.title')"
    icon="i-tabler-plus"
    size="md"
  >
    <div class="space-y-3">
      <UFormField :label="t('common.name')" required>
        <UInput
          v-model="createDraft.name"
          size="sm"
          :placeholder="t('containers.name_placeholder')"
          @keyup.enter="onConfirmCreate"
        />
      </UFormField>
      <UFormField :label="t('common.description')">
        <UTextarea
          v-model="createDraft.description"
          :rows="3"
          size="sm"
          :placeholder="t('containers.description_placeholder')"
        />
      </UFormField>
      <UFormField :label="t('common.category')">
        <UInputMenu
          v-model="createDraft.category"
          :items="allCategories"
          create-item
          size="sm"
          :placeholder="t('containers.category_placeholder')"
          @create="onCreateDraftCategory"
        />
      </UFormField>
    </div>

    <template #footer>
      <UButton size="sm" variant="ghost" color="neutral" @click="createDialogOpen = false">
        {{ t('common.cancel') }}
      </UButton>
      <UButton
        size="sm"
        color="primary"
        icon="i-tabler-check"
        :disabled="createName.length === 0"
        @click="onConfirmCreate"
      >
        {{ t('containers.create_dialog.confirm') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useLocalStorage } from '@vueuse/core'
import { useContainersStore } from '@/stores/containers'
import { useExecutionStore } from '@/stores/execution'
import { useBatchSelect } from '@/composables/useBatchSelect'
import { useConfirm } from '@/composables/useConfirm'
import { backend, type Container } from '@/lib/backend'
import { useRouter } from 'vue-router'
import AppCard from '@/components/common/AppCard.vue'
import StatusPill from '@/components/common/StatusPill.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import { addCreatedCategory, uniqueCategoryOptions } from '@/components/containers/categoryOptions'
import {
  buildContainerPage,
  containerCategories,
  containerNodeCount,
  containerTagsByCount,
  formatContainerDate,
  type ContainerSortKey,
} from '@/lib/containerList'

const { t } = useI18n()

const router = useRouter()
const store = useContainersStore()
const execStore = useExecutionStore()
const toast = useToast()
const search = ref('')
const sortKey = useLocalStorage<ContainerSortKey>('containers.sortKey', 'updatedAt')
const sortDesc = useLocalStorage('containers.sortDesc', true)
const viewMode = useLocalStorage<'cards' | 'list'>('containers.viewMode', 'cards')
const filtersOpen = useLocalStorage('containers.filtersOpen', false)
const page = ref(1)
const pageSize = useLocalStorage('containers.pageSize', 24)
const createDialogOpen = ref(false)
const createDraft = ref({
  name: '',
  description: '',
  category: '',
})
type ListColumnKey = 'status' | 'category' | 'tags' | 'nodes' | 'createdAt' | 'updatedAt' | 'hotkey'
type ListColumn = {
  key: ListColumnKey
  label: string
  width: string
  align?: 'right'
}
const defaultListColumns: ListColumnKey[] = ['status', 'category', 'tags', 'nodes', 'hotkey']
const listColumns = useLocalStorage<ListColumnKey[]>('containers.listColumns', defaultListColumns)

// 批量删除（E.5）
const batch = useBatchSelect()
const { confirm } = useConfirm()

const runningCount = computed(
  () => store.list.filter((container) => isRunning(container.id)).length,
)
const totalNodeCount = computed(() =>
  store.list.reduce((total, container) => total + containerNodeCount(container), 0),
)
const categoryCount = computed(() => containerCategories(store.list).length)

const sortItems = computed(() => [
  { label: t('containers.sort.name'), value: 'name' },
  { label: t('containers.sort.created_at'), value: 'createdAt' },
  { label: t('containers.sort.updated_at'), value: 'updatedAt' },
  { label: t('containers.sort.nodes'), value: 'nodes' },
])
const sortDirectionLabel = computed(() =>
  sortDesc.value ? t('containers.sort.desc') : t('containers.sort.asc'),
)
const pageSizeItems = computed(() =>
  [12, 24, 48, 96].map((n) => ({ label: t('containers.pagination.per_page', { n }), value: n })),
)
const createName = computed(() => createDraft.value.name.trim())
const visibleContainerIDs = computed(() => visibleContainers.value.map((c) => c.id))
const batchMenuItems = computed(() => [
  [
    {
      label: t('containers.batch_actions.select_page'),
      icon: 'i-tabler-checklist',
      disabled: visibleContainerIDs.value.length === 0,
      onSelect: () => batch.selectAll(visibleContainerIDs.value),
    },
    {
      label: t('containers.batch_actions.clear'),
      icon: 'i-tabler-x',
      disabled: batch.count.value === 0,
      onSelect: () => batch.clear(),
    },
  ],
  [
    {
      label: t('containers.batch_actions.delete'),
      icon: 'i-tabler-trash',
      color: 'error' as const,
      disabled: batch.count.value === 0,
      onSelect: () => {
        void onBatchDelete()
      },
    },
  ],
])
const listColumnOptions = computed<ListColumn[]>(() => [
  { key: 'status', label: t('containers.list.status'), width: '96px' },
  { key: 'category', label: t('containers.list.category'), width: '120px' },
  { key: 'tags', label: t('containers.list.tags'), width: 'minmax(140px, 0.8fr)' },
  { key: 'nodes', label: t('containers.list.nodes'), width: '72px' },
  { key: 'createdAt', label: t('containers.list.created_at'), width: '132px' },
  { key: 'updatedAt', label: t('containers.list.updated_at'), width: '132px' },
  { key: 'hotkey', label: t('containers.list.hotkey'), width: '110px' },
])
const visibleColumnSet = computed(() => new Set(listColumns.value))
const activeListColumns = computed(() => {
  return listColumnOptions.value.filter((column) => visibleColumnSet.value.has(column.key))
})
const listGridTemplate = computed(() => {
  return [
    '40px',
    'minmax(240px, 1.4fr)',
    ...activeListColumns.value.map((column) => column.width),
    '84px',
  ].join(' ')
})
const columnMenuItems = computed(() => [
  listColumnOptions.value.map((column) => ({
    label: column.label,
    type: 'checkbox' as const,
    checked: visibleColumnSet.value.has(column.key),
    onUpdateChecked: (checked: boolean) => setColumnVisible(column.key, checked),
  })),
  [
    {
      label: t('containers.columns.reset'),
      icon: 'i-tabler-restore',
      onSelect: () => {
        listColumns.value = [...defaultListColumns]
      },
    },
  ],
])
const rowMenuItems = (c: Container) => [
  [
    { label: t('containers.edit'), icon: 'i-tabler-edit', onSelect: () => onEdit(c) },
    {
      label: t('containers.export'),
      icon: 'i-tabler-package-export',
      onSelect: () => {
        void onExport(c)
      },
    },
  ],
  [
    {
      label: t('containers.delete.title'),
      icon: 'i-tabler-trash',
      color: 'error' as const,
      onSelect: () => {
        void onAskDelete(c)
      },
    },
  ],
]

function isColumnVisible(key: ListColumnKey): boolean {
  return visibleColumnSet.value.has(key)
}

function setColumnVisible(key: ListColumnKey, checked: boolean) {
  const current = new Set(listColumns.value)
  if (checked) {
    current.add(key)
  } else {
    current.delete(key)
  }
  const orderedKeys = listColumnOptions.value.map((column) => column.key)
  listColumns.value = orderedKeys.filter((columnKey) => current.has(columnKey))
}

async function onBatchDelete() {
  const ids = [...batch.selected.value]
  if (ids.length === 0) return
  if (ids.some((id) => store.isRecordingLocked(id))) {
    toast.add({ title: t('containers.toast.recording_locked'), color: 'warning' })
    return
  }
  const yes = await confirm({
    title: t('containers.batch_delete.title'),
    description: t('containers.batch_delete.desc', { n: ids.length }),
    color: 'error',
    confirmText: t('containers.batch_delete.confirm'),
  })
  if (yes !== true) return
  const ok = await store.deleteMany(ids)
  if (!ok) {
    toast.add({ title: t('containers.toast.batch_partial_fail'), color: 'warning' })
  }
  batch.clear()
}
function isRunning(id: string): boolean {
  return execStore.running && execStore.currentTargetID === id
}

async function onRun(c: Container) {
  await store.run(c.id)
}

async function onStop() {
  await store.stopAll()
  toast.add({
    title: t('containers.toast.stop_signal'),
    color: 'neutral',
    icon: 'i-tabler-square',
  })
}

onMounted(() => {
  void store.reload()
})

const selectedTags = ref<string[]>([])
const categoryFilter = ref<string>('all')
const createdCategories = ref<string[]>([])
const activeFilterCount = computed(
  () => selectedTags.value.length + (categoryFilter.value === 'all' ? 0 : 1),
)

const tagsByCount = computed(() => {
  return containerTagsByCount(store.list ?? [])
})
const allTagItems = computed(() => tagsByCount.value.map(({ tag }) => tag))
const allCategories = computed(() => {
  return uniqueCategoryOptions(containerCategories(store.list ?? []), createdCategories.value, [
    createDraft.value.category,
  ])
})
const categoryItems = computed(() => [
  { label: t('containers.filter_category_all'), value: 'all' },
  ...allCategories.value.map((category) => ({ label: category, value: `cat:${category}` })),
])

function onCreateDraftCategory(item: string) {
  const result = addCreatedCategory(createdCategories.value, item)
  if (!result.value) return
  createdCategories.value = result.categories
  createDraft.value.category = result.value
}

const pageResult = computed(() =>
  buildContainerPage(store.list, {
    query: search.value,
    category: categoryFilter.value === 'all' ? null : categoryFilter.value.slice(4),
    tags: selectedTags.value,
    sortKey: sortKey.value,
    sortDesc: sortDesc.value,
    page: page.value,
    pageSize: pageSize.value,
  }),
)
const visibleContainers = computed(() => pageResult.value.pageItems)
const pageTotalLabel = computed(() => {
  if (pageResult.value.total === 0) return t('containers.pagination.empty')
  return t('containers.pagination.range', {
    start: pageResult.value.start,
    end: pageResult.value.end,
    total: pageResult.value.total,
  })
})

watch([search, categoryFilter, selectedTags, sortKey, sortDesc, viewMode, pageSize], () => {
  page.value = 1
})
watch(
  () => pageResult.value.page,
  (p) => {
    if (page.value !== p) page.value = p
  },
)

function resetFilters() {
  search.value = ''
  categoryFilter.value = 'all'
  selectedTags.value = []
}

function onCreate() {
  createDraft.value = {
    name: t('containers.create_default_name', { n: store.list.length + 1 }),
    description: '',
    category: '',
  }
  createDialogOpen.value = true
}

async function onConfirmCreate() {
  const name = createName.value
  if (!name) return
  const c = await store.create(name)
  if (!c) return
  const patch: Partial<Container> = {}
  const description = createDraft.value.description.trim()
  const category = createDraft.value.category.trim()
  if (description) patch.description = description
  if (category) patch.category = category
  if (Object.keys(patch).length > 0) {
    await store.update(c.id, patch)
  }
  createDialogOpen.value = false
  onEdit(c)
}

function onEdit(c: Container) {
  router.push(`/containers/${c.id}/edit`)
}

function exportFilename(c: Container): string {
  const base = (c.name || c.packageName || c.id || 'container')
    .trim()
    .replace(/[<>:"/\\|?*]+/g, '-')
    .replace(/[\p{Cc}]+/gu, '-')
    .replace(/\s+/g, '-')
    .replace(/^-+|-+$/g, '')
  return `${base || c.id || 'container'}.yotta-container.zip`
}

function withContainerZipExtension(path: string): string {
  return path.toLowerCase().endsWith('.yotta-container.zip') ? path : `${path}.yotta-container.zip`
}

async function onExport(c: Container) {
  const picked = await backend.containers.pickExportPath(
    exportFilename(c),
    t('containers.export_dialog_title'),
    t('containers.export_dialog_button'),
  )
  if (!picked) return
  const destPath = withContainerZipExtension(picked)
  const ok = await store.exportPackage(c.id, destPath)
  if (!ok) return
  toast.add({
    title: t('containers.toast.export_success'),
    description: destPath,
    color: 'success',
    icon: 'i-tabler-package-export',
  })
}

async function onAskDelete(c: Container) {
  if (store.isRecordingLocked(c.id)) {
    toast.add({ title: t('containers.toast.recording_locked'), color: 'warning' })
    return
  }
  const yes = await confirm({
    title: t('containers.delete.title'),
    description: `${t('containers.delete.desc_prefix')}${c.name}${t('containers.delete.desc_suffix')}`,
    color: 'error',
    confirmText: t('containers.delete.confirm'),
  })
  if (yes !== true) return
  await store.remove(c.id)
}
</script>
