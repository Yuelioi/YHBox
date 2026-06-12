<template>
  <UContextMenu :items="ctxMenuItems">
    <div
      :class="[
        'rounded-md border bg-elevated/30 transition-colors cursor-pointer relative group',
        viewMode === 'list' ? 'p-2 flex items-center gap-3' : 'p-3',
        selected ? 'border-primary ring-2 ring-primary/40' : 'border-default hover:border-primary/50',
      ]"
      @click="$emit('select', sg.id)"
    >
      <div
        v-if="viewMode === 'list'"
        class="size-7 rounded-md flex items-center justify-center shrink-0 bg-fuchsia-500/15 border border-fuchsia-500/40"
      >
        <UIcon name="i-tabler-subtask" class="size-4 text-fuchsia-300" />
      </div>

      <div v-if="viewMode === 'list'" class="min-w-0 flex-1 flex items-center gap-2">
        <span class="text-sm font-medium truncate text-default">{{ displayName }}</span>
        <span v-if="sg.description" class="text-[11px] text-dimmed truncate flex-1">{{ sg.description }}</span>
        <UBadge size="xs" variant="subtle" color="neutral" class="shrink-0">
          {{ t('library.card.node_count', { n: nodeCount }) }}
        </UBadge>
      </div>

      <template v-else>
        <div class="flex items-center gap-2 mb-2">
          <UIcon name="i-tabler-subtask" class="size-4 text-fuchsia-300" />
          <span class="text-sm font-medium truncate flex-1">{{ displayName }}</span>
          <UBadge size="xs" variant="subtle" color="neutral">{{ nodeCount }}</UBadge>
        </div>
        <p v-if="sg.description" class="text-[11px] text-dimmed mb-2 line-clamp-2">{{ sg.description }}</p>
        <div class="flex gap-1 flex-wrap">
          <UBadge v-for="tag in sg.tags ?? []" :key="tag" size="xs" variant="outline" color="neutral">
            {{ tag }}
          </UBadge>
        </div>
      </template>
    </div>
  </UContextMenu>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Subgraph } from '@/lib/backend'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { errorMessage } from '@/lib/invoke'
import { useLibraryStore } from '@/stores/library'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    sg: Subgraph
    selected?: boolean
    viewMode?: 'grid' | 'list'
  }>(),
  { selected: false, viewMode: 'grid' },
)

const emit = defineEmits<{
  select: [sgID: string]
}>()

const libraryStore = useLibraryStore()
const toast = useToast()
const { confirm } = useConfirm()

const displayName = computed(() => props.sg.label || props.sg.id)
const nodeCount = computed(() => props.sg.graph?.nodes?.length ?? 0)

const ctxMenuItems = computed(() => [
  [
    {
      label: t('library.card.view_detail'),
      icon: 'i-tabler-info-circle',
      onSelect: () => emit('select', props.sg.id),
    },
    {
      label: t('library.card.duplicate'),
      icon: 'i-tabler-copy-plus',
      onSelect: () => onDuplicate(),
    },
  ],
  [
    {
      label: t('library.card.copy_id'),
      icon: 'i-tabler-copy',
      onSelect: () => onCopyID(),
    },
  ],
  [
    {
      label: t('library.card.delete'),
      icon: 'i-tabler-trash',
      color: 'error' as const,
      onSelect: () => onDelete(),
    },
  ],
])

async function onCopyID() {
  try {
    await navigator.clipboard.writeText(props.sg.id)
    toast.add({ title: t('toast.copy_id_success'), color: 'success', icon: 'i-tabler-check', duration: 1500 })
  } catch (e: any) {
    toast.add({ title: t('toast.copy_failed'), description: errorMessage(e), color: 'error' })
  }
}

// 复制为新子图 (fork, ≈Blender Make Local): 想独立改不影响引用方时用。
async function onDuplicate() {
  const dup = await libraryStore.duplicateSubgraph(props.sg.id)
  if (dup) {
    toast.add({ title: t('library.card.duplicated', { name: dup.label }), color: 'success', icon: 'i-tabler-check' })
  }
}

// 删除安全: 先扫引用 — 被容器使用时警告里带"被 N 个容器使用", 确认才真删。
async function onDelete() {
  const refs = await libraryStore.referrersOf(props.sg.id)
  const useCount = libraryStore.containerUseCount(refs)
  const desc = useCount > 0
    ? t('library.card.delete_confirm_referenced', { name: displayName.value, n: useCount })
    : t('library.card.delete_confirm_desc', { name: displayName.value })
  const yes = await confirm({
    title: t('library.card.delete_confirm_title'),
    description: desc,
    color: 'error',
    confirmText: t('common.delete'),
  })
  if (yes !== true) return
  const ok = await libraryStore.deleteSubgraph(props.sg.id)
  if (!ok) {
    toast.add({ title: t('toast.delete_failed'), color: 'error' })
  }
}
</script>
