<!--
LibraryClipCard 录制片段卡片（库视图 clips tab 用）。

支持 view mode (grid / list) + selection 状态。
卡片本体只读 + UContextMenu 右键菜单；详细编辑（改名 / desc / tags）走右侧 LibraryDetailPanel。
保留 hover 状态下的删除按钮快捷入口；保留拖拽 dataTransfer（虽然 clip 当前没消费者，但语义留着）。
-->
<template>
  <UContextMenu :items="ctxMenuItems">
    <div
      :class="[
        'rounded-md border bg-elevated/30 transition-colors relative group cursor-pointer',
        viewMode === 'list' ? 'p-2 flex items-center gap-3' : 'p-3',
        selected ? 'border-primary ring-2 ring-primary/40' : 'border-default hover:border-primary/50',
      ]"
      @click="$emit('select', clip)"
    >
      <!-- list mode 左侧 icon 圆 -->
      <div
        v-if="viewMode === 'list'"
        class="size-7 rounded-md flex items-center justify-center shrink-0 bg-emerald-500/15 border border-emerald-500/40"
      >
        <UIcon name="i-tabler-vinyl" class="size-4 text-emerald-300" />
      </div>

      <template v-if="viewMode === 'list'">
        <div class="min-w-0 flex-1 flex items-center gap-2">
          <span class="text-sm font-medium truncate text-default">
            {{ clip.label || clip.id }}
          </span>
          <span v-if="clip.description" class="text-[11px] text-dimmed truncate flex-1">
            {{ clip.description }}
          </span>
          <div class="flex items-center gap-1 shrink-0">
            <UBadge v-for="t in tags.slice(0, 3)" :key="t" size="xs" variant="subtle">{{ t }}</UBadge>
            <UBadge v-if="tags.length > 3" size="xs" variant="ghost">+{{ tags.length - 3 }}</UBadge>
          </div>
          <div class="flex items-center gap-1 text-[11px] text-dimmed shrink-0">
            <UIcon name="i-tabler-clock" class="size-3" />
            <span>{{ durationLabel }}</span>
            <span class="opacity-50">·</span>
            <span>{{ clip.eventCount }} ev</span>
          </div>
        </div>
      </template>

      <!-- grid mode -->
      <template v-else>
        <div class="flex items-center gap-2 mb-2">
          <UIcon name="i-tabler-vinyl" class="size-4 text-emerald-300" />
          <span class="text-sm font-medium truncate flex-1">
            {{ clip.label || clip.id }}
          </span>
        </div>

        <div class="flex items-center gap-2 text-[11px] text-dimmed mb-2">
          <UIcon name="i-tabler-clock" class="size-3" />
          <span>{{ durationLabel }}</span>
          <span class="opacity-50">·</span>
          <UIcon name="i-tabler-calendar" class="size-3" />
          <span>{{ createdLabel }}</span>
          <span class="opacity-50">·</span>
          <span>{{ clip.eventCount }} ev</span>
        </div>

        <p v-if="clip.description" class="text-[11px] text-dimmed mb-2 line-clamp-2">
          {{ clip.description }}
        </p>

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
import { useClipsStore, type ClipSummary } from '@/stores/clips'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@nuxt/ui/composables'
import { fmtMillis } from '@/lib/format'

const props = withDefaults(
  defineProps<{
    clip: ClipSummary
    selected?: boolean
    viewMode?: 'grid' | 'list'
  }>(),
  { selected: false, viewMode: 'grid' },
)

const emit = defineEmits<{
  select: [clip: ClipSummary]
}>()

const clipsStore = useClipsStore()
const { confirm } = useConfirm()
const toast = useToast()

const tags = computed<string[]>(() => props.clip.tags ?? [])

const durationLabel = computed(() => fmtMillis(Math.floor((props.clip.durationUs ?? 0) / 1000)))

const createdLabel = computed(() => {
  const iso = props.clip.createdAt
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d.getTime())) return '—'
  const now = new Date()
  const sameDay =
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate()
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  if (sameDay) return `${hh}:${mm}`
  const M = String(d.getMonth() + 1).padStart(2, '0')
  const D = String(d.getDate()).padStart(2, '0')
  return `${M}-${D} ${hh}:${mm}`
})

const ctxMenuItems = computed(() => [
  [
    {
      label: '查看详情',
      icon: 'i-tabler-info-circle',
      onSelect: () => emit('select', props.clip),
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
    await navigator.clipboard.writeText(props.clip.id)
    toast.add({ title: '已复制 ID', color: 'success', icon: 'i-tabler-check' })
  } catch (e: any) {
    toast.add({ title: '复制失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onDelete() {
  const yes = await confirm({
    title: '删除录制片段',
    description: `确认删除片段 "${props.clip.label || props.clip.id}"？此操作不可恢复。`,
    color: 'error',
    confirmText: '删除',
  })
  if (yes !== true) return
  try {
    await clipsStore.remove(props.clip.id)
    toast.add({ title: '已删除', color: 'success' })
  } catch (e: any) {
    toast.add({ title: '删除失败', description: String(e?.message ?? e), color: 'error' })
  }
}
</script>
