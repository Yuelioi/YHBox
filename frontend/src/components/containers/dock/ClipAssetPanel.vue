<!-- clip 资产停靠面板. 主操作 = 放进画布: 行可拖到画布(落松手处) / 双击插到中心 (裸 PlayClip).
     单击=选中(批量); 右键/⋯=插入 + 详情(改名/标签等低频 → 按需弹小 modal) + 删除. -->
<template>
  <div class="asset-panel relative flex h-full min-h-0 flex-col" :data-workspace="workspace">
    <div v-if="drillIn && detailId && !workspace" class="flex h-full min-h-0 flex-col">
      <div class="flex shrink-0 items-center gap-2 border-b border-default px-3 py-2">
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-arrow-left"
          @click="drillIn = false"
        >
          {{ t('common.back') }}
        </UButton>
        <span class="min-w-0 flex-1 truncate text-sm font-medium text-highlighted">{{
          byId(detailId)?.label || detailId
        }}</span>
        <UButton size="xs" color="primary" icon="i-tabler-package-import" @click="onDetailInsert">
          {{ t('library.explorer.insert') }}
        </UButton>
      </div>
      <ClipDetailPanel :clip-id="detailId" @insert="onDetailInsert" />
    </div>

    <template v-else>
      <div
        class="flex shrink-0 items-center justify-between gap-3 border-b border-default px-3 py-2.5"
      >
        <div class="min-w-0">
          <p class="text-sm font-semibold text-highlighted">{{ t('assetBrowser.actionClips') }}</p>
          <p class="text-xs text-dimmed">{{ t('assetBrowser.clipSubtitle') }}</p>
        </div>
        <div class="flex shrink-0 items-center gap-2 text-xs text-dimmed">
          <UIcon name="i-tabler-grip-horizontal" class="size-4" />
          <span>{{ t('assetBrowser.dragToCanvas') }}</span>
        </div>
      </div>

      <div class="flex min-h-0 flex-1 overflow-hidden">
        <AssetCategoryRail
          v-if="workspace"
          v-model="categoryFilter"
          :items="categoryFilterItems"
          class="asset-category-rail"
        />
        <div class="flex min-w-0 flex-1 flex-col gap-2.5 overflow-hidden p-3">
          <AssetBrowserToolbar
            ref="toolbarRef"
            v-model:query="query"
            v-model:category="categoryFilter"
            v-model:tags="tagFilter"
            v-model:sort-key="sortKey"
            v-model:sort-desc="sortDesc"
            v-model:view="viewMode"
            :search-placeholder="t('clip.manager.search')"
            :category-items="categoryFilterItems"
            :tag-items="allTags"
            :sort-items="sortItems"
            allow-view-switch
            :show-search="!workspace"
            :show-category-scopes="!workspace"
          />

          <AssetSelectionBar
            :count="selected.size"
            :batch-items="batchMenuItems"
            @clear="selClear()"
          />

          <div data-asset-browser-list class="min-h-0 flex-1 overflow-y-auto pr-1 select-none">
            <div
              v-if="filteredItems.length === 0"
              class="flex min-h-56 flex-col items-center justify-center text-center"
            >
              <div class="mb-3 flex size-11 items-center justify-center rounded-lg bg-elevated/60">
                <UIcon name="i-tabler-player-record" class="size-5 text-muted" />
              </div>
              <p class="text-sm font-medium text-toned">
                <span v-if="entries.length === 0">{{ t('clip.manager.empty') }}</span>
                <span v-else>{{ t('clip.manager.no_match', { search: effectiveQuery }) }}</span>
              </p>
              <p
                v-if="entries.length === 0"
                class="mt-1 max-w-64 text-xs leading-relaxed text-dimmed"
              >
                {{ t('assetBrowser.clipEmptyHint') }}
              </p>
            </div>

            <div v-else class="space-y-1">
              <template v-for="group in groupedItems" :key="group.category">
                <div class="px-1 pb-1.5 pt-3 text-xs font-medium text-muted">
                  {{ group.category }}
                </div>
                <div
                  role="listbox"
                  :aria-label="group.category"
                  aria-multiselectable="true"
                  data-asset-list
                  class="clip-grid grid gap-3"
                  :class="viewMode === 'list' ? 'clip-grid--list' : ''"
                >
                  <UContextMenu
                    v-for="item in group.items"
                    :key="item.id"
                    :items="ctxMenuItems(item)"
                  >
                    <div
                      draggable="true"
                      role="option"
                      data-asset-option
                      :data-asset-id="item.id"
                      :tabindex="isTabStop(item.id) ? 0 : -1"
                      :aria-selected="isSelected(item.id)"
                      :aria-label="
                        t('editor.dock.select_asset', {
                          name: item.label || t('clip.manager.untitled'),
                        })
                      "
                      class="group cursor-grab overflow-hidden rounded-lg border bg-default active:cursor-grabbing focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                      :class="
                        isSelected(item.id)
                          ? 'border-primary bg-primary/5'
                          : 'border-default hover:border-accented hover:bg-elevated/25'
                      "
                      @click="onRowClick(item.id, $event)"
                      @dblclick="onPick(item.id)"
                      @focus="setActive(item.id)"
                      @keydown="onRowKeydown(item.id, $event)"
                      @contextmenu="selClick(item.id)"
                      @dragstart="(e) => startEditorDrag({ type: 'clip', id: item.id }, e)"
                    >
                      <div class="clip-preview aspect-[15/7] border-b border-default">
                        <ClipTimelinePreview
                          :duration-us="item.durationUs"
                          :event-count="item.eventCount"
                          :mouse-mode="item.meta?.mouseMode"
                          :base-resolution="item.meta?.baseResolution"
                        />
                      </div>
                      <div class="min-w-0 p-3">
                        <div class="flex items-start gap-2">
                          <div class="min-w-0 flex-1">
                            <div class="flex items-center gap-2">
                              <span class="truncate text-[13px] font-semibold text-highlighted">{{
                                item.label || t('clip.manager.untitled')
                              }}</span>
                              <UIcon
                                v-if="isSelected(item.id)"
                                name="i-tabler-circle-check-filled"
                                class="size-4 shrink-0 text-primary"
                              />
                            </div>
                            <p
                              v-if="item.description"
                              class="mt-1 line-clamp-2 text-xs leading-relaxed text-dimmed"
                            >
                              {{ item.description }}
                            </p>
                          </div>
                          <UButton
                            size="xs"
                            variant="ghost"
                            color="neutral"
                            icon="i-tabler-dots"
                            class="size-7 shrink-0 p-0 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100"
                            :aria-label="t('editor.dock.detail')"
                            @click.stop="openDetail(item.id)"
                            @dblclick.stop
                          />
                        </div>
                        <div
                          class="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-dimmed"
                        >
                          <span>{{ formatDuration(item.durationUs) }}</span>
                          <span>{{ t('assetBrowser.inputEvents', { n: item.eventCount }) }}</span>
                          <span v-if="item.tags?.length" class="truncate">{{
                            item.tags.slice(0, 2).join(' · ')
                          }}</span>
                        </div>
                      </div>
                    </div>
                  </UContextMenu>
                </div>
              </template>
            </div>
          </div>

          <AssetPager
            v-model:page="page"
            :total="pageResult.total"
            :total-pages="pageResult.totalPages"
            :items-per-page="pageSize"
          />
        </div>

        <AssetWorkspaceInspector
          v-if="workspace"
          :open="drillIn && !!detailId"
          :title="
            (detailId && (store.clips.find((item) => item.id === detailId)?.label || detailId)) ||
            t('assetBrowser.actionClips')
          "
          @close="drillIn = false"
        >
          <ClipDetailPanel :clip-id="detailId" @insert="onDetailInsert" />
        </AssetWorkspaceInspector>
      </div>
    </template>
  </div>

  <!-- 批量加标签 -->
  <BaseModal
    v-model:open="batchTagsOpen"
    :title="t('library.batch.add_tags_title')"
    icon="i-tabler-tags"
    size="md"
  >
    <UInputMenu
      v-model="batchTags"
      multiple
      :create-item="'always'"
      :items="allTags"
      size="sm"
      :placeholder="t('library.batch.add_tags_placeholder')"
      @create="(v: string) => (batchTags = [...batchTags, v])"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchTagsOpen = false">{{
        t('common.cancel')
      }}</UButton>
      <UButton color="primary" :disabled="batchTags.length === 0" @click="onBatchAddTags">{{
        t('library.batch.add_tags_apply')
      }}</UButton>
    </template>
  </BaseModal>

  <!-- 批量改分类 -->
  <BaseModal
    v-model:open="batchCategoryOpen"
    :title="t('library.batch.change_category_title')"
    icon="i-tabler-category"
    size="md"
  >
    <UInputMenu
      v-model="batchCategory"
      :create-item="'always'"
      :items="allCategories"
      size="sm"
      :placeholder="t('library.batch.change_category_placeholder')"
      @create="(v: string) => (batchCategory = v)"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchCategoryOpen = false">{{
        t('common.cancel')
      }}</UButton>
      <UButton color="primary" @click="onBatchChangeCategory">{{
        t('library.batch.change_category_apply')
      }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, nextTick, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConfirm } from '@/composables/useConfirm'
import { useLocalStorage } from '@vueuse/core'
import { backend } from '@/lib/backend'
import { useClipsStore, type ClipSummary } from '@/stores/clips'
import { useListSelection } from '@/composables/editor/useListSelection'
import { filterSubgraphs, groupByCategory, paginate } from '@/lib/libraryFilter'
import { startEditorDrag } from '@/composables/editor/useEditorDragDrop'
import BaseModal from '@/components/common/BaseModal.vue'
import AssetSelectionBar from './AssetSelectionBar.vue'
import AssetBrowserToolbar from './AssetBrowserToolbar.vue'
import AssetPager from './AssetPager.vue'
import AssetCategoryRail from './AssetCategoryRail.vue'
import AssetWorkspaceInspector from './AssetWorkspaceInspector.vue'
import ClipTimelinePreview from './ClipTimelinePreview.vue'
import ClipDetailPanel from '@/components/containers/ClipDetailPanel.vue'
import { useRovingAssetList } from '@/composables/editor/useRovingAssetList'
import { useAssetBrowserPreferences } from '@/composables/editor/useAssetBrowserPreferences'

const { t } = useI18n()
const { workspace = false, workspaceQuery = '' } = defineProps<{
  workspace?: boolean
  workspaceQuery?: string
}>()
const emit = defineEmits<{
  'pick-clip': [clipID: string]
}>()

// 选中 clip → 插裸 PlayClip 引用节点.
function onPick(clipID: string) {
  emit('pick-clip', clipID)
}

const store = useClipsStore()
const { confirm } = useConfirm()

const { query, categoryFilter, tagFilter, sortKey, sortDesc, viewMode } =
  useAssetBrowserPreferences<'label' | 'createdAt' | 'duration'>('clips', 'label')
const effectiveQuery = computed(() => workspaceQuery.trim() || query.value)
const toolbarRef = useTemplateRef<{ focusSearch: () => Promise<void> }>('toolbarRef')

const sortItems = computed(() => [
  { label: t('clip.manager.view_by_name'), value: 'label' },
  { label: t('clip.manager.view_by_created'), value: 'createdAt' },
  { label: t('clip.manager.view_by_duration'), value: 'duration' },
])

const entries = computed<ClipSummary[]>(() => store.clips)

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const c of entries.value) if (c.category) set.add(c.category)
  return [...set].sort()
})
const allTags = computed(() => {
  const set = new Set<string>()
  for (const c of entries.value) for (const tg of c.tags ?? []) set.add(tg)
  return [...set].sort()
})

const categoryFilterItems = computed(() => [
  { label: t('library.explorer.filter_category_all'), id: 'all' },
  ...allCategories.value.map((c) => ({ label: c, id: `c:${c}` })),
  { label: t('library.explorer.uncategorized'), id: 'none' },
])

function formatDuration(us: number): string {
  const ms = us / 1000
  if (ms < 1000) return `${Math.round(ms)} ms`
  return `${(ms / 1000).toFixed(1)} s`
}

const filteredItems = computed<ClipSummary[]>(() => {
  const arr = filterSubgraphs(entries.value, {
    query: effectiveQuery.value,
    category:
      categoryFilter.value === 'all'
        ? null
        : categoryFilter.value === 'none'
          ? ''
          : categoryFilter.value.slice(2),
    tags: tagFilter.value,
  })
  const sorted = [...arr]
  sorted.sort((a, b) => {
    let cmp = 0
    switch (sortKey.value) {
      case 'label':
        cmp = (a.label ?? '').localeCompare(b.label ?? '')
        break
      case 'createdAt':
        cmp = (a.createdAt ?? '').localeCompare(b.createdAt ?? '')
        break
      case 'duration':
        cmp = a.durationUs - b.durationUs
        break
    }
    return sortDesc.value ? -cmp : cmp
  })
  return sorted
})

const page = ref(1)
const pageSize = useLocalStorage('clip.pageSize', 50)
const pageResult = computed(() => paginate(filteredItems.value, page.value, pageSize.value))
const groupedItems = computed(() =>
  groupByCategory(pageResult.value.pageItems, t('library.explorer.uncategorized')),
)

watch([effectiveQuery, categoryFilter, tagFilter, pageSize, sortKey, sortDesc], () => {
  page.value = 1
})
watch(
  () => pageResult.value.totalPages,
  (tp) => {
    if (page.value > tp) page.value = tp
  },
)

const visibleIds = computed(() => groupedItems.value.flatMap((g) => g.items.map((i) => i.id)))
const { selected, click: selClick, clear: selClear, isSelected } = useListSelection(visibleIds)
const { isTabStop, setActive, move } = useRovingAssetList(visibleIds)

// 数据由 AssetDockPanel 统一预载；浏览偏好保留到下次打开。
onMounted(async () => {
  await nextTick()
  await toolbarRef.value?.focusSearch()
})

function onRowClick(id: string, e: MouseEvent) {
  detailId.value = id
  if (workspace && !e.ctrlKey && !e.metaKey && !e.shiftKey) drillIn.value = true
  selClick(id, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })
}

function onRowKeydown(id: string, e: KeyboardEvent) {
  if (e.target !== e.currentTarget) return
  if (move(id, e)) return
  if (e.key === 'Enter') {
    e.preventDefault()
    onPick(id)
    return
  }
  if (e.key !== ' ') return
  e.preventDefault()
  selClick(id)
}

function byId(id: string): ClipSummary | undefined {
  return store.clips.find((c) => c.id === id)
}

// ── 详情 (按需弹出) ──
const detailId = ref<string | null>(null)
const drillIn = ref(false)
function openDetail(id: string) {
  detailId.value = id
  drillIn.value = true
}
function onDetailInsert() {
  if (detailId.value) onPick(detailId.value)
  if (!workspace) drillIn.value = false
}

function ctxMenuItems(item: ClipSummary) {
  return [
    [
      {
        label: t('library.explorer.insert'),
        icon: 'i-tabler-package-import',
        onSelect: () => onPick(item.id),
      },
      {
        label: t('editor.dock.detail'),
        icon: 'i-tabler-info-circle',
        onSelect: () => openDetail(item.id),
      },
    ],
    [
      {
        label: t('library.card.delete'),
        icon: 'i-tabler-trash',
        color: 'error' as const,
        onSelect: () => onDelete(item),
      },
    ],
  ]
}

// 单项删除: 先扫引用, 被引用则警告, 确认后删.
async function onDelete(item: ClipSummary) {
  const refs = await backend.assets.referrers(item.id)
  const name = item.label || t('clip.manager.untitled')
  const desc =
    (refs?.length ?? 0) > 0
      ? t('clip.manager.batch_delete_confirm_referenced', { n: 1, refs: refs?.length ?? 0 })
      : t('clip.manager.batch_delete_confirm', { n: 1 })
  const yes = await confirm({
    title: t('clip.manager.batch_delete_title'),
    description: `${name} — ${desc}`,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  await store.remove(item.id)
  await store.refresh()
}

// ── 批量 ──
const batchMenuItems = computed(() => [
  [
    {
      label: t('library.batch.add_tags'),
      icon: 'i-tabler-tags',
      onSelect: () => {
        batchTagsOpen.value = true
      },
    },
    {
      label: t('library.batch.change_category'),
      icon: 'i-tabler-category',
      onSelect: () => {
        batchCategoryOpen.value = true
      },
    },
  ],
  [
    {
      label: t('library.batch.delete'),
      icon: 'i-tabler-trash',
      color: 'error' as const,
      onSelect: () => {
        void onBatchDelete()
      },
    },
  ],
])

const batchTagsOpen = ref(false)
const batchTags = ref<string[]>([])
async function onBatchAddTags() {
  const add = batchTags.value.map((s) => s.trim()).filter(Boolean)
  if (add.length === 0) {
    batchTagsOpen.value = false
    return
  }
  for (const id of selected.value) {
    const c = byId(id)
    if (!c) continue
    const tags = [...new Set([...(c.tags ?? []), ...add])]
    await backend.clipsContainer.update(id, c.label, c.description ?? '', c.category ?? '', tags)
  }
  await store.refresh()
  batchTagsOpen.value = false
  batchTags.value = []
}

const batchCategoryOpen = ref(false)
const batchCategory = ref('')
async function onBatchChangeCategory() {
  const target = batchCategory.value.trim()
  for (const id of selected.value) {
    const c = byId(id)
    if (!c) continue
    await backend.clipsContainer.update(id, c.label, c.description ?? '', target, c.tags ?? [])
  }
  await store.refresh()
  batchCategoryOpen.value = false
  batchCategory.value = ''
}

async function onBatchDelete() {
  const ids = [...selected.value]
  const referenced: string[] = []
  for (const id of ids) {
    const refs = await backend.assets.referrers(id)
    if ((refs?.length ?? 0) > 0) referenced.push(byId(id)?.label || id)
  }
  const desc =
    referenced.length > 0
      ? t('clip.manager.batch_delete_confirm_referenced', {
          n: ids.length,
          refs: referenced.length,
        })
      : t('clip.manager.batch_delete_confirm', { n: ids.length })
  const yes = await confirm({
    title: t('clip.manager.batch_delete_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  for (const id of ids) await store.remove(id)
  await store.refresh()
  selClear()
}
</script>

<style scoped>
.asset-panel {
  container-type: inline-size;
}

.clip-grid {
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
}

.clip-grid--list {
  grid-template-columns: 1fr;
}

.clip-grid--list [data-asset-option] {
  display: grid;
  grid-template-columns: 170px minmax(0, 1fr);
}

.clip-grid--list .clip-preview {
  border-bottom: 0;
  border-right: 1px solid var(--ui-border);
}

@container (width < 760px) {
  [data-workspace='true'] .asset-category-rail {
    display: none;
  }
}

@container (width < 520px) {
  .clip-grid {
    grid-template-columns: 1fr;
  }
}
</style>
