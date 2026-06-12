<!-- 子图库 modal (编辑器内唯一库管理入口). 入口: toolbar 📚 / 左 rail.
     本地 tab: 单击插引用; 右键 增删改查; 悬停右栏详情。在线 tab: 占位 (跨机分享留口)。 -->
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

    <!-- 本地: 列表 + 详情双栏 -->
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

        <div class="max-h-[60vh] overflow-y-auto">
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
            <!-- Group by primary tag -->
            <template v-for="group in groupedItems" :key="group.tag">
              <div class="text-[10px] font-semibold text-dimmed uppercase tracking-wider px-1 pt-2 pb-0.5">
                {{ group.tag }}
              </div>
              <UContextMenu v-for="item in group.items" :key="item.id" :items="ctxMenuItems(item)">
                <div
                  class="rounded p-3 cursor-pointer"
                  :class="detailID === item.id ? 'bg-elevated/60' : 'bg-elevated/30 hover:bg-elevated/60'"
                  @click="onPick(item.id)"
                  @mouseenter="onHoverRow(item.id)"
                  @contextmenu="detailID = item.id"
                >
                  <div class="flex items-start gap-2">
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
                          v-for="t in item.tags"
                          :key="t"
                          class="px-1.5 py-0 bg-elevated/60 text-[9px] rounded text-dimmed"
                        >
                          {{ t }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </UContextMenu>
            </template>
          </div>
        </div>
      </div>

      <LibraryDetailPanel class="max-h-[65vh]" :sgID="detailID" @cleared="detailID = null" />
    </div>
  </BaseModal>

  <!-- 编辑信息 (名称/描述/标签) — merge patch + rev 乐观锁 -->
  <BaseModal v-model:open="editOpen" :title="t('library.explorer.edit_title')" icon="i-tabler-pencil" size="md">
    <div class="space-y-3">
      <div class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.name') }}</label>
        <UInput v-model="editForm.label" size="sm" />
      </div>
      <div class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.description') }}</label>
        <UTextarea v-model="editForm.description" :rows="3" size="sm" />
      </div>
      <div class="space-y-1.5">
        <label class="block text-xs text-toned">{{ t('common.tags') }}</label>
        <UInput v-model="editForm.tags" size="sm" :placeholder="t('library.explorer.tags_hint')" />
      </div>
    </div>
    <template #footer>
      <UButton variant="ghost" color="neutral" @click="editOpen = false">{{ t('common.cancel') }}</UButton>
      <UButton color="primary" :disabled="!editForm.label.trim()" @click="onSaveEdit">{{ t('common.save') }}</UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useLibraryStore } from '@/stores/library'
import { useConfirm } from '@/composables/useConfirm'
import { useDialogOpen } from '@/composables/editor/useDialogOpen'
import { useAutoFocusOnOpen } from '@/composables/editor/useAutoFocusOnOpen'
import BaseModal from '@/components/common/BaseModal.vue'
import LibraryDetailPanel from '@/components/containers/LibraryDetailPanel.vue'
import { backend, type Subgraph } from '@/lib/backend'
import { errorMessage, toastError } from '@/lib/invoke'

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

onMounted(() => refreshLibrary())
useAutoFocusOnOpen(modelOpen, searchInputRef, {
  onOpen: () => { void refreshLibrary(); query.value = '' },
})

const filteredItems = computed<Subgraph[]>(() => {
  const q = query.value.toLowerCase().trim()
  if (!q) return lib.subgraphs
  return lib.subgraphs.filter((item) => {
    const hay =
      `${item.label} ${item.description ?? ''} ${(item.tags ?? []).join(' ')}`.toLowerCase()
    return hay.includes(q)
  })
})

// Group by primary tag (first tag); untagged go under "(未分类)"
interface TagGroup {
  tag: string
  items: Subgraph[]
}

const groupedItems = computed<TagGroup[]>(() => {
  const map = new Map<string, Subgraph[]>()
  for (const item of filteredItems.value) {
    const primaryTag = (item.tags ?? [])[0] ?? t('library.explorer.uncategorized')
    if (!map.has(primaryTag)) map.set(primaryTag, [])
    map.get(primaryTag)!.push(item)
  }
  return Array.from(map.entries()).map(([tag, items]) => ({ tag, items }))
})

function onPick(libraryID: string) {
  emit('pick-subgraph', libraryID)
  modelOpen.value = false
}

// 详情栏跟最后悬停/右键的行 (粘滞, 不随 mouseleave 清空 — 鼠标能移去面板操作)。
// 悬停去抖 250ms: detailID 一变 LibraryDetailPanel 就拉 referrers, 不去抖扫库会刷成 RPC 雨。
const detailID = ref<string | null>(null)
let hoverTimer = 0
function onHoverRow(id: string) {
  window.clearTimeout(hoverTimer)
  hoverTimer = window.setTimeout(() => { detailID.value = id }, 250)
}
onBeforeUnmount(() => window.clearTimeout(hoverTimer))

function ctxMenuItems(item: Subgraph) {
  return [
    [
      { label: t('library.explorer.insert'), icon: 'i-tabler-package-import', onSelect: () => onPick(item.id) },
      { label: t('library.card.duplicate'), icon: 'i-tabler-copy-plus', onSelect: () => onDuplicate(item) },
      { label: t('library.explorer.edit_info'), icon: 'i-tabler-pencil', onSelect: () => openEdit(item) },
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
  if (ok) {
    if (detailID.value === item.id) detailID.value = null
  } else {
    toast.add({ title: t('toast.delete_failed'), color: 'error' })
  }
}

// ── 编辑信息 (改名/描述/标签) ──
const editOpen = ref(false)
const editTarget = ref<Subgraph | null>(null)
const editForm = ref({ label: '', description: '', tags: '' })

function openEdit(item: Subgraph) {
  editTarget.value = item
  editForm.value = {
    label: item.label ?? '',
    description: item.description ?? '',
    tags: (item.tags ?? []).join(', '),
  }
  editOpen.value = true
}

async function onSaveEdit() {
  const sg = editTarget.value
  if (!sg) return
  const tags = editForm.value.tags.split(',').map((s) => s.trim()).filter(Boolean)
  const patch = {
    label: editForm.value.label.trim(),
    description: editForm.value.description.trim(),
    tags,
  }
  // 裸版本 + try/catch: error-only RPC 经 invoke 包装后成败同为 undefined, 辨不出结果。
  try {
    await backend.subgraphs.updateSilent(sg.id, JSON.stringify(patch), sg.rev)
  } catch (e) {
    toastError(errorMessage(e))
    return
  }
  await lib.reload()
  editOpen.value = false
}
</script>
