<!-- clip 管理 modal (编辑器内入口: 左 rail 🎬). 两栏: 左列表(搜索/分类·标签过滤/分组/多选/分页) + 右 ClipDetailPanel.
     批量: 选中后下拉改标签/改分类/删. 排序: 名称/创建/时长 × 正逆序. clip 全局资产; 复刻子图库定式, 复用 libraryFilter/useListSelection.
     录制走 toolbar, 此面板只管已录的. -->
<template>
  <BaseModal v-model:open="modelOpen" :title="t('clip.manager.title')" icon="i-tabler-movie" size="5xl">
    <div class="flex gap-4">
      <div class="flex-1 min-w-0 space-y-3">
        <div class="flex items-center gap-3">
          <UInput
            ref="searchInputRef"
            v-model="query"
            :placeholder="t('clip.manager.search')"
            icon="i-tabler-search"
            size="sm"
            class="flex-1"
            @keydown.escape="modelOpen = false"
          />
          <USelect v-model="sortKey" :items="sortItems" size="xs" class="w-32" />
          <UButton
            size="xs"
            variant="ghost"
            :icon="sortDesc ? 'i-tabler-sort-descending' : 'i-tabler-sort-ascending'"
            :title="sortDesc ? t('clip.manager.sort_desc') : t('clip.manager.sort_asc')"
            @click="sortDesc = !sortDesc"
          />
        </div>

        <div class="flex items-center gap-2">
          <USelectMenu v-model="categoryFilter" :items="categoryFilterItems" value-key="id" size="xs" class="w-40" />
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
          <div v-if="filteredItems.length === 0" class="text-center text-xs text-dimmed py-8 italic">
            <span v-if="entries.length === 0">{{ t('clip.manager.empty') }}</span>
            <span v-else>{{ t('clip.manager.no_match', { search: query }) }}</span>
          </div>

          <div v-else class="space-y-2">
            <template v-for="group in groupedItems" :key="group.category">
              <div class="text-[10px] font-semibold text-dimmed uppercase tracking-wider px-1 pt-2 pb-0.5">
                {{ group.category }}
              </div>
              <div
                v-for="item in group.items"
                :key="item.id"
                class="group rounded p-2.5 cursor-pointer"
                :class="isSelected(item.id) ? 'bg-primary/15 ring-1 ring-inset ring-primary/50' : 'bg-elevated/30 hover:bg-elevated/60'"
                @click="onRowClick(item.id, $event)"
              >
                <div class="flex items-center gap-2">
                  <span
                    class="shrink-0 transition-opacity"
                    :class="isSelected(item.id) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'"
                    @click.stop
                  >
                    <UCheckbox
                      :model-value="isSelected(item.id)"
                      size="sm"
                      @update:model-value="selClick(item.id, { ctrl: true })"
                    />
                  </span>
                  <UIcon name="i-tabler-movie" class="size-4 text-primary shrink-0" />
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium truncate">{{ item.label || t('clip.manager.untitled') }}</div>
                    <div class="text-[10px] text-dimmed flex items-center gap-1.5 truncate mt-0.5">
                      <span>{{ formatDuration(item.durationUs) }}</span>
                      <span>· {{ item.eventCount }} {{ t('clip.manager.events') }}</span>
                      <span v-if="item.tags?.length" class="truncate">· {{ item.tags.join(', ') }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </div>

        <!-- 底部双态 -->
        <div class="flex items-center justify-between gap-3 pt-2 border-t border-default">
          <div v-if="selected.size === 0" class="text-[11px] text-dimmed">
            {{ t('library.toolbar.total', { n: pageResult.total }) }}
          </div>
          <div v-else class="flex items-center gap-1.5 min-w-0">
            <span class="text-[11px] text-toned shrink-0">{{ t('library.batch.selected_n', { n: selected.size }) }}</span>
            <UDropdownMenu :items="batchMenuItems">
              <UButton size="xs" variant="soft" color="primary" icon="i-tabler-stack-2" trailing-icon="i-tabler-chevron-down">
                {{ t('library.batch.menu') }}
              </UButton>
            </UDropdownMenu>
            <UButton size="xs" variant="ghost" color="neutral" icon="i-tabler-x" :title="t('library.batch.clear')" @click="selClear()" />
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

      <ClipDetailPanel class="max-h-[65vh]" :clip-id="anchor" />
    </div>
  </BaseModal>

  <!-- 批量加标签 -->
  <BaseModal v-model:open="batchTagsOpen" :title="t('library.batch.add_tags_title')" icon="i-tabler-tags" size="md">
    <UInputMenu v-model="batchTags" multiple creatable :items="allTags" size="sm" :placeholder="t('library.batch.add_tags_placeholder')" />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchTagsOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" :disabled="batchTags.length === 0" @click="onBatchAddTags">{{ t('library.batch.add_tags_apply') }}</UButton>
    </template>
  </BaseModal>

  <!-- 批量改分类 -->
  <BaseModal v-model:open="batchCategoryOpen" :title="t('library.batch.change_category_title')" icon="i-tabler-category" size="md">
    <UInputMenu v-model="batchCategory" creatable :items="allCategories" size="sm" :placeholder="t('library.batch.change_category_placeholder')" />
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="batchCategoryOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" @click="onBatchChangeCategory">{{ t('library.batch.change_category_apply') }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useConfirm } from '@/composables/useConfirm'
import { useLocalStorage } from '@vueuse/core'
import { backend } from '@/lib/backend'
import { useClipsStore, type ClipSummary } from '@/stores/clips'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { useAutoFocusOnOpen } from '@/composables/editor/useAutoFocusOnOpen'
import { useListSelection } from '@/composables/editor/useListSelection'
import { filterSubgraphs, groupByCategory, paginate } from '@/lib/libraryFilter'
import BaseModal from '@/components/common/BaseModal.vue'
import ClipDetailPanel from '@/components/containers/ClipDetailPanel.vue'

const { t } = useI18n()
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ 'update:open': [v: boolean] }>()
const modelOpen = useDialogOpen(props, emit)

const store = useClipsStore()
const { confirm } = useConfirm()

const query = ref('')
const searchInputRef = ref<any>(null)

const sortKey = ref<'label' | 'createdAt' | 'duration'>('label')
const sortDesc = ref(false)
const sortItems = computed(() => [
  { label: t('clip.manager.view_by_name'), value: 'label' },
  { label: t('clip.manager.view_by_created'), value: 'createdAt' },
  { label: t('clip.manager.view_by_duration'), value: 'duration' },
])

// ClipSummary 自带 label/description/category/tags → libraryFilter 直接吃.
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

const categoryFilter = ref<string>('all')
const tagFilter = ref<string[]>([])
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
    query: query.value,
    category: categoryFilter.value === 'all' ? null : categoryFilter.value === 'none' ? '' : categoryFilter.value.slice(2),
    tags: tagFilter.value,
  })
  const sorted = [...arr]
  sorted.sort((a, b) => {
    let cmp = 0
    switch (sortKey.value) {
      case 'label': cmp = (a.label ?? '').localeCompare(b.label ?? ''); break
      case 'createdAt': cmp = (a.createdAt ?? '').localeCompare(b.createdAt ?? ''); break
      case 'duration': cmp = a.durationUs - b.durationUs; break
    }
    return sortDesc.value ? -cmp : cmp
  })
  return sorted
})

const page = ref(1)
const pageSize = useLocalStorage('clip.pageSize', 50)
const pageSizeItems = computed(() => [20, 50, 100].map((n) => ({ label: t('library.toolbar.per_page', { n }), value: n })))
const pageResult = computed(() => paginate(filteredItems.value, page.value, pageSize.value))
const groupedItems = computed(() => groupByCategory(pageResult.value.pageItems, t('library.explorer.uncategorized')))

watch([query, categoryFilter, tagFilter, pageSize, sortKey, sortDesc], () => { page.value = 1 })
watch(() => pageResult.value.totalPages, (tp) => { if (page.value > tp) page.value = tp })

const visibleIds = computed(() => groupedItems.value.flatMap((g) => g.items.map((i) => i.id)))
const { selected, anchor, click: selClick, clear: selClear, isSelected } = useListSelection(visibleIds)

useAutoFocusOnOpen(modelOpen, searchInputRef, {
  onOpen: () => {
    void store.refresh()
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

function byId(id: string): ClipSummary | undefined {
  return store.clips.find((c) => c.id === id)
}

// ── 批量 ──
const batchMenuItems = computed(() => [
  [
    { label: t('library.batch.add_tags'), icon: 'i-tabler-tags', onSelect: () => { batchTagsOpen.value = true } },
    { label: t('library.batch.change_category'), icon: 'i-tabler-category', onSelect: () => { batchCategoryOpen.value = true } },
  ],
  [
    { label: t('library.batch.delete'), icon: 'i-tabler-trash', color: 'error' as const, onSelect: () => { void onBatchDelete() } },
  ],
])

const batchTagsOpen = ref(false)
const batchTags = ref<string[]>([])
async function onBatchAddTags() {
  const add = batchTags.value.map((s) => s.trim()).filter(Boolean)
  if (add.length === 0) { batchTagsOpen.value = false; return }
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
  const desc = referenced.length > 0
    ? t('clip.manager.batch_delete_confirm_referenced', { n: ids.length, refs: referenced.length })
    : t('clip.manager.batch_delete_confirm', { n: ids.length })
  const yes = await confirm({ title: t('clip.manager.batch_delete_title'), description: desc, color: 'error', confirmText: t('common.delete') })
  if (yes !== true) return
  for (const id of ids) await store.remove(id)
  await store.refresh()
  selClear()
}
</script>
