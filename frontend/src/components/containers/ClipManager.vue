<!-- clip 全局资产管理 (编辑器内入口: 左 rail 🎬). 列表式 — clip 是输入事件录制, 无可视缩略图.
     行: 名称(双击改名) / 时长 / 事件数 / 标签; 删除前扫引用 (PlayClip 节点). 读全局 clipsStore. -->
<template>
  <div class="space-y-3">
    <div class="flex items-center gap-2 flex-wrap">
      <UInput
        v-model="search"
        icon="i-tabler-search"
        :placeholder="t('clip.manager.search')"
        class="flex-1 min-w-48 max-w-md"
        size="sm"
      />
      <USelect v-model="sortKey" :items="sortItems" size="sm" class="w-44" />
      <UButton
        size="xs"
        variant="ghost"
        :icon="sortDesc ? 'i-tabler-sort-descending' : 'i-tabler-sort-ascending'"
        :title="sortDesc ? t('clip.manager.sort_desc') : t('clip.manager.sort_asc')"
        @click="sortDesc = !sortDesc"
      />
      <span class="text-xs text-dimmed">{{ filtered.length }} / {{ entries.length }}</span>
    </div>

    <!-- 空态 -->
    <div
      v-if="entries.length === 0"
      class="rounded-md border border-dashed border-default/60 bg-elevated/40 py-12 px-6 text-center"
    >
      <UIcon name="i-tabler-movie-off" class="size-8 text-dimmed mx-auto mb-2" />
      <p class="text-sm text-dimmed">{{ t('clip.manager.empty') }}</p>
      <p class="text-[11px] text-dimmed mt-1">{{ t('clip.manager.empty_hint') }}</p>
    </div>

    <!-- 过滤后空态 -->
    <div
      v-else-if="filtered.length === 0"
      class="rounded-md border border-dashed border-default/60 bg-elevated/40 py-10 px-6 text-center text-sm text-dimmed"
    >
      {{ t('clip.manager.no_match', { search }) }}
    </div>

    <!-- 列表 -->
    <div v-else class="space-y-1.5">
      <div
        v-for="c in filtered"
        :key="c.id"
        class="group flex items-center gap-3 rounded-md border border-default bg-elevated/30 px-3 py-2 hover:border-primary/60 transition-colors"
      >
        <UIcon name="i-tabler-movie" class="size-4 text-primary shrink-0" />
        <div class="flex-1 min-w-0">
          <UInput
            v-if="editingId === c.id"
            v-model="editLabel"
            size="xs"
            :autofocus="true"
            @keydown.enter="commitRename(c)"
            @keydown.escape="editingId = ''"
            @blur="commitRename(c)"
          />
          <div
            v-else
            class="text-sm text-highlighted truncate cursor-text"
            :title="t('clip.manager.rename_tip')"
            @dblclick="startRename(c)"
          >
            {{ c.label || t('clip.manager.untitled') }}
          </div>
          <div class="text-[10px] text-dimmed flex items-center gap-2 truncate mt-0.5">
            <span class="inline-flex items-center gap-0.5">
              <UIcon name="i-tabler-clock" class="size-3" /> {{ formatDuration(c.durationUs) }}
            </span>
            <span class="inline-flex items-center gap-0.5">
              <UIcon name="i-tabler-list" class="size-3" /> {{ c.eventCount }} {{ t('clip.manager.events') }}
            </span>
            <span v-if="c.tags?.length" class="truncate">· {{ c.tags.join(', ') }}</span>
          </div>
        </div>
        <UButton
          size="xs"
          variant="ghost"
          :color="copiedId === c.id ? 'success' : 'neutral'"
          :icon="copiedId === c.id ? 'i-tabler-check' : 'i-tabler-copy'"
          :title="copiedId === c.id ? t('common.copied') : t('clip.manager.copy_id')"
          @click="copyId(c.id)"
        />
        <UButton
          size="xs"
          variant="ghost"
          color="error"
          icon="i-tabler-trash"
          :title="t('clip.manager.delete_tip', { name: c.label })"
          @click="onDelete(c)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { useClipsStore, type ClipSummary } from '@/stores/clips'
import { backend } from '@/lib/backend'
import { errorMessage } from '@/lib/invoke'

const { t } = useI18n()
const store = useClipsStore()
const toast = useToast()
const { confirm } = useConfirm()

const search = ref('')
const sortKey = ref<'label' | 'createdAt' | 'duration'>('label')
const sortDesc = ref(false)

const sortItems = computed(() => [
  { label: t('clip.manager.view_by_name'), value: 'label' },
  { label: t('clip.manager.view_by_created'), value: 'createdAt' },
  { label: t('clip.manager.view_by_duration'), value: 'duration' },
])

const entries = computed(() => store.clips)

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  let arr = entries.value
  if (q) {
    arr = arr.filter(
      (c) =>
        c.label?.toLowerCase().includes(q) ||
        c.id.toLowerCase().includes(q) ||
        c.tags?.some((tag) => tag.toLowerCase().includes(q)),
    )
  }
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

function formatDuration(us: number): string {
  const ms = us / 1000
  if (ms < 1000) return `${Math.round(ms)} ms`
  return `${(ms / 1000).toFixed(1)} s`
}

// ── 改名 (双击行内编辑) ──
const editingId = ref('')
const editLabel = ref('')
function startRename(c: ClipSummary) {
  editingId.value = c.id
  editLabel.value = c.label
}
async function commitRename(c: ClipSummary) {
  if (editingId.value !== c.id) return
  const next = editLabel.value.trim()
  editingId.value = ''
  if (!next || next === c.label) return
  await store.update(c.id, { label: next, description: c.description ?? '', tags: c.tags ?? [] })
}

// ── 复制 ID ──
const copiedId = ref('')
let copiedTimer = 0
async function copyId(id: string) {
  try {
    await navigator.clipboard.writeText(id)
    copiedId.value = id
    clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => (copiedId.value = ''), 1500)
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

// ── 删除 (先扫引用; clip 是 KindClip 资产, referrers 找 PlayClip 节点) ──
async function onDelete(c: ClipSummary) {
  const refs = await backend.assets.referrers(c.id)
  const n = refs?.length ?? 0
  const description =
    n > 0
      ? t('clip.manager.delete_confirm_referenced', { name: c.label, n })
      : t('clip.manager.delete_confirm', { name: c.label })
  const yes = await confirm({
    title: t('clip.manager.delete_title'),
    description,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  await store.remove(c.id)
}
</script>
