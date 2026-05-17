<!--
LibraryCard 库列表卡片（子图 / 模板共用）。

支持两种 view mode：
  - grid (默认)：紧凑 3 列卡片
  - list：单行 row，icon + label + 描述 + tags + 右侧 meta

交互：
  - 单击卡片 → emit('select', item)
  - 双击 → emit('open', item)（子图 = 进容器编辑器；template = 打开 detail）
  - 右键 → UContextMenu（重命名占位 / 复制 ID / 删除 / 导入到当前容器）
  - 拖拽源（dataTransfer 'application/yhfish-library-item'）— 容器编辑器接 copy-on-use
-->
<template>
  <UContextMenu :items="ctxMenuItems">
    <div
      :class="[
        'rounded-md border bg-elevated/30 transition-colors cursor-grab relative group',
        viewMode === 'list' ? 'p-2 flex items-center gap-3' : 'p-3',
        selected ? 'border-primary ring-2 ring-primary/40' : 'border-default hover:border-primary/50',
      ]"
      :draggable="true"
      @dragstart="onDragStart"
      @click="$emit('select', item)"
      @dblclick="$emit('open', item)"
    >
      <!-- list mode 左侧 icon 圆 -->
      <div
        v-if="viewMode === 'list'"
        class="size-7 rounded-md flex items-center justify-center shrink-0"
        :class="item.kind === 'subgraph'
          ? 'bg-fuchsia-500/15 border border-fuchsia-500/40'
          : 'bg-violet-500/15 border border-violet-500/40'"
      >
        <UIcon
          :name="item.kind === 'subgraph' ? 'i-tabler-package' : 'i-tabler-photo'"
          class="size-4"
          :class="item.kind === 'subgraph' ? 'text-fuchsia-300' : 'text-violet-300'"
        />
      </div>

      <!-- 内容区 -->
      <div v-if="viewMode === 'list'" class="min-w-0 flex-1 flex items-center gap-2">
        <span class="text-sm font-medium truncate text-default">{{ displayName }}</span>
        <span v-if="description" class="text-[11px] text-dimmed truncate flex-1">{{ description }}</span>
        <div class="flex items-center gap-1 shrink-0">
          <UBadge v-for="t in tags.slice(0, 3)" :key="t" size="xs" variant="subtle">{{ t }}</UBadge>
          <UBadge v-if="tags.length > 3" size="xs" variant="ghost">+{{ tags.length - 3 }}</UBadge>
        </div>
        <UBadge
          v-if="item.kind === 'subgraph'"
          size="xs"
          variant="subtle"
          color="neutral"
          class="shrink-0"
        >
          {{ item.graph?.nodes?.length ?? 0 }} 节点
        </UBadge>
      </div>

      <!-- grid mode -->
      <template v-else>
        <div class="flex items-center gap-2 mb-2">
          <UIcon
            :name="item.kind === 'subgraph' ? 'i-tabler-package' : 'i-tabler-photo'"
            class="size-4"
            :class="item.kind === 'subgraph' ? 'text-fuchsia-300' : 'text-violet-300'"
          />
          <span class="text-sm font-medium truncate flex-1">{{ displayName }}</span>
          <UBadge v-if="item.kind === 'subgraph'" size="xs" variant="subtle" color="neutral">
            {{ item.graph?.nodes?.length ?? 0 }}
          </UBadge>
        </div>
        <p v-if="description" class="text-[11px] text-dimmed mb-2 line-clamp-2">{{ description }}</p>
        <div v-if="tags.length > 0" class="flex flex-wrap gap-1">
          <UBadge v-for="t in tags.slice(0, 4)" :key="t" size="xs" variant="subtle">{{ t }}</UBadge>
          <UBadge v-if="tags.length > 4" size="xs" variant="ghost">+{{ tags.length - 4 }}</UBadge>
        </div>
      </template>
    </div>
  </UContextMenu>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { LibraryItem } from '@/lib/backend'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'
import { backend } from '@/lib/backend'
import { useLibraryStore } from '@/stores/library'

const props = withDefaults(
  defineProps<{
    item: LibraryItem
    selected?: boolean
    viewMode?: 'grid' | 'list'
  }>(),
  { selected: false, viewMode: 'grid' },
)

const emit = defineEmits<{
  select: [item: LibraryItem]
  open: [item: LibraryItem]
  'import-to-active': [item: LibraryItem]
}>()

const editorStore = useContainerEditorStore()
const libraryStore = useLibraryStore()
const toast = useToast()
const { confirm } = useConfirm()

const itemID = computed(() => (props.item.kind === 'subgraph' ? props.item.id : props.item.key))

const displayName = computed(() => {
  if (props.item.kind === 'subgraph') return props.item.label || props.item.id
  return props.item.meta?.name || props.item.key
})
const description = computed(() => {
  if (props.item.kind === 'subgraph') return props.item.description
  return props.item.meta?.description
})
const tags = computed<string[]>(() => {
  if (props.item.kind === 'subgraph') return props.item.tags ?? []
  return props.item.meta?.tags ?? []
})

const ctxMenuItems = computed(() => [
  [
    {
      label: '查看详情',
      icon: 'i-tabler-info-circle',
      onSelect: () => emit('select', props.item),
    },
    {
      label: '导入到当前容器',
      icon: 'i-tabler-arrow-bar-to-down',
      disabled: !editorStore.activeContainerID || props.item.kind !== 'subgraph',
      onSelect: () => emit('import-to-active', props.item),
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
    await navigator.clipboard.writeText(itemID.value)
    toast.add({ title: '已复制 ID', color: 'success', icon: 'i-tabler-check' })
  } catch (e: any) {
    toast.add({ title: '复制失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onDelete() {
  const yes = await confirm({
    title: props.item.kind === 'subgraph' ? '删除库子图' : '删除库模板',
    description: `确认删除 "${displayName.value}"？此操作不可恢复。`,
    color: 'error',
    confirmText: '删除',
  })
  if (yes !== true) return
  try {
    if (props.item.kind === 'subgraph') {
      await backend.library.deleteSubgraph(props.item.id)
    } else {
      await backend.library.deleteTemplateMany([props.item.key])
    }
    toast.add({ title: '已删除', color: 'success' })
    await libraryStore.reload()
  } catch (e: any) {
    toast.add({ title: '删除失败', description: String(e?.message ?? e), color: 'error' })
  }
}

function onDragStart(e: DragEvent) {
  if (!e.dataTransfer) return
  e.dataTransfer.setData(
    'application/yhfish-library-item',
    JSON.stringify({ kind: props.item.kind, id: itemID.value }),
  )
  e.dataTransfer.effectAllowed = 'copy'
}
</script>
