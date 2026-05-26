<template>
  <div
    class="container-node"
    :class="[
      v.bg,
      v.border,
      selected ? 'is-selected' : '',
      isRunning ? 'is-running' : '',
      isDisabled ? 'is-disabled' : '',
    ]"
    :style="{ minWidth: '220px', maxWidth: '360px' }"
  >
    <!-- Header (浓 saturation, accent bar 在左侧) -->
    <div class="node-header" :class="v.headerBg">
      <span class="header-accent" :class="v.accent" />
      <UIcon :name="v.icon" class="header-icon shrink-0" />
      <div class="header-text min-w-0 flex-1">
        <div class="header-label truncate">{{ displayLabel }}</div>
        <div v-if="kindSubtitle" class="header-sub truncate">{{ kindSubtitle }}</div>
      </div>
      <UIcon
        v-if="isDisabled"
        name="i-tabler-ban"
        class="size-3.5 text-warning shrink-0"
        title="此节点已禁用 (运行时跳过)"
      />
      <span
        v-if="isRunning"
        class="size-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0"
      />
    </div>

    <!-- Body: 左右两列 pin, UE Blueprint 风格. -->
    <div class="node-body" :style="{ minHeight: bodyHeight + 'px' }">
      <div class="pin-col pin-col-left">
        <div v-for="p in leftPins" :key="'lp-' + p.id" class="pin-row pin-row-left">
          <span class="pin-label truncate" :style="{ color: labelColor(p) }">{{ p.label }}</span>
        </div>
      </div>
      <div class="pin-col pin-col-right">
        <div v-for="p in rightPins" :key="'rp-' + p.id" class="pin-row pin-row-right">
          <span class="pin-label truncate" :style="{ color: labelColor(p) }">{{ p.label }}</span>
        </div>
      </div>
    </div>

    <!-- Config preview -->
    <div v-if="preview.length > 0" class="node-footer" :class="v.border">
      <div v-for="p in preview" :key="p.k" class="preview-row truncate">
        <span class="preview-key">{{ p.k }}</span>
        <span class="preview-val">{{ p.v }}</span>
      </div>
    </div>

    <!-- Subgraph 子图 ID + 节点数 -->
    <div v-if="kind === 'Subgraph'" class="node-footer subgraph-footer" :class="v.border">
      <UIcon name="i-tabler-arrow-narrow-right" class="size-3 shrink-0 text-dimmed" />
      <span class="truncate font-mono">{{ props.data?.config?.SubgraphID || '(未选)' }}</span>
      <span
        v-if="boundSubgraphNodeCount !== null"
        class="ml-auto shrink-0 text-fuchsia-300/80 font-medium"
      >{{ boundSubgraphNodeCount }} 节点</span>
    </div>

    <!-- Handles: exec 三角 + data 圆按 type 上色 -->
    <Handle
      v-for="(p, i) in leftPins"
      :key="'h-l-' + p.id"
      :id="p.id"
      type="target"
      :position="Position.Left"
      :style="handleStyle(p, i)"
      :class="['handle-base', p.kind === 'exec' ? 'handle-exec' : 'handle-data']"
    />
    <Handle
      v-for="(p, i) in rightPins"
      :key="'h-r-' + p.id"
      :id="p.id"
      type="source"
      :position="Position.Right"
      :style="handleStyle(p, i)"
      :class="['handle-base', p.kind === 'exec' ? 'handle-exec' : 'handle-data']"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { useExecutionStore } from '@/stores/execution'
import { pinsFor, KIND_VISUAL, KIND_LABEL_ZH, resolveSubgraphCallExecOut } from './pinSpec'
import { getSpec } from './nodeRegistry/registry'
import { TYPE_COLOR } from './nodeRegistry/index'
import type { PinType } from './nodeRegistry/index'
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

// 节点 visual: header 浓一档 (-500/35), accent bar 用纯 -500.
const v = computed(() => {
  const base = KIND_VISUAL[kind.value] ?? {
    icon: 'i-tabler-circle',
    bg: 'bg-muted',
    border: 'border-default',
  }
  const m = /^bg-([a-z]+)-\d+\/\d+$/.exec(base.bg)
  const color = m ? m[1] : 'zinc'
  return {
    ...base,
    headerBg: `bg-${color}-500/30`,
    accent: `bg-${color}-400`,
  }
})

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

const dataTypeMap = computed<{ in: Record<string, PinType>; out: Record<string, PinType> }>(() => {
  const spec = getSpec(kind.value)
  if (!spec) return { in: {}, out: {} }
  const dyn = spec.dataInDynamicFn ? spec.dataInDynamicFn(props.data?.config ?? null) : {}
  return {
    in: { ...spec.dataIn, ...dyn },
    out: spec.dataOut,
  }
})

interface PinEntry {
  id: string
  label: string
  kind: 'exec' | 'data'
  type: PinType
  dir: 'in' | 'out'
}

const leftPins = computed<PinEntry[]>(() => [
  ...pins.value.execIn.map((p): PinEntry => ({ id: p, label: p, kind: 'exec', type: 'any', dir: 'in' })),
  ...pins.value.dataIn.map(
    (p): PinEntry => ({ id: p, label: p, kind: 'data', type: dataTypeMap.value.in[p] ?? 'any', dir: 'in' }),
  ),
])
const rightPins = computed<PinEntry[]>(() => [
  ...execOutPinsForRender.value.map(
    (p): PinEntry => ({ id: p.id, label: p.label, kind: 'exec', type: 'any', dir: 'out' }),
  ),
  ...pins.value.dataOut.map(
    (p): PinEntry => ({ id: p, label: p, kind: 'data', type: dataTypeMap.value.out[p] ?? 'any', dir: 'out' }),
  ),
])

// label 颜色: exec 用默认色 (CSS 控), data 用 type 颜色但 alpha 调暗一档.
function labelColor(p: PinEntry): string {
  if (p.kind === 'exec') return ''
  return TYPE_COLOR[p.type] ?? '#9ca3af'
}

// Handle 样式: exec 三角形 (clip-path), data 圆形 + type 颜色.
function handleStyle(p: PinEntry, i: number): Record<string, string> {
  const top = HEADER_H + BODY_PAD_TOP + i * ROW_H + ROW_H / 2 + 'px'
  if (p.kind === 'exec') {
    return {
      top,
      background: '#e5e7eb',
      clipPath: 'polygon(0% 0%, 100% 50%, 0% 100%)',
      borderRadius: '0',
      border: 'none',
      width: '11px',
      height: '11px',
    }
  }
  const color = TYPE_COLOR[p.type] ?? '#9ca3af'
  return {
    top,
    background: color,
    border: '2px solid rgba(0,0,0,0.4)',
    width: '12px',
    height: '12px',
    boxShadow: `0 0 0 1px ${color}33`,
  }
}

// preview: stringify 友好化 (object → JSON, 防 [object Object])
const preview = computed(() => {
  const cfg = props.data?.config ?? {}
  const skip = new Set(['n', 'literal'])
  const out: { k: string; v: string }[] = []
  for (const k of Object.keys(cfg)) {
    if (skip.has(k)) continue
    const raw = cfg[k]
    if (raw === '' || raw == null) continue
    let s: string
    if (typeof raw === 'object') {
      try {
        s = JSON.stringify(raw)
      } catch {
        s = '[object]'
      }
    } else {
      s = String(raw)
    }
    out.push({ k, v: s.length > 32 ? s.slice(0, 32) + '…' : s })
    if (out.length >= 3) break
  }
  return out
})

const HEADER_H = 38
const BODY_PAD_TOP = 6
const ROW_H = 22

const bodyHeight = computed(() => Math.max(leftPins.value.length, rightPins.value.length) * ROW_H)
</script>

<style scoped>
.container-node {
  position: relative;
  border-radius: 10px;
  border-width: 1px;
  font-size: 12px;
  box-shadow:
    0 8px 24px -8px rgba(0, 0, 0, 0.55),
    0 2px 4px -1px rgba(0, 0, 0, 0.3),
    inset 0 1px 0 0 rgba(255, 255, 255, 0.04);
  transition: box-shadow 180ms ease, transform 180ms ease;
}
.container-node:hover {
  box-shadow:
    0 12px 32px -8px rgba(0, 0, 0, 0.6),
    0 3px 6px -1px rgba(0, 0, 0, 0.35),
    inset 0 1px 0 0 rgba(255, 255, 255, 0.06);
}
.container-node.is-selected {
  box-shadow:
    0 0 0 2px var(--ui-primary, #6366f1),
    0 0 0 5px rgba(99, 102, 241, 0.22),
    0 10px 28px -8px rgba(0, 0, 0, 0.55);
}
.container-node.is-disabled {
  opacity: 0.5;
  filter: grayscale(0.8);
}
.container-node.is-running {
  animation: pulse-running 1.5s ease-in-out infinite;
}

.node-header {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px 8px 14px;
  height: 38px;
  border-top-left-radius: 9px;
  border-top-right-radius: 9px;
  overflow: hidden;
}
.header-accent {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
}
.header-icon {
  width: 18px;
  height: 18px;
  color: rgba(255, 255, 255, 0.92);
}
.header-text {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 1px;
  line-height: 1.1;
}
.header-label {
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.1px;
  color: rgba(255, 255, 255, 0.95);
  font-family:
    'Inter', system-ui, -apple-system, 'PingFang SC', 'Microsoft YaHei', sans-serif;
}
.header-sub {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.45);
  font-weight: 400;
}

.node-body {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 6px 0 8px 0;
}
.pin-col {
  display: flex;
  flex-direction: column;
  flex: 1 1 50%;
  min-width: 0;
}
.pin-col-left {
  padding-left: 14px;
}
.pin-col-right {
  padding-right: 14px;
}
.pin-row {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 22px;
  font-size: 11px;
  line-height: 1;
  user-select: none;
  pointer-events: none;
}
.pin-row-right {
  justify-content: flex-end;
}
.pin-label {
  font-family: 'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.2px;
  font-weight: 500;
}

.node-footer {
  padding: 6px 12px;
  border-top-width: 1px;
  border-style: solid;
  background: rgba(0, 0, 0, 0.15);
  color: var(--ui-text-dimmed);
  font-size: 10.5px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.preview-row {
  display: flex;
  gap: 6px;
  font-family: 'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace;
}
.preview-key {
  color: var(--ui-text-toned);
  font-weight: 500;
}
.preview-key::after {
  content: ':';
}
.preview-val {
  color: var(--ui-text-dimmed);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}
.subgraph-footer {
  flex-direction: row;
  align-items: center;
  gap: 6px;
}

/* Handles */
:deep(.vue-flow__handle.handle-base) {
  z-index: 5;
  transition: transform 120ms ease, box-shadow 120ms ease, filter 120ms ease;
}
:deep(.vue-flow__handle.handle-base:hover) {
  transform: translateY(-50%) scale(1.5);
  filter: brightness(1.2);
}
:deep(.vue-flow__handle.handle-exec:hover) {
  filter: drop-shadow(0 0 6px rgba(255, 255, 255, 0.7));
}
:deep(.vue-flow__handle.handle-data:hover) {
  box-shadow: 0 0 8px currentColor !important;
}

@keyframes pulse-running {
  0%,
  100% {
    box-shadow:
      0 0 0 2px rgba(52, 211, 153, 0.85),
      0 0 18px rgba(52, 211, 153, 0.55),
      0 8px 22px -8px rgba(0, 0, 0, 0.55);
  }
  50% {
    box-shadow:
      0 0 0 3px rgba(52, 211, 153, 1),
      0 0 38px rgba(52, 211, 153, 0.85),
      0 8px 22px -8px rgba(0, 0, 0, 0.55);
  }
}
</style>
