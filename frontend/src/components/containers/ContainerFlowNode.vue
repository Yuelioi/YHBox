<template>
  <div
    class="container-node rounded-md border text-xs shadow-sm transition-all"
    :class="[
      v.bg,
      v.border,
      selected ? 'ring-2 ring-primary' : '',
      isRunning ? 'ring-2 ring-emerald-400 shadow-emerald-500/30 animate-pulse-running' : '',
      isDisabled ? 'opacity-50 grayscale' : '',
    ]"
    :style="{ minWidth: '220px' }"
  >
    <!-- Header -->
    <div class="flex items-center gap-1.5 px-2.5 py-1 border-b" :class="v.border">
      <UIcon :name="v.icon" class="size-3.5 text-default shrink-0" />
      <span class="font-medium text-default truncate">{{ displayLabel }}</span>
      <span v-if="kindSubtitle" class="text-[9px] text-dimmed truncate shrink-0">({{ kindSubtitle }})</span>
      <UIcon
        v-if="isDisabled"
        name="i-tabler-ban"
        class="size-3 text-warning shrink-0"
        title="此节点已禁用 (运行时跳过)"
      />
      <span
        v-if="isRunning"
        class="ml-auto size-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0"
      />
    </div>

    <!-- Body: UE Blueprint 风格. 左列 input (exec+data), 右列 output (exec+data).
         每行 ROW_H 高, 节点 height 由 max(leftPins.length, rightPins.length) 决定. -->
    <div class="pin-body" :style="{ minHeight: bodyHeight + 'px' }">
      <div class="pin-col pin-col-left">
        <div v-for="p in leftPins" :key="'lp-' + p.id" class="pin-row pin-row-left">
          <span v-if="p.kind === 'exec'" class="pin-arrow text-default">▶</span>
          <span v-else class="pin-dot bg-blue-400" />
          <span
            class="pin-label truncate"
            :class="p.kind === 'exec' ? 'text-default' : 'text-blue-200'"
          >{{ p.label }}</span>
        </div>
      </div>
      <div class="pin-col pin-col-right">
        <div v-for="p in rightPins" :key="'rp-' + p.id" class="pin-row pin-row-right">
          <span
            class="pin-label truncate"
            :class="p.kind === 'exec' ? 'text-default' : 'text-blue-200'"
          >{{ p.label }}</span>
          <span v-if="p.kind === 'exec'" class="pin-arrow text-default">▶</span>
          <span v-else class="pin-dot bg-blue-400" />
        </div>
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

    <!-- Subgraph 选中的子图 ID 预览 + 内部节点数 -->
    <div v-if="kind === 'Subgraph'" class="mt-1 text-[10px] text-dimmed font-mono truncate px-2.5 pb-1 flex items-center gap-1.5">
      <span class="truncate">→ {{ props.data?.config?.SubgraphID || '(未选)' }}</span>
      <span
        v-if="boundSubgraphNodeCount !== null"
        class="ml-auto shrink-0 text-fuchsia-300/80 not-italic"
      >{{ boundSubgraphNodeCount }} 节点</span>
    </div>

    <!-- Handles 绝对定位到对应 pin row 的 y 坐标. left col = Position.Left, right col = Position.Right -->
    <Handle
      v-for="(p, i) in leftPins"
      :key="'h-l-' + p.id"
      :id="p.id"
      type="target"
      :position="Position.Left"
      :style="{ top: pinTop(i) + 'px' }"
      :class="
        p.kind === 'exec'
          ? 'w-2.5! h-2.5! bg-elevated! border! border-accented!'
          : 'w-2! h-2! bg-blue-400! border-0!'
      "
    />
    <Handle
      v-for="(p, i) in rightPins"
      :key="'h-r-' + p.id"
      :id="p.id"
      type="source"
      :position="Position.Right"
      :style="{ top: pinTop(i) + 'px' }"
      :class="
        p.kind === 'exec'
          ? 'w-2.5! h-2.5! bg-elevated! border! border-accented!'
          : 'w-2! h-2! bg-blue-400! border-0!'
      "
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
  data: { kind: string; config?: Record<string, any>; disabled?: boolean; label?: string }
  selected?: boolean
}>()

const kind = computed(() => props.data?.kind ?? '')
const displayLabel = computed(() =>
  props.data?.label ? props.data.label : (KIND_LABEL_ZH[kind.value] ?? kind.value),
)
const kindSubtitle = computed(() =>
  props.data?.label ? (KIND_LABEL_ZH[kind.value] ?? kind.value) : null,
)
const isRunning = computed(() => execStore.running && execStore.currentNodeID === props.id)
const isDisabled = computed(() => props.data?.disabled === true)
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

const boundSubgraphNodeCount = computed<number | null>(() => {
  if (kind.value !== 'Subgraph') return null
  const sgID = props.data?.config?.SubgraphID
  if (!sgID) return null
  const sg = editorStore.subgraphsForCurrentContainer.find((s) => s.id === sgID)
  return sg?.graph?.nodes?.length ?? null
})

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

// UE 风格 pin column: 左 = execIn 接 dataIn, 右 = execOut 接 dataOut. 每行一个 pin.
interface PinEntry {
  id: string
  label: string
  kind: 'exec' | 'data'
}

const leftPins = computed<PinEntry[]>(() => [
  ...pins.value.execIn.map((p): PinEntry => ({ id: p, label: p, kind: 'exec' })),
  ...pins.value.dataIn.map((p): PinEntry => ({ id: p, label: p, kind: 'data' })),
])
const rightPins = computed<PinEntry[]>(() => [
  ...execOutPinsForRender.value.map((p): PinEntry => ({ id: p.id, label: p.label, kind: 'exec' })),
  ...pins.value.dataOut.map((p): PinEntry => ({ id: p, label: p, kind: 'data' })),
])

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

// header 26px + 每 pin 行 ROW_H. 节点 height 由 max(leftPins, rightPins) 决定.
const HEADER_H = 26
const ROW_H = 18

const bodyHeight = computed(() => Math.max(leftPins.value.length, rightPins.value.length) * ROW_H)

function pinTop(i: number) {
  return HEADER_H + i * ROW_H + ROW_H / 2 - 5
}
</script>

<style scoped>
.container-node {
  position: relative;
}

.pin-body {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}
.pin-col {
  display: flex;
  flex-direction: column;
  flex: 1 1 50%;
  min-width: 0;
}
.pin-col-left {
  padding-left: 12px;
}
.pin-col-right {
  padding-right: 12px;
}
.pin-row {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 18px;
  font-size: 10px;
  line-height: 1;
  color: var(--ui-text-dimmed);
  user-select: none;
  pointer-events: none;
}
.pin-row-right {
  justify-content: flex-end;
}
.pin-label {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.pin-arrow {
  font-size: 8px;
  line-height: 1;
}
.pin-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 9999px;
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
