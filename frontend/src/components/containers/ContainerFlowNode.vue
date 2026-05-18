<template>
  <div
    class="container-node rounded-md border text-xs shadow-sm transition-all"
    :class="[
      v.bg,
      v.border,
      selected ? 'ring-2 ring-primary' : '',
      isRunning ? 'ring-2 ring-emerald-400 shadow-emerald-500/30 animate-pulse-running' : '',
    ]"
    :style="{ minWidth: '180px' }"
  >
    <!-- Header -->
    <div class="flex items-center gap-1.5 px-2.5 py-1 border-b" :class="v.border">
      <UIcon :name="v.icon" class="size-3.5 text-default shrink-0" />
      <span class="font-medium text-default truncate">{{ label }}</span>
      <span
        v-if="isRunning"
        class="ml-auto size-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0"
      />
    </div>

    <!-- Pin rows: 每行 ROW_H 高，左右 label 各 8px 内边距给 handle 让位 -->
    <div class="pin-grid">
      <div v-for="i in pinRows" :key="'row-' + i" class="pin-row">
        <span class="pin-label-left">{{ pins.execIn[i] ?? '' }}</span>
        <span class="pin-label-right">{{ pins.execOut[i] ?? '' }}</span>
      </div>
    </div>

    <!-- Config preview (前 3 个 string config) -->
    <div
      v-if="preview.length > 0"
      class="px-2.5 py-1 border-t space-y-0.5 text-[10px] text-dimmed"
      :class="v.border"
    >
      <div v-for="p in preview" :key="p.k" class="truncate font-mono">
        <span class="text-toned">{{ p.k }}</span
        >: {{ p.v }}
      </div>
    </div>

    <!-- Data pins row -->
    <div
      v-if="pins.dataIn.length > 0 || pins.dataOut.length > 0"
      class="px-2.5 py-1 flex items-center gap-2 border-t"
      :class="v.border"
    >
      <span
        v-for="pin in pins.dataIn"
        :key="'din-l-' + pin"
        class="text-[9px] text-blue-300 font-mono"
        >↓{{ pin }}</span
      >
      <span class="grow" />
      <span
        v-for="pin in pins.dataOut"
        :key="'dout-l-' + pin"
        class="text-[9px] text-blue-300 font-mono"
        >{{ pin }}↓</span
      >
    </div>

    <!-- Subgraph 选中的子图 ID 预览 + 内部节点数 (子图复杂度一眼可见) -->
    <div v-if="kind === 'Subgraph'" class="mt-1 text-[10px] text-dimmed font-mono truncate px-2.5 pb-1 flex items-center gap-1.5">
      <span class="truncate">→ {{ props.data?.config?.subgraphId || '(未选)' }}</span>
      <span
        v-if="boundSubgraphNodeCount !== null"
        class="ml-auto shrink-0 text-fuchsia-300/80 not-italic"
      >{{ boundSubgraphNodeCount }} 节点</span>
    </div>

    <!-- Handles (绝对定位到 pin row 的 y 坐标) -->
    <Handle
      v-for="(pin, i) in pins.execIn"
      :key="'ein-' + pin"
      :id="pin"
      type="target"
      :position="Position.Left"
      :style="{ top: execInTop(i) + 'px' }"
      class="w-2.5! h-2.5! bg-elevated! border! border-accented!"
    />

    <Handle
      v-for="(pin, i) in execOutPinsForRender"
      :key="'eout-' + pin.id"
      :id="pin.id"
      type="source"
      :position="Position.Right"
      :style="{ top: execOutTop(i) + 'px' }"
      class="w-2.5! h-2.5! bg-elevated! border! border-accented!"
    >
      <span v-if="kind === 'Subgraph'" class="text-[9px] text-toned ml-1">{{ pin.label }}</span>
    </Handle>

    <Handle
      v-for="(pin, i) in pins.dataIn"
      :key="'din-' + pin"
      :id="pin"
      type="target"
      :position="Position.Bottom"
      :style="{ left: dataInLeft(i) + 'px' }"
      class="w-2! h-2! bg-blue-400! border-0!"
    />

    <Handle
      v-for="(pin, i) in pins.dataOut"
      :key="'dout-' + pin"
      :id="pin"
      type="source"
      :position="Position.Bottom"
      :style="{ right: dataOutRight(i) + 'px', left: 'auto' }"
      class="w-2! h-2! bg-blue-400! border-0!"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { useExecutionStore } from '@/stores/execution'
import { pinsFor, KIND_VISUAL, KIND_LABEL_ZH, resolveSubgraphCallExecOut } from './pinSpec'
import { useContainerEditorStore } from '@/stores/containerEditor'

const execStore = useExecutionStore()

const props = defineProps<{
  id: string
  data: { kind: string; config?: Record<string, any> }
  selected?: boolean
}>()

const kind = computed(() => props.data?.kind ?? '')
const label = computed(() => KIND_LABEL_ZH[kind.value] ?? kind.value)
const isRunning = computed(() => execStore.running && execStore.currentNodeID === props.id)
const v = computed(
  () =>
    KIND_VISUAL[kind.value] ?? {
      icon: 'i-tabler-circle',
      bg: 'bg-muted',
      border: 'border-default',
    },
)

const pins = computed(() => pinsFor(kind.value, props.data?.config ?? null))

const editorStore = useContainerEditorStore()

// Subgraph 调用节点: 绑定子图的内部节点数 (画布上一眼看到子图复杂度)
const boundSubgraphNodeCount = computed<number | null>(() => {
  if (kind.value !== 'Subgraph') return null
  const sgID = props.data?.config?.subgraphId
  if (!sgID) return null
  const sg = editorStore.subgraphsForCurrentContainer.find((s) => s.id === sgID)
  return sg?.graph?.nodes?.length ?? null
})

// exec-out pins 渲染数据（特判 Subgraph 节点）
// 返回 { id: string, label: string }[] —— id 是 vue-flow handle id，label 是显示文本
const execOutPinsForRender = computed(() => {
  if (kind.value === 'Subgraph') {
    const decls = resolveSubgraphCallExecOut(
      { config: props.data?.config as any },
      editorStore.subgraphsForCurrentContainer,
    )
    return decls.map((d) => ({ id: d.id, label: d.name }))
  }
  return pins.value.execOut.map((id: string) => ({ id, label: id }))
})

const preview = computed(() => {
  const cfg = props.data?.config ?? {}
  const skip = new Set(['n']) // n 已通过分支数体现
  const out: { k: string; v: string }[] = []
  for (const k of Object.keys(cfg)) {
    if (skip.has(k)) continue
    const raw = cfg[k]
    if (raw === '' || raw == null) continue
    out.push({ k, v: String(raw).slice(0, 26) })
    if (out.length >= 3) break
  }
  return out
})

// header 26px + 每 pin 行 18px。pin 行起点 = header_h
const HEADER_H = 26
const ROW_H = 18

const pinRows = computed(() => {
  const n = Math.max(pins.value.execIn.length, pins.value.execOut.length)
  return Array.from({ length: n }, (_, i) => i)
})

function execInTop(i: number) {
  return HEADER_H + i * ROW_H + ROW_H / 2 - 5
}
function execOutTop(i: number) {
  return HEADER_H + i * ROW_H + ROW_H / 2 - 5
}
function dataInLeft(i: number) {
  return 16 + i * 22
}
function dataOutRight(i: number) {
  return 16 + i * 22
}
</script>

<style scoped>
.container-node {
  position: relative;
}

.pin-grid {
  display: flex;
  flex-direction: column;
}
.pin-row {
  display: flex;
  align-items: center;
  height: 18px;
  padding: 0 12px;
  font-size: 10px;
  line-height: 1;
  color: var(--ui-text-dimmed);
  user-select: none;
  pointer-events: none;
}
.pin-label-left {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.pin-label-right {
  margin-left: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

:deep(.vue-flow__handle) {
  z-index: 5;
}

.animate-pulse-running {
  animation: pulse-running 1.5s ease-in-out infinite;
}
@keyframes pulse-running {
  0%,
  100% {
    box-shadow:
      0 0 0 2px rgba(52, 211, 153, 0.6),
      0 0 12px rgba(52, 211, 153, 0.4);
  }
  50% {
    box-shadow:
      0 0 0 2px rgba(52, 211, 153, 0.9),
      0 0 24px rgba(52, 211, 153, 0.7);
  }
}
</style>
