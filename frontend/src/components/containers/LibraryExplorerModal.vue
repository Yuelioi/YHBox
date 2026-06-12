<!-- 子图库 modal (编辑器内唯一库管理入口). 入口: toolbar 📚 / 左 rail.
     本地 tab: 单击选中出详情; 双击/详情按钮插引用; Ctrl/Shift 多选 + 批量删/加标签;
     右键单项菜单。在线 tab: 占位 (跨机分享留口)。 -->
<template>
  <BaseModal
    v-model:open="modelOpen"
    :title="t('library.explorer.title')"
    icon="i-tabler-books"
    size="5xl"
  >
    <template #header-extra>
      <UTabs
        v-model="activeTab"
        :items="tabItems"
        size="xs"
        :content="false"
        class="mr-2"
      />
    </template>

    <!-- 在线: 占位 -->
    <div v-if="activeTab === 'online'" class="flex flex-col items-center justify-center text-center py-16">
      <UIcon name="i-tabler-cloud" class="size-12 text-dimmed mb-3" />
      <h3 class="text-sm text-toned font-medium">{{ t('library.online.title') }}</h3>
      <p class="text-xs text-dimmed mt-2 max-w-xs">{{ t('library.online.desc') }}</p>
    </div>

    <!-- 本地: 列表 + 右栏 (详情/批量) 双栏 -->
    <div v-else class="flex gap-4">
      <div class="flex-1 min-w-0 space-y-3">
        <div class="flex items-center gap-3">
          <UInput
            ref="searchInputRef"
            v-model="query"
            :placeholder="t('library.explorer.search')"
            icon="i-tabler-search"
            size="sm"
            class="flex-1"
            @keydown.escape="modelOpen = false"
          />
          <span class="text-[10px] text-dimmed">{{ t('library.explorer.esc_close') }}</span>
        </div>

        <div class="flex items-center gap-2">
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

        <div class="max-h-[56vh] overflow-y-auto select-none">
          <div
            v-if="filteredItems.length === 0"
            class="text-center text-xs text-dimmed py-8 italic"
          >
            <span v-if="lib.loading">{{ t('library.loading') }}</span>
            <span v-else-if="lib.subgraphs.length === 0"
              >{{ t('library.explorer.empty') }}</span
            >
            <span v-else>{{ t('library.explorer.no_match') }}</span>
          </div>

          <div v-else class="space-y-2">
            <template v-for="group in groupedItems" :key="group.category">
              <div class="text-[10px] font-semibold text-dimmed uppercase tracking-wider px-1 pt-2 pb-0.5">
                {{ group.category }}
              </div>
              <UContextMenu v-for="item in group.items" :key="item.id" :items="ctxMenuItems(item)">
                <div
                  class="group rounded p-3 cursor-pointer"
                  :class="isSelected(item.id) ? 'bg-primary/15 ring-1 ring-inset ring-primary/50' : 'bg-elevated/30 hover:bg-elevated/60'"
                  @click="onRowClick(item.id, $event)"
                  @dblclick="onPick(item.id)"
                  @contextmenu="selClick(item.id)"
                >
                  <div class="flex items-start gap-2">
                    <span
                      class="mt-0.5 shrink-0 transition-opacity"
                      :class="isSelected(item.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'"
                      @click.stop
                      @dblclick.stop
                    >
                      <UCheckbox
                        :model-value="isSelected(item.id)"
                        size="sm"
                        @update:model-value="selClick(item.id, { ctrl: true })"
                      />
                    </span>
                    <UIcon name="i-tabler-package" class="size-4 text-primary mt-0.5 shrink-0" />
                    <div class="flex-1 min-w-0">
                      <div class="text-sm font-medium">{{ item.label }}</div>
                      <div
                        v-if="item.description"
                        class="text-[11px] text-dimmed mt-0.5 line-clamp-2"
                      >
                        {{ item.description }}
                      </div>
                      <div
                        v-if="item.tags && item.tags.length > 0"
                        class="flex flex-wrap gap-1 mt-1"
                      >
                        <span
                          v-for="tg in item.tags"
                          :key="tg"
                          class="px-1.5 py-0 bg-elevated/60 text-[9px] rounded text-dimmed"
                        >
                          {{ tg }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </UContextMenu>
            </template>
          </div>
        </div>

        <!-- 底部工具栏 (双态): 无选中 = 计数 + 分页; 有选中 = 批量操作 + 分页 -->
        <div class="flex items-center justify-between gap-3 pt-2 border-t border-default">
          <div v-if="selected.size === 0" class="text-[11px] text-dimmed">
            {{ t('library.toolbar.total', { n: pageResult.total }) }}
          </div>
          <div v-else class="flex items-center gap-1.5 flex-wrap">
            <span class="text-[11px] text-toned">{{ t('library.batch.selected_n', { n: selected.size }) }}</span>
            <UButton size="xs" variant="soft" color="error" icon="i-tabler-trash" @click="onBatchDelete">{{ t('library.batch.delete') }}</UButton>
            <UButton size="xs" variant="soft" color="primary" icon="i-tabler-tags" @click="batchTagsOpen = true">{{ t('library.batch.add_tags') }}</UButton>
            <UButton size="xs" variant="soft" color="primary" icon="i-tabler-category" @click="batchCategoryOpen = true">{{ t('library.batch.change_category') }}</UButton>
            <UButton size="xs" variant="ghost" color="neutral" @click="selClear()">{{ t('library.batch.clear') }}</UButton>
          </div>
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

      <LibraryDetailPanel
        class="max-h-[65vh]"
        :sgID="anchor"
        @insert="anchor && onPick(anchor)"
      />
    </div>
  </BaseModal>

  <!-- 批量加标签 -->
  <BaseModal v-model:open="batchTagsOpen" :title="t('library.batch.add_tags_title')" icon="i-tabler-tags" size="md">
    <UInputMenu
      v-model="batchTags"
      multiple
      creatable
      :items="allTags"
      size="sm"
      :placeholder="t('library.batch.add_tags_placeholder')"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchTagsOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" :disabled="batchTags.length === 0" @click="onBatchAddTags">{{ t('library.batch.add_tags_apply') }}</UButton>
    </template>
  </BaseModal>

  <!-- 批量改分类 -->
  <BaseModal v-model:open="batchCategoryOpen" :title="t('library.batch.change_category_title')" icon="i-tabler-category" size="md">
    <UInputMenu
      v-model="batchCategory"
      creatable
      :items="allCategories"
      size="sm"
      :placeholder="t('library.batch.change_category_placeholder')"
    />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchCategoryOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" @click="onBatchChangeCategory">{{ t('library.batch.change_category_apply') }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useLocalStorage } from '@vueuse/core'
import { useLibraryStore } from '@/stores/library'
import { useConfirm } from '@/composables/useConfirm'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { useAutoFocusOnOpen } from '@/composables/editor/useAutoFocusOnOpen'
import { useListSelection } from '@/composables/editor/useListSelection'
import { filterSubgraphs, groupByCategory, paginate } from '@/lib/libraryFilter'
import BaseModal from '@/components/common/BaseModal.vue'
import LibraryDetailPanel from '@/components/containers/LibraryDetailPanel.vue'
import { backend, type Subgraph } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'

const { t } = useI18n()

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  'update:open': [v: boolean]
  'pick-subgraph': [libraryID: string]
}>()

const modelOpen = useDialogOpen(props, emit)

const query = ref('')
const searchInputRef = ref<any>(null)

const lib = useLibraryStore()
const toast = useToast()
const { confirm } = useConfirm()

const activeTab = ref<'local' | 'online'>('local')
const tabItems = computed(() => [
  { label: t('library.explorer.tab_local'), value: 'local' },
  { label: t('library.explorer.tab_online'), value: 'online' },
])

// Hydrate on mount; refresh when modal opens (cheap — backend caches).
async function refreshLibrary() {
  await lib.reload()
}

// ── 过滤 + 分组 ──
// categoryFilter 用 'all' / 'none' / 'c:<名>' 前缀编码, 防用户分类名撞保留值。
const categoryFilter = ref<string>('all')
const tagFilter = ref<string[]>([])

const allCategories = computed(() => {
  const set = new Set<string>()
  for (const sg of lib.subgraphs) if (sg.category) set.add(sg.category)
  return [...set].sort()
})
const allTags = computed(() => {
  const set = new Set<string>()
  for (const sg of lib.subgraphs) for (const tg of sg.tags ?? []) set.add(tg)
  return [...set].sort()
})

const categoryFilterItems = computed(() => [
  { label: t('library.explorer.filter_category_all'), id: 'all' },
  ...allCategories.value.map((c) => ({ label: c, id: `c:${c}` })),
  { label: t('library.explorer.uncategorized'), id: 'none' },
])

const filteredItems = computed<Subgraph[]>(() =>
  filterSubgraphs(lib.subgraphs, {
    query: query.value,
    category:
      categoryFilter.value === 'all' ? null : categoryFilter.value === 'none' ? '' : categoryFilter.value.slice(2),
    tags: tagFilter.value,
  }),
)

// ── 分页 (过滤后扁平切页, 页内再分组; 每页条数本机记忆) ──
const page = ref(1)
const pageSize = useLocalStorage('library.pageSize', 50)
const pageSizeItems = computed(() => [20, 50, 100].map((n) => ({ label: t('library.toolbar.per_page', { n }), value: n })))

const pageResult = computed(() => paginate(filteredItems.value, page.value, pageSize.value))
const groupedItems = computed(() => groupByCategory(pageResult.value.pageItems, t('library.explorer.uncategorized')))

watch([query, categoryFilter, tagFilter, pageSize], () => { page.value = 1 })
watch(() => pageResult.value.totalPages, (tp) => { if (page.value > tp) page.value = tp })

// ── 选中 (单击/Ctrl/Shift/勾选框; 右键收敛单选); 详情栏跟锚点 = 最后操作行 ──
const visibleIds = computed(() => groupedItems.value.flatMap((g) => g.items.map((i) => i.id)))
const { selected, anchor, click: selClick, clear: selClear, isSelected } = useListSelection(visibleIds)

onMounted(() => refreshLibrary())
useAutoFocusOnOpen(modelOpen, searchInputRef, {
  onOpen: () => {
    void refreshLibrary()
    query.value = ''
    categoryFilter.value = 'all'
    tagFilter.value = []
    page.value = 1
    selClear()
  },
})

function onRowClick(id: string, e: MouseEvent) {
  selClick(id, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })
}

function onPick(libraryID: string) {
  emit('pick-subgraph', libraryID)
  modelOpen.value = false
}

function ctxMenuItems(item: Subgraph) {
  return [
    [
      { label: t('library.explorer.insert'), icon: 'i-tabler-package-import', onSelect: () => onPick(item.id) },
      { label: t('library.card.duplicate'), icon: 'i-tabler-copy-plus', onSelect: () => onDuplicate(item) },
    ],
    [
      { label: t('library.card.copy_id'), icon: 'i-tabler-copy', onSelect: () => onCopyID(item) },
    ],
    [
      { label: t('library.card.delete'), icon: 'i-tabler-trash', color: 'error' as const, onSelect: () => onDelete(item) },
    ],
  ]
}

async function onCopyID(item: Subgraph) {
  try {
    await navigator.clipboard.writeText(item.id)
    toast.add({ title: t('toast.copy_id_success'), color: 'success', icon: 'i-tabler-check', duration: 1500 })
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

// 复制为新子图 (fork, ≈Blender Make Local): 想独立改不影响引用方时用。
async function onDuplicate(item: Subgraph) {
  const dup = await lib.duplicateSubgraph(item.id)
  if (dup) {
    toast.add({ title: t('library.card.duplicated', { name: dup.label }), color: 'success', icon: 'i-tabler-check' })
  }
}

// 删除安全: 先扫引用 — 被容器使用时警告里带"被 N 个容器使用", 确认才真删。
async function onDelete(item: Subgraph) {
  const refs = await lib.referrersOf(item.id)
  const useCount = lib.containerUseCount(refs)
  const name = item.label || item.id
  const desc = useCount > 0
    ? t('library.card.delete_confirm_referenced', { name, n: useCount })
    : t('library.card.delete_confirm_desc', { name })
  const yes = await confirm({
    title: t('library.card.delete_confirm_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  const ok = await lib.deleteSubgraph(item.id)
  if (!ok) {
    toast.add({ title: t('toast.delete_failed'), color: 'error' })
  }
  // 选中项随 visibleIds 收缩自动剔除, 无需手动清。
}

// ── 批量删除: 逐项扫引用汇总警告, 确认后逐项删 (各带 rev), 失败聚合一条 toast ──
async function onBatchDelete() {
  const ids = [...selected.value]
  const referenced: string[] = []
  for (const id of ids) {
    const refs = await lib.referrersOf(id)
    if (lib.containerUseCount(refs) > 0) referenced.push(lib.byId(id)?.label || id)
  }
  const desc = referenced.length > 0
    ? t('library.batch.delete_confirm_referenced', { n: ids.length, m: referenced.length, names: referenced.join('、') })
    : t('library.batch.delete_confirm_desc', { n: ids.length })
  const yes = await confirm({
    title: t('library.batch.delete_confirm_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  let failed = 0
  for (const id of ids) {
    if (!(await lib.deleteSubgraph(id))) failed++
  }
  if (failed > 0) {
    toast.add({ title: t('library.batch.partial_failed', { n: failed }), color: 'error' })
  }
  selClear()
}

// ── 批量加标签: 客户端算并集发全量数组 (RFC7386 数组整体替换) ──
const batchTagsOpen = ref(false)
const batchTags = ref<string[]>([])

async function onBatchAddTags() {
  const add = batchTags.value.map((s) => s.trim()).filter(Boolean)
  if (add.length === 0) {
    batchTagsOpen.value = false
    return
  }
  const ids = [...selected.value]
  let failed = 0
  for (const id of ids) {
    const sg = lib.byId(id)
    if (!sg) {
      failed++
      continue
    }
    const tags = [...new Set([...(sg.tags ?? []), ...add])]
    try {
      await backend.subgraphs.updateSilent(sg.id, JSON.stringify({ tags }), sg.rev)
    } catch {
      failed++
    }
  }
  await lib.reload()
  if (failed > 0) {
    toast.add({ title: t('library.batch.partial_failed', { n: failed }), color: 'error' })
  }
  batchTagsOpen.value = false
  batchTags.value = []
}

// ── 批量改分类: 目标分类整体覆盖各选中项 (空 = 移到未分类) ──
const batchCategoryOpen = ref(false)
const batchCategory = ref('')

async function onBatchChangeCategory() {
  const target = batchCategory.value.trim()
  const ids = [...selected.value]
  let failed = 0
  for (const id of ids) {
    const sg = lib.byId(id)
    if (!sg) {
      failed++
      continue
    }
    try {
      await backend.subgraphs.updateSilent(sg.id, JSON.stringify({ category: target }), sg.rev)
    } catch {
      failed++
    }
  }
  await lib.reload()
  if (failed > 0) {
    toast.add({ title: t('library.batch.partial_failed', { n: failed }), color: 'error' })
  }
  batchCategoryOpen.value = false
  batchCategory.value = ''
}
</script>
