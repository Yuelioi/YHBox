<!-- 模板资产停靠面板 — 缩略图网格 (主操作=看图, 高频零点击).
     pick 模式: 点缩略图=勾选指派给节点 (✓ 角标, 实时回写). 管理模式: 点=选中(批量); ⋯=详情(改名/重拍/变体/删除, 低频按需弹).
     批量改标签/分类/删. 新建截图: tplStore.containerId 定位目标窗口. -->
<template>
  <div class="flex flex-col h-full min-h-0">
    <!-- 顶部: 新建截图 -->
    <div class="flex items-center border-b border-default px-3 py-2 shrink-0">
      <UButton color="primary" icon="i-tabler-camera" size="xs" @click="onNewTemplate">
        {{ t('template.capture.title') }}
      </UButton>
    </div>

    <div class="flex-1 min-h-0 overflow-hidden flex flex-col gap-2 p-3">
      <div class="flex items-center gap-3 shrink-0">
        <UInput
          ref="searchInputRef"
          v-model="query"
          :placeholder="t('template.manager.search')"
          icon="i-tabler-search"
          size="sm"
          class="flex-1"
        />
        <USelect v-model="sortKey" :items="sortItems" size="xs" class="w-32" />
        <UButton
          size="xs"
          variant="ghost"
          :icon="sortDesc ? 'i-tabler-sort-descending' : 'i-tabler-sort-ascending'"
          :title="sortDesc ? t('template.manager.sort_desc') : t('template.manager.sort_asc')"
          @click="sortDesc = !sortDesc"
        />
      </div>

      <div class="flex items-center gap-2 shrink-0">
        <USelectMenu
          v-model="categoryFilter"
          :items="categoryFilterItems"
          value-key="id"
          size="xs"
          class="w-40"
        />
        <UInputMenu
          v-model="tagFilter"
          multiple
          :items="allTags"
          size="xs"
          :placeholder="t('library.explorer.filter_tags')"
          class="flex-1"
        />
      </div>

      <AssetSelectionBar
        v-if="!pickMode"
        :count="selected.size"
        :batch-items="batchMenuItems"
        @clear="selClear()"
      />

      <!-- 缩略图网格 -->
      <div class="flex-1 min-h-0 overflow-y-auto select-none">
        <div v-if="filteredItems.length === 0" class="text-center text-xs text-dimmed py-8 italic">
          <span v-if="entries.length === 0">{{ t('template.manager.empty') }}</span>
          <span v-else>{{ t('template.manager.no_match', { search: query }) }}</span>
        </div>

        <template v-else>
          <div v-for="group in groupedItems" :key="group.category">
            <div
              class="text-[10px] font-semibold text-dimmed uppercase tracking-wider px-1 pt-2 pb-1"
            >
              {{ group.category }}
            </div>
            <div
              class="grid gap-2.5"
              style="grid-template-columns: repeat(auto-fill, minmax(112px, 1fr))"
            >
              <div
                v-for="item in group.items"
                :key="item.guid"
                class="group relative rounded-lg overflow-hidden cursor-pointer border transition-colors"
                :class="
                  cellActive(item.guid)
                    ? 'border-primary ring-1 ring-inset ring-primary/60'
                    : 'border-default hover:border-primary/40'
                "
                @click="onCellClick(item.guid, $event)"
                @dblclick="!pickMode && openDetail(item.guid)"
              >
                <div class="aspect-[4/3] bg-sunken flex items-center justify-center">
                  <TemplateThumb :sha="item.firstBlobSha" />
                </div>
                <div
                  class="px-1.5 py-1 text-[11px] truncate"
                  :class="cellActive(item.guid) ? 'text-highlighted' : 'text-toned'"
                >
                  {{ item.name || item.guid }}
                </div>

                <!-- pick: 已指派 ✓ 角标 -->
                <div
                  v-if="pickMode && isAssigned(item.guid)"
                  class="absolute top-1 right-1 size-5 rounded-full bg-primary text-inverted flex items-center justify-center shadow"
                >
                  <UIcon name="i-tabler-check" class="size-3.5" />
                </div>

                <!-- 管理: ⋯ 详情 (hover); 深色半透明, 缩略图上够对比 (不用 solid 白) -->
                <button
                  v-if="!pickMode"
                  type="button"
                  class="absolute top-1 right-1 size-6 rounded-md bg-black/55 text-white opacity-0 group-hover:opacity-100 hover:bg-black/75 flex items-center justify-center transition-colors"
                  :aria-label="t('editor.dock.detail')"
                  @click.stop="openDetail(item.guid)"
                  @dblclick.stop
                >
                  <UIcon name="i-tabler-dots" class="size-4" />
                </button>
              </div>
            </div>
          </div>
        </template>
      </div>

      <!-- 底部: pick=已指派计数 / 管理=总数 + 分页 (批量操作移到顶部上下文条) -->
      <div class="flex items-center justify-between gap-3 pt-2 border-t border-default shrink-0">
        <span v-if="pickMode" class="text-[11px] text-toned shrink-0">{{
          t('template.picker.selected_count', { n: assigned.length })
        }}</span>
        <span v-else class="text-[11px] text-dimmed shrink-0">{{
          t('library.toolbar.total', { n: pageResult.total })
        }}</span>
        <div class="flex items-center gap-2 shrink-0">
          <UPagination
            v-if="pageResult.totalPages > 1"
            v-model:page="page"
            :total="pageResult.total"
            :items-per-page="pageSize"
            :sibling-count="1"
            size="xs"
          />
          <USelect v-model="pageSize" :items="pageSizeItems" size="xs" class="w-24" />
        </div>
      </div>
    </div>
  </div>

  <!-- 详情 (按需): 改名/描述/分类/标签/多分辨率变体/重拍/删除 -->
  <BaseModal
    v-model:open="detailOpen"
    :title="t('editor.dock.detail')"
    icon="i-tabler-photo"
    size="md"
  >
    <TemplateDetailPanel
      :guid="detailId"
      :pick-mode="false"
      :assigned="false"
      @toggle-assign="() => {}"
    />
  </BaseModal>

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
import { computed, ref, watch, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useLocalStorage } from '@vueuse/core'
import { backend, type AssetSummary } from '@/lib/backend'
import { useTemplatesStore } from '@/stores/templates'
import { useConfirm } from '@/composables/useConfirm'
import { useListSelection } from '@/composables/editor/useListSelection'
import { filterSubgraphs, groupByCategory, paginate } from '@/lib/libraryFilter'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import BaseModal from '@/components/common/BaseModal.vue'
import AssetSelectionBar from './AssetSelectionBar.vue'
import TemplateDetailPanel from '@/components/containers/TemplateDetailPanel.vue'
import TemplateThumb from './TemplateThumb.vue'

const { t } = useI18n()
const props = defineProps<{ pickMode?: boolean; modelValue?: string[] }>()
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
  selClick(guid, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })
}

const tplStore = useTemplatesStore()
const toast = useToast()
const { confirm } = useConfirm()

const query = ref('')
const searchInputRef = ref<any>(null)

// 排序
const sortKey = ref<'name' | 'createdAt' | 'variantCount'>('name')
const sortDesc = ref(false)
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
const categoryFilter = ref<string>('all')
const tagFilter = ref<string[]>([])
const categoryFilterItems = computed(() => [
  { label: t('library.explorer.filter_category_all'), id: 'all' },
  ...allCategories.value.map((c) => ({ label: c, id: `c:${c}` })),
  { label: t('library.explorer.uncategorized'), id: 'none' },
])

const filteredItems = computed<TplItem[]>(() => {
  const arr = filterSubgraphs(entries.value, {
    query: query.value,
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
const pageSizeItems = computed(() =>
  [20, 50, 100].map((n) => ({ label: t('library.toolbar.per_page', { n }), value: n })),
)
const pageResult = computed(() => paginate(filteredItems.value, page.value, pageSize.value))
const groupedItems = computed(() =>
  groupByCategory(pageResult.value.pageItems, t('library.explorer.uncategorized')),
)

watch([query, categoryFilter, tagFilter, pageSize, sortKey, sortDesc], () => {
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

// 面板 mount (切到资产·模板 tab) → reload + 重置过滤 + 聚焦搜索.
onMounted(async () => {
  void tplStore.reload()
  query.value = ''
  categoryFilter.value = 'all'
  tagFilter.value = []
  page.value = 1
  selClear()
  await nextTick()
  const el = searchInputRef.value?.$el as HTMLElement | undefined
  el?.querySelector('input')?.focus()
})

// ── 详情 (按需弹出) ──
const detailOpen = ref(false)
const detailId = ref<string | null>(null)
function openDetail(guid: string) {
  detailId.value = guid
  detailOpen.value = true
}

// 新建截图
async function onNewTemplate() {
  const id = 'tpl-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>(
    'tools:picker-result',
    (p) => p?.id === id,
  )
  await backend.tools.openScreenPicker('template_save', id, tplStore.containerId)
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
