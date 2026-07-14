<!-- 视觉模板资产面板：共享筛选/分页/键盘模型，dock 原位钻取详情，workspace 常驻检查器。
     pick 模式保留点选即回写；管理模式单击选中，双击或详情按钮进入编辑。 -->
<template>
  <div class="asset-panel relative flex h-full min-h-0 flex-col" :data-workspace="workspace">
    <div v-if="drillIn && detailId && !workspace && !pickMode" class="flex h-full min-h-0 flex-col">
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
          tplStore.map[detailId]?.name || detailId
        }}</span>
      </div>
      <TemplateDetailPanel
        :guid="detailId"
        :container-id="containerId"
        :pick-mode="false"
        :assigned="false"
      />
    </div>

    <template v-else>
      <div
        class="flex shrink-0 items-center justify-between gap-3 border-b border-default px-3 py-2.5"
      >
        <div class="min-w-0">
          <p class="text-sm font-semibold text-highlighted">
            {{ t('assetBrowser.visualTemplates') }}
          </p>
          <p class="text-xs text-dimmed">{{ t('assetBrowser.templateSubtitle') }}</p>
        </div>
        <UButton
          color="primary"
          icon="i-tabler-camera"
          size="xs"
          class="shrink-0"
          @click="onNewTemplate"
        >
          {{ t('template.capture.title') }}
        </UButton>
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
            :search-placeholder="t('template.manager.search')"
            :category-items="categoryFilterItems"
            :tag-items="allTags"
            :sort-items="sortItems"
            allow-view-switch
            :show-search="!workspace"
            :show-category-scopes="!workspace"
          />

          <AssetSelectionBar
            v-if="!pickMode"
            :count="selected.size"
            :batch-items="batchMenuItems"
            @clear="selClear()"
          />

          <div data-asset-browser-list class="min-h-0 flex-1 overflow-y-auto pr-1 select-none">
            <div
              v-if="filteredItems.length === 0"
              class="flex min-h-56 flex-col items-center justify-center text-center"
            >
              <div
                class="mb-3 flex size-11 items-center justify-center rounded-lg bg-elevated/60 text-muted"
              >
                <UIcon
                  :name="entries.length === 0 ? 'i-tabler-camera-plus' : 'i-tabler-filter-off'"
                  class="size-5"
                />
              </div>
              <p class="text-sm font-medium text-toned">
                <span v-if="entries.length === 0">{{ t('template.manager.empty') }}</span>
                <span v-else>{{ t('template.manager.no_match', { search: effectiveQuery }) }}</span>
              </p>
              <p
                v-if="entries.length === 0"
                class="mt-1 max-w-64 text-xs leading-relaxed text-dimmed"
              >
                {{ t('template.manager.empty_hint') }}
              </p>
            </div>

            <template v-else>
              <div v-for="group in groupedItems" :key="group.category">
                <div class="px-1 pb-1.5 pt-3 text-xs font-medium text-muted">
                  {{ group.category }}
                </div>
                <div
                  role="listbox"
                  :aria-label="group.category"
                  aria-multiselectable="true"
                  data-asset-list
                  class="template-asset-grid grid gap-3"
                  :class="viewMode === 'list' ? 'template-asset-grid--list' : ''"
                >
                  <div
                    v-for="item in group.items"
                    :key="item.guid"
                    role="option"
                    data-asset-option
                    :data-asset-id="item.guid"
                    :tabindex="isTabStop(item.guid) ? 0 : -1"
                    :aria-selected="cellActive(item.guid)"
                    :aria-label="t('editor.dock.select_asset', { name: item.name || item.guid })"
                    class="group relative cursor-pointer overflow-hidden rounded-lg border bg-default transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                    :class="
                      cellActive(item.guid)
                        ? 'border-primary bg-primary/5 ring-1 ring-inset ring-primary/50'
                        : 'border-default hover:border-accented hover:bg-elevated/25'
                    "
                    @click="onCellClick(item.guid, $event)"
                    @dblclick="!pickMode && openDetail(item.guid)"
                    @focus="setActive(item.guid)"
                    @keydown="onCellKeydown(item.guid, $event)"
                  >
                    <div
                      class="template-card-preview flex aspect-[16/10] items-center justify-center bg-sunken p-3"
                    >
                      <TemplateThumb
                        :sha="item.firstBlobSha"
                        :alt="item.name || item.guid"
                        :max-upscale="1"
                      />
                    </div>
                    <div class="min-w-0 px-2.5 py-2">
                      <div class="flex items-center gap-2">
                        <span
                          class="min-w-0 flex-1 truncate text-[13px] font-medium text-highlighted"
                          >{{ item.name || item.guid }}</span
                        >
                        <UIcon
                          v-if="cellActive(item.guid)"
                          name="i-tabler-circle-check-filled"
                          class="size-4 shrink-0 text-primary"
                        />
                      </div>
                      <div class="mt-1 flex items-center gap-1.5 text-xs text-dimmed">
                        <span>{{
                          t('assetBrowser.variantCount', { n: item.variantCount ?? 0 })
                        }}</span>
                        <span v-if="item.tags?.length" class="truncate"
                          >· {{ item.tags.slice(0, 2).join(' · ') }}</span
                        >
                      </div>
                    </div>

                    <!-- pick: 已指派 ✓ 角标 -->
                    <div
                      v-if="pickMode && isAssigned(item.guid)"
                      class="absolute top-1 right-1 size-5 rounded-full bg-primary text-inverted flex items-center justify-center shadow"
                    >
                      <UIcon name="i-tabler-check" class="size-3.5" />
                    </div>

                    <!-- 管理: ⋯ 详情 (hover); 深色半透明, 缩略图上够对比 (不用 solid 白) -->
                    <UButton
                      v-if="!pickMode"
                      size="xs"
                      variant="soft"
                      color="neutral"
                      icon="i-tabler-dots"
                      class="absolute right-2 top-2 size-7 p-0 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100"
                      :aria-label="t('editor.dock.detail')"
                      @click.stop="openDetail(item.guid)"
                      @dblclick.stop
                    />
                  </div>
                </div>
              </div>
            </template>
          </div>

          <div
            v-if="pickMode"
            class="flex shrink-0 items-center justify-between border-t border-default pt-2 text-xs text-toned"
          >
            <span>{{ t('template.picker.selected_count', { n: assigned.length }) }}</span>
            <span>{{ t('library.toolbar.total', { n: pageResult.total }) }}</span>
          </div>
          <AssetPager
            v-else
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
            (detailId && (tplStore.map[detailId]?.name || detailId)) ||
            t('assetBrowser.visualTemplates')
          "
          @close="drillIn = false"
        >
          <TemplateDetailPanel
            :guid="detailId"
            :container-id="containerId"
            :pick-mode="false"
            :assigned="false"
          />
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
import { useLocalStorage } from '@vueuse/core'
import { backend, type AssetSummary } from '@/lib/backend'
import { useTemplatesStore } from '@/stores/templates'
import { useConfirm } from '@/composables/useConfirm'
import { useListSelection } from '@/composables/editor/useListSelection'
import { filterSubgraphs, groupByCategory, paginate } from '@/lib/libraryFilter'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import BaseModal from '@/components/common/BaseModal.vue'
import AssetSelectionBar from './AssetSelectionBar.vue'
import AssetBrowserToolbar from './AssetBrowserToolbar.vue'
import AssetPager from './AssetPager.vue'
import AssetCategoryRail from './AssetCategoryRail.vue'
import AssetWorkspaceInspector from './AssetWorkspaceInspector.vue'
import TemplateDetailPanel from '@/components/containers/TemplateDetailPanel.vue'
import TemplateThumb from './TemplateThumb.vue'
import { useRovingAssetList } from '@/composables/editor/useRovingAssetList'
import { useAssetBrowserPreferences } from '@/composables/editor/useAssetBrowserPreferences'

const { t } = useI18n()
const props = defineProps<{
  containerId: string
  pickMode?: boolean
  modelValue?: string[]
  workspace?: boolean
  workspaceQuery?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [v: string[]] }>()

// pick 模式: 缩略图勾选=指派给节点 (按 modelValue 回显); 管理模式: 选中=批量.
const assigned = computed<string[]>(() => props.modelValue ?? [])
function isAssigned(guid: string) {
  return assigned.value.includes(guid)
}
function toggleAssign(guid: string) {
  emit(
    'update:modelValue',
    isAssigned(guid) ? assigned.value.filter((g) => g !== guid) : [...assigned.value, guid],
  )
}
function cellActive(guid: string) {
  return props.pickMode ? isAssigned(guid) : isSelected(guid)
}
function onCellClick(guid: string, e: MouseEvent) {
  if (props.pickMode) {
    toggleAssign(guid)
    return
  }
  detailId.value = guid
  if (props.workspace && !e.ctrlKey && !e.metaKey && !e.shiftKey) drillIn.value = true
  selClick(guid, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })
}

function onCellKeydown(guid: string, e: KeyboardEvent) {
  if (e.target !== e.currentTarget) return
  if (move(guid, e)) return
  if (e.key === 'Enter' && !props.pickMode) {
    e.preventDefault()
    openDetail(guid)
    return
  }
  if (e.key !== ' ' && e.key !== 'Enter') return
  e.preventDefault()
  if (props.pickMode) toggleAssign(guid)
  else selClick(guid)
}

const tplStore = useTemplatesStore()
const { confirm } = useConfirm()

const { query, categoryFilter, tagFilter, sortKey, sortDesc, viewMode } =
  useAssetBrowserPreferences<'name' | 'createdAt' | 'variantCount'>('templates', 'name')
const effectiveQuery = computed(() =>
  props.workspace ? (props.workspaceQuery ?? '').trim() : query.value,
)
const toolbarRef = useTemplateRef<{ focusSearch: () => Promise<void> }>('toolbarRef')

// 排序
const sortItems = computed(() => [
  { label: t('template.manager.view_by_name'), value: 'name' },
  { label: t('template.manager.view_by_created'), value: 'createdAt' },
  { label: t('template.manager.view_by_res'), value: 'variantCount' },
])

// entries: AssetSummary[] (template kind); 加 label 别名供 libraryFilter (它认 label).
type TplItem = AssetSummary & { label: string }
const entries = computed<TplItem[]>(() =>
  Object.values(tplStore.map).map((s) => ({ ...s, label: s.name })),
)

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const s of entries.value) if (s.category) set.add(s.category)
  return [...set].sort()
})
const allTags = computed(() => {
  const set = new Set<string>()
  for (const s of entries.value) for (const tg of s.tags ?? []) set.add(tg)
  return [...set].sort()
})

// 过滤
const categoryFilterItems = computed(() => [
  { label: t('library.explorer.filter_category_all'), id: 'all' },
  ...allCategories.value.map((c) => ({ label: c, id: `c:${c}` })),
  { label: t('library.explorer.uncategorized'), id: 'none' },
])

const filteredItems = computed<TplItem[]>(() => {
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
      case 'name':
        cmp = (a.name ?? '').localeCompare(b.name ?? '')
        break
      case 'createdAt':
        cmp = (a.createdAt ?? '').localeCompare(b.createdAt ?? '')
        break
      case 'variantCount':
        cmp = (a.variantCount ?? 0) - (b.variantCount ?? 0)
        break
    }
    return sortDesc.value ? -cmp : cmp
  })
  return sorted
})

// 分页 + 分组
const page = ref(1)
const pageSize = useLocalStorage('template.pageSize', 50)
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

// 多选 (批量用)
const visibleIds = computed(() => groupedItems.value.flatMap((g) => g.items.map((i) => i.guid)))
const { selected, click: selClick, clear: selClear, isSelected } = useListSelection(visibleIds)
const { isTabStop, setActive, move } = useRovingAssetList(visibleIds)

// 数据由 AssetDockPanel 统一预载；浏览偏好保留到下次打开。
onMounted(async () => {
  await nextTick()
  await toolbarRef.value?.focusSearch()
})

const detailId = ref<string | null>(null)
const drillIn = ref(false)
function openDetail(guid: string) {
  detailId.value = guid
  drillIn.value = true
}

// 新建截图
async function onNewTemplate() {
  const id = 'tpl-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>(
    'tools:picker-result',
    (p) => p?.id === id,
  )
  await backend.tools.openScreenPicker('template_save', id, props.containerId)
  const result = await waiter
  if (!result.payload?.cancelled) {
    await tplStore.reload()
    if (props.pickMode && result.payload?.guid && !isAssigned(result.payload.guid)) {
      emit('update:modelValue', [...assigned.value, result.payload.guid])
    }
  }
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
  for (const guid of selected.value) {
    const s = tplStore.map[guid]
    if (!s) continue
    const tags = [...new Set([...(s.tags ?? []), ...add])]
    await backend.assets.updateMeta(guid, s.name, s.description ?? '', s.category ?? '', tags)
  }
  await tplStore.reload()
  batchTagsOpen.value = false
  batchTags.value = []
}

const batchCategoryOpen = ref(false)
const batchCategory = ref('')
async function onBatchChangeCategory() {
  const target = batchCategory.value.trim()
  for (const guid of selected.value) {
    const s = tplStore.map[guid]
    if (!s) continue
    await backend.assets.updateMeta(guid, s.name, s.description ?? '', target, s.tags ?? [])
  }
  await tplStore.reload()
  batchCategoryOpen.value = false
  batchCategory.value = ''
}

async function onBatchDelete() {
  const ids = [...selected.value]
  const referenced: string[] = []
  for (const guid of ids) {
    const refs = await backend.assets.referrers(guid)
    if ((refs?.length ?? 0) > 0) referenced.push(tplStore.map[guid]?.name || guid)
  }
  const desc =
    referenced.length > 0
      ? t('template.manager.batch_delete_confirm_referenced', {
          n: ids.length,
          refs: referenced.length,
        })
      : t('template.manager.batch_delete_confirm', { n: ids.length })
  const yes = await confirm({
    title: t('template.manager.batch_delete_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  for (const guid of ids) await backend.assets.delete_(guid)
  await tplStore.reload()
  selClear()
}
</script>

<style scoped>
.asset-panel {
  container-type: inline-size;
}

.template-asset-grid {
  grid-template-columns: repeat(auto-fill, minmax(156px, 1fr));
}

.template-asset-grid--list {
  grid-template-columns: 1fr;
}

.template-asset-grid--list > [data-asset-option] {
  display: grid;
  grid-template-columns: 132px minmax(0, 1fr);
}

@container (width < 760px) {
  [data-workspace='true'] .asset-category-rail {
    display: none;
  }
}

@container (width < 520px) {
  .template-asset-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .template-asset-grid--list {
    grid-template-columns: 1fr;
  }
}
</style>
