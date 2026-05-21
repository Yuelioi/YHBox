<template>
  <UContextMenu :items="ctxMenuItems">
    <div
      :class="[
        'rounded-md border bg-elevated/30 transition-colors cursor-pointer relative group',
        viewMode === 'list' ? 'p-2 flex items-center gap-3' : 'p-3',
        selected ? 'border-primary ring-2 ring-primary/40' : 'border-default hover:border-primary/50',
      ]"
      @click="$emit('select', sgID)"
    >
      <div
        v-if="viewMode === 'list'"
        class="size-7 rounded-md flex items-center justify-center shrink-0 bg-fuchsia-500/15 border border-fuchsia-500/40"
      >
        <UIcon name="i-tabler-package" class="size-4 text-fuchsia-300" />
      </div>

      <div v-if="viewMode === 'list'" class="min-w-0 flex-1 flex items-center gap-2">
        <span class="text-sm font-medium truncate text-default">{{ displayName }}</span>
        <span v-if="pkg.root.description" class="text-[11px] text-dimmed truncate flex-1">{{ pkg.root.description }}</span>
        <UBadge size="xs" variant="subtle" color="neutral" class="shrink-0">
          {{ embeddedCount }} 子图
        </UBadge>
      </div>

      <template v-else>
        <div class="flex items-center gap-2 mb-2">
          <UIcon name="i-tabler-package" class="size-4 text-fuchsia-300" />
          <span class="text-sm font-medium truncate flex-1">{{ displayName }}</span>
          <UBadge size="xs" variant="subtle" color="neutral">{{ embeddedCount }}</UBadge>
        </div>
        <p v-if="pkg.root.description" class="text-[11px] text-dimmed mb-2 line-clamp-2">{{ pkg.root.description }}</p>
        <div class="flex gap-1 flex-wrap">
          <UBadge v-if="pkg.templates.length > 0" size="xs" variant="outline" color="neutral">
            <UIcon name="i-tabler-photo" class="size-3 mr-0.5" />{{ pkg.templates.length }} 模板
          </UBadge>
          <UBadge v-if="pkg.clips.length > 0" size="xs" variant="outline" color="neutral">
            <UIcon name="i-tabler-vinyl" class="size-3 mr-0.5" />{{ pkg.clips.length }} 片段
          </UBadge>
        </div>
      </template>
    </div>
  </UContextMenu>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { SubgraphPackage } from '@/lib/backend'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { useLibraryStore } from '@/stores/library'

const props = withDefaults(
  defineProps<{
    sgID: string
    pkg: SubgraphPackage
    selected?: boolean
    viewMode?: 'grid' | 'list'
  }>(),
  { selected: false, viewMode: 'grid' },
)

const emit = defineEmits<{
  select: [sgID: string]
  import: [sgID: string]
}>()

const libraryStore = useLibraryStore()
const toast = useToast()
const { confirm } = useConfirm()

const displayName = computed(() => props.pkg.root.label || props.sgID)
const embeddedCount = computed(() => 1 + Object.keys(props.pkg.embedded ?? {}).length)

const ctxMenuItems = computed(() => [
  [
    {
      label: '查看详情',
      icon: 'i-tabler-info-circle',
      onSelect: () => emit('select', props.sgID),
    },
    {
      label: '导入到容器',
      icon: 'i-tabler-arrow-bar-to-down',
      onSelect: () => emit('import', props.sgID),
    },
  ],
  [
    {
      label: '复制 ID',
      icon: 'i-tabler-copy',
      onSelect: () => onCopyID(),
    },
  ],
  [
    {
      label: '删除',
      icon: 'i-tabler-trash',
      color: 'error' as const,
      onSelect: () => onDelete(),
    },
  ],
])

async function onCopyID() {
  try {
    await navigator.clipboard.writeText(props.sgID)
    toast.add({ title: '已复制 ID', color: 'success', icon: 'i-tabler-check' })
  } catch (e: any) {
    toast.add({ title: '复制失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onDelete() {
  const yes = await confirm({
    title: '删除库子图',
    description: `确认删除 "${displayName.value}"？此操作不可恢复。`,
    color: 'error',
    confirmText: '删除',
  })
  if (yes !== true) return
  const ok = await libraryStore.deletePackage(props.sgID)
  if (ok) {
    toast.add({ title: '已删除', color: 'success' })
  } else {
    toast.add({ title: '删除失败', color: 'error' })
  }
}
</script>
