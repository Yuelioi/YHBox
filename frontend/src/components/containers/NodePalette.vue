<template>
  <div class="flex flex-col h-full text-xs">
    <UTabs v-model="activeTab" :items="paletteTabs" size="xs" :ui="{ list: 'h-8 shrink-0' }" />

    <div class="flex-1 overflow-y-auto pt-2 space-y-2">
      <!-- 节点 tab -->
      <template v-if="activeTab === 'nodes'">
        <div v-for="group in groups" :key="group.label">
          <div class="font-medium text-toned mb-1">{{ group.label }}</div>
          <div class="space-y-0.5">
            <button
              v-for="n in group.items"
              :key="n.kind"
              type="button"
              class="w-full text-left px-2 py-1 rounded hover:bg-elevated/40 text-default transition-colors cursor-grab active:cursor-grabbing"
              draggable="true"
              @dragstart="onNodeDragStart($event, n.kind)"
              @click="$emit('add', n.kind)"
            >
              <UIcon :name="n.icon" class="size-3.5 mr-1.5 inline text-dimmed" />
              {{ n.label }}
            </button>
          </div>
        </div>
      </template>

      <!-- 库 tab：库子图 mini 列表 -->
      <template v-else-if="activeTab === 'library'">
        <div class="px-1 pb-1">
          <UInput
            v-model="libSearch"
            placeholder="搜索库子图..."
            icon="i-tabler-search"
            size="xs"
          />
        </div>

        <p v-if="libraryStore.subgraphs.length === 0" class="text-[11px] text-dimmed text-center py-4">
          库还是空的<br />
          <a class="text-primary cursor-pointer hover:underline" @click="goLibrary">去库管理</a>
        </p>
        <p
          v-else-if="filteredLibrary.length === 0"
          class="text-[11px] text-dimmed text-center py-4"
        >
          没有匹配的子图
        </p>

        <div v-for="sg in filteredLibrary" :key="sg.id" class="px-0.5">
          <UContextMenu :items="ctxMenuItemsFor(sg)">
            <div
              draggable="true"
              class="rounded-md border border-default/60 bg-elevated/30 p-1.5 cursor-grab hover:border-primary/40 transition-colors"
              :title="tooltipFor(sg)"
              @dragstart="onLibraryDragStart($event, sg)"
              @dblclick="goLibrary"
            >
              <div class="flex items-center gap-1.5">
                <UIcon name="i-tabler-package" class="size-3 text-fuchsia-300 shrink-0" />
                <span class="text-[11px] truncate flex-1">{{ sg.label || sg.id }}</span>
                <UBadge size="xs" variant="subtle" color="neutral" class="shrink-0">
                  {{ sg.graph?.nodes?.length ?? 0 }}
                </UBadge>
              </div>
              <div v-if="(sg.tags ?? []).length > 0" class="flex flex-wrap gap-0.5 mt-1">
                <UBadge
                  v-for="t in (sg.tags ?? []).slice(0, 3)"
                  :key="t"
                  size="xs"
                  variant="subtle"
                  class="text-[9px] py-0 px-1"
                >
                  {{ t }}
                </UBadge>
                <UBadge
                  v-if="(sg.tags ?? []).length > 3"
                  size="xs"
                  variant="ghost"
                  class="text-[9px] py-0 px-1"
                >
                  +{{ (sg.tags ?? []).length - 3 }}
                </UBadge>
              </div>
            </div>
          </UContextMenu>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useLibraryStore } from '@/stores/library'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useToast } from '@nuxt/ui/composables'
import { backend, type LibrarySubgraph } from '@/lib/backend'
import { KIND_LABEL_ZH } from './pinSpec'

defineEmits<{ add: [kind: string] }>()

const libraryStore = useLibraryStore()
const editorStore = useContainerEditorStore()
const router = useRouter()
const toast = useToast()

const paletteTabs = [
  { label: '节点', value: 'nodes' },
  { label: '库', value: 'library' },
]
const activeTab = ref('nodes')
const libSearch = ref('')

const filteredLibrary = computed(() => {
  const q = libSearch.value.trim().toLowerCase()
  let list = libraryStore.subgraphs.slice()
  if (q) {
    list = list.filter(
      (sg) =>
        (sg.label ?? '').toLowerCase().includes(q) ||
        (sg.id ?? '').toLowerCase().includes(q) ||
        (sg.description ?? '').toLowerCase().includes(q) ||
        (sg.tags ?? []).some((t) => t.toLowerCase().includes(q)),
    )
  }
  return list
})

function tooltipFor(sg: LibrarySubgraph): string {
  const parts = [sg.label || sg.id]
  if (sg.description) parts.push(sg.description)
  if ((sg.tags ?? []).length > 0) parts.push('#' + (sg.tags ?? []).join(' #'))
  return parts.join('\n')
}

function ctxMenuItemsFor(sg: LibrarySubgraph) {
  return [
    [
      {
        label: '导入到当前容器',
        icon: 'i-tabler-arrow-bar-to-down',
        disabled: !editorStore.activeContainerID,
        onSelect: () => onImport(sg),
      },
    ],
    [
      {
        label: '在库管理中查看',
        icon: 'i-tabler-external-link',
        onSelect: () => goLibrary(),
      },
      {
        label: '复制 ID',
        icon: 'i-tabler-copy',
        onSelect: () => onCopyID(sg.id),
      },
    ],
  ]
}

async function onImport(sg: LibrarySubgraph) {
  if (!editorStore.activeContainerID) {
    toast.add({ title: '当前没有打开的容器编辑器', color: 'warning' })
    return
  }
  try {
    const r = (await backend.library.copyToContainer(sg.id, editorStore.activeContainerID)) as any
    toast.add({
      title: `已导入子图: ${r?.label ?? r?.id ?? ''}`,
      color: 'success',
      icon: 'i-tabler-check',
    })
  } catch (e: any) {
    toast.add({ title: '导入失败', description: String(e?.message ?? e), color: 'error' })
  }
}

async function onCopyID(id: string) {
  try {
    await navigator.clipboard.writeText(id)
    toast.add({ title: '已复制 ID', color: 'success', icon: 'i-tabler-check' })
  } catch (e: any) {
    toast.add({ title: '复制失败', description: String(e?.message ?? e), color: 'error' })
  }
}

function onNodeDragStart(e: DragEvent, kind: string) {
  if (!e.dataTransfer) return
  e.dataTransfer.effectAllowed = 'copy'
  e.dataTransfer.setData('application/x-yhbox-node', kind)
}

function onLibraryDragStart(e: DragEvent, sg: LibrarySubgraph) {
  if (!e.dataTransfer) return
  e.dataTransfer.effectAllowed = 'copy'
  e.dataTransfer.setData(
    'application/yhfish-library-item',
    JSON.stringify({ kind: 'subgraph', id: sg.id }),
  )
}

function goLibrary() {
  void router.push('/library')
}

onMounted(() => {
  void libraryStore.reload()
})

const KINDS_BY_GROUP: {
  label: string
  items: { kind: string; icon: string }[]
}[] = [
  {
    label: '控制流',
    items: [
      { kind: 'Start', icon: 'i-tabler-player-play' },
      { kind: 'Sleep', icon: 'i-tabler-clock' },
      { kind: 'Loop', icon: 'i-tabler-repeat' },
      { kind: 'If', icon: 'i-tabler-git-branch' },
      { kind: 'Switch', icon: 'i-tabler-switch-3' },
      { kind: 'Parallel', icon: 'i-tabler-columns-3' },
      { kind: 'Race', icon: 'i-tabler-flag' },
      { kind: 'Stop', icon: 'i-tabler-square' },
      { kind: 'Break', icon: 'i-tabler-player-skip-forward' },
      { kind: 'Continue', icon: 'i-tabler-corner-down-left' },
    ],
  },
  {
    label: '变量',
    items: [
      { kind: 'SetVar', icon: 'i-tabler-equal' },
      { kind: 'IncVar', icon: 'i-tabler-circle-plus' },
    ],
  },
  {
    label: '图像',
    items: [
      { kind: 'WaitTemplate', icon: 'i-tabler-eye' },
      { kind: 'CheckTemplate', icon: 'i-tabler-search' },
      { kind: 'ClickTemplate', icon: 'i-tabler-target' },
      { kind: 'DetectColor', icon: 'i-tabler-color-picker' },
    ],
  },
  {
    label: '输入',
    items: [
      { kind: 'ClickAt', icon: 'i-tabler-click' },
      { kind: 'KeyPress', icon: 'i-tabler-keyboard' },
      { kind: 'MouseMoveRel', icon: 'i-tabler-arrows-move' },
      { kind: 'Scroll', icon: 'i-tabler-mouse' },
    ],
  },
  {
    label: '录制',
    items: [
      { kind: 'PlayClip', icon: 'i-tabler-vinyl' },
    ],
  },
  { label: '事件', items: [{ kind: 'OnEvent', icon: 'i-tabler-radio' }] },
  {
    label: '调试',
    items: [
      { kind: 'Log', icon: 'i-tabler-file-text' },
      { kind: 'Toast', icon: 'i-tabler-bell' },
    ],
  },
  {
    label: '子图',
    items: [
      { kind: 'Subgraph', icon: 'i-tabler-package' },
      { kind: 'SubgraphInput', icon: 'i-tabler-arrow-bar-to-right' },
      { kind: 'SubgraphOutput', icon: 'i-tabler-arrow-bar-to-left' },
    ],
  },
  {
    label: '系统',
    items: [{ kind: 'BringGameForeground', icon: 'i-tabler-app-window' }],
  },
  {
    label: '配置',
    items: [
      { kind: 'WindowTarget', icon: 'i-tabler-app-window' },
      { kind: 'MouseCalibration', icon: 'i-tabler-target' },
    ],
  },
  {
    label: '检测 v3',
    items: [
      { kind: 'DetectColorHSV', icon: 'i-tabler-palette' },
      { kind: 'ROIColorScan', icon: 'i-tabler-scan-eye' },
      { kind: 'Screenshot', icon: 'i-tabler-camera' },
    ],
  },
  {
    label: '输入长按 v3',
    items: [
      { kind: 'KeyHoldStart', icon: 'i-tabler-keyboard' },
      { kind: 'KeyHoldStop', icon: 'i-tabler-keyboard-off' },
      { kind: 'MouseHoldStart', icon: 'i-tabler-hand-click' },
      { kind: 'MouseHoldStop', icon: 'i-tabler-hand-off' },
    ],
  },
  {
    label: '时序 v3',
    items: [
      { kind: 'StopwatchStart', icon: 'i-tabler-player-play' },
      { kind: 'StopwatchStop', icon: 'i-tabler-player-stop' },
      { kind: 'StopwatchRead', icon: 'i-tabler-stopwatch' },
    ],
  },
  {
    label: '错误处理 v3',
    items: [
      { kind: 'Try', icon: 'i-tabler-shield-exclamation' },
      { kind: 'Throw', icon: 'i-tabler-bolt' },
    ],
  },
]

const groups = KINDS_BY_GROUP.map((g) => ({
  label: g.label,
  items: g.items.map((n) => ({ ...n, label: KIND_LABEL_ZH[n.kind] ?? n.kind })),
}))
</script>
