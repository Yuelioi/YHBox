<!-- 模板管理 modal (编辑器内入口: 左 rail 📷). 两栏: 左列表(搜索/分类·标签过滤/分组/多选/分页) + 右 TemplateDetailPanel.
     批量: 选中后下拉改标签/改分类/删. 排序: 名称/创建/变体 × 正逆序. 模板全局资产; 复刻子图库定式, 复用 libraryFilter/useListSelection.
     新建截图: tplStore.containerId(编辑器注入)定位当前容器目标窗口. -->
<template>
  <BaseModal v-model:open="modelOpen" :title="t('template.manager.title')" icon="i-tabler-photo" size="5xl">
    <template #header-extra>
      <UButton color="primary" icon="i-tabler-camera" size="xs" class="mr-2" @click="onNewTemplate">
        {{ t('template.capture.title') }}
      </UButton>
    </template>

    <div class="flex gap-4">
      <div class="flex-1 min-w-0 space-y-3">
        <div class="flex items-center gap-3">
          <UInput
            ref="searchInputRef"
            v-model="query"
            :placeholder="t('template.manager.search')"
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
            :title="sortDesc ? t('template.manager.sort_desc') : t('template.manager.sort_asc')"
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
            <span v-if="entries.length === 0">{{ t('template.manager.empty') }}</span>
            <span v-else>{{ t('template.manager.no_match', { search: query }) }}</span>
          </div>

          <div v-else class="space-y-2">
            <template v-for="group in groupedItems" :key="group.category">
              <div class="text-[10px] font-semibold text-dimmed uppercase tracking-wider px-1 pt-2 pb-0.5">
                {{ group.category }}
              </div>
              <div
                v-for="item in group.items"
                :key="item.guid"
                class="group rounded p-2.5 cursor-pointer"
                :class="rowActive(item.guid) ? 'bg-primary/15 ring-1 ring-inset ring-primary/50' : 'bg-elevated/30 hover:bg-elevated/60'"
                @click="onRowClick(item.guid, $event)"
              >
                <div class="flex items-center gap-2">
                  <span
                    class="shrink-0 transition-opacity"
                    :class="pickMode || rowActive(item.guid) ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'"
                    @click.stop
                  >
                    <UCheckbox
                      :model-value="rowActive(item.guid)"
                      size="sm"
                      @update:model-value="onRowCheckbox(item.guid)"
                    />
                  </span>
                  <UIcon name="i-tabler-photo" class="size-4 text-primary shrink-0" />
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium truncate">{{ item.name || item.guid }}</div>
                    <div class="text-[10px] text-dimmed flex items-center gap-1.5 truncate mt-0.5">
                      <span>{{ item.variantCount }} {{ t('template.manager.variant_count') }}</span>
                      <span v-if="item.tags?.length" class="truncate">· {{ item.tags.join(', ') }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </div>

        <!-- 底部双态: 无选中=计数+分页; 有选中=批量+分页 -->
        <div class="flex items-center justify-between gap-3 pt-2 border-t border-default">
          <!-- pick 模式: 已选用于节点 + 完成 -->
          <div v-if="pickMode" class="flex items-center gap-2 min-w-0">
            <span class="text-[11px] text-toned">{{ t('template.picker.selected_count', { n: assigned.length }) }}</span>
            <UButton size="xs" color="primary" @click="modelOpen = false">{{ t('template.picker.done') }}</UButton>
          </div>
          <div v-else-if="selected.size === 0" class="text-[11px] text-dimmed">
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

      <TemplateDetailPanel
        class="max-h-[65vh]"
        :guid="anchor"
        :pick-mode="pickMode"
        :assigned="anchor ? isAssigned(anchor) : false"
        @toggle-assign="anchor && toggleAssign(anchor)"
      />
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
import { useToast } from '@nuxt/ui/composables'
import { useLocalStorage } from '@vueuse/core'
import { backend, type AssetSummary } from '@/lib/backend'
import { useTemplatesStore } from '@/stores/templates'
import { useConfirm } from '@/composables/useConfirm'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { useAutoFocusOnOpen } from '@/composables/editor/useAutoFocusOnOpen'
import { useListSelection } from '@/composables/editor/useListSelection'
import { filterSubgraphs, groupByCategory, paginate } from '@/lib/libraryFilter'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import BaseModal from '@/components/common/BaseModal.vue'
import TemplateDetailPanel from '@/components/containers/TemplateDetailPanel.vue'

const { t } = useI18n()
const props = defineProps<{ open: boolean; pickMode?: boolean; modelValue?: string[] }>()
const emit = defineEmits<{ 'update:open': [v: boolean]; 'update:modelValue': [v: string[]] }>()
const modelOpen = useDialogOpen(props, emit)

// pick 模式: 行勾选框=指派给节点 (按 modelValue 回显); 管理模式=批量选 (useListSelection, isSelected/selClick 见下).
const assigned = computed<string[]>(() => props.modelValue ?? [])
function isAssigned(guid: string) { return assigned.value.includes(guid) }
function toggleAssign(guid: string) {
  emit('update:modelValue', isAssigned(guid) ? assigned.value.filter((g) => g !== guid) : [...assigned.value, guid])
}
function rowActive(guid: string) { return props.pickMode ? isAssigned(guid) : isSelected(guid) }
function onRowCheckbox(guid: string) {
  if (props.pickMode) toggleAssign(guid)
  else selClick(guid, { ctrl: true })
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
const entries = computed<TplItem[]>(() => Object.values(tplStore.map).map((s) => ({ ...s, label: s.name })))

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
    category: categoryFilter.value === 'all' ? null : categoryFilter.value === 'none' ? '' : categoryFilter.value.slice(2),
    tags: tagFilter.value,
  })
  const sorted = [...arr]
  sorted.sort((a, b) => {
    let cmp = 0
    switch (sortKey.value) {
      case 'name': cmp = (a.name ?? '').localeCompare(b.name ?? ''); break
      case 'createdAt': cmp = (a.createdAt ?? '').localeCompare(b.createdAt ?? ''); break
      case 'variantCount': cmp = (a.variantCount ?? 0) - (b.variantCount ?? 0); break
    }
    return sortDesc.value ? -cmp : cmp
  })
  return sorted
})

// 分页 + 分组
const page = ref(1)
const pageSize = useLocalStorage('template.pageSize', 50)
const pageSizeItems = computed(() => [20, 50, 100].map((n) => ({ label: t('library.toolbar.per_page', { n }), value: n })))
const pageResult = computed(() => paginate(filteredItems.value, page.value, pageSize.value))
const groupedItems = computed(() => groupByCategory(pageResult.value.pageItems, t('library.explorer.uncategorized')))

watch([query, categoryFilter, tagFilter, pageSize, sortKey, sortDesc], () => { page.value = 1 })
watch(() => pageResult.value.totalPages, (tp) => { if (page.value > tp) page.value = tp })

// 多选 + 详情跟锚点
const visibleIds = computed(() => groupedItems.value.flatMap((g) => g.items.map((i) => i.guid)))
const { selected, anchor, click: selClick, clear: selClear, isSelected } = useListSelection(visibleIds)

useAutoFocusOnOpen(modelOpen, searchInputRef, {
  onOpen: () => {
    void tplStore.reload()
    query.value = ''
    categoryFilter.value = 'all'
    tagFilter.value = []
    page.value = 1
    selClear()
  },
})

function onRowClick(id: string, e: MouseEvent) {
  if (props.pickMode) { selClick(id); return } // pick: 行点击只设详情锚点, 不做批量多选
  selClick(id, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey })
}

// 新建截图 (搬自原 TemplatesTab)
async function onNewTemplate() {
  const id = 'tpl-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
  const waiter = awaitWailsEvent<{ id: string; mode: string; payload: any }>('tools:picker-result', (p) => p?.id === id)
  await backend.tools.openScreenPicker('template_save', id, tplStore.containerId)
  const result = await waiter
  if (!result.payload?.cancelled) {
    await tplStore.reload()
    if (props.pickMode && result.payload?.guid && !isAssigned(result.payload.guid)) {
      emit('update:modelValue', [...assigned.value, result.payload.guid])
    }
  }
}

// ── 批量: 直接发 RPC (不逐次 reload), 最后 reload 一次 ──
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
  const desc = referenced.length > 0
    ? t('template.manager.batch_delete_confirm_referenced', { n: ids.length, refs: referenced.length })
    : t('template.manager.batch_delete_confirm', { n: ids.length })
  const yes = await confirm({ title: t('template.manager.batch_delete_title'), description: desc, color: 'error', confirmText: t('common.delete') })
  if (yes !== true) return
  for (const guid of ids) await backend.assets.delete_(guid)
  await tplStore.reload()
  selClear()
}
</script>
