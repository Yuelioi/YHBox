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
    :style="{ minWidth: '200px' }"
  >
    <!-- Header (浓 saturation 区分 body) -->
    <div class="node-header" :class="v.headerBg">
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

    <!-- Body: 左右两列 pin, UE Blueprint 风格. Handle 绝对定位到 row y. -->
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

    <!-- Config preview (前 3 个 string config) -->
    <div v-if="preview.length > 0" class="node-preview" :class="v.border">
      <div v-for="p in preview" :key="p.k" class="truncate font-mono">
        <span class="text-toned">{{ p.k }}</span
        >: {{ p.v }}
      </div>
    </div>

    <!-- Subgraph 子图 ID + 内部节点数 -->
    <div v-if="kind === 'Subgraph'" class="node-subgraph" :class="v.border">
      <span class="truncate">→ {{ props.data?.config?.SubgraphID || '(未选)' }}</span>
      <span
        v-if="boundSubgraphNodeCount !== null"
        class="ml-auto shrink-0 text-fuchsia-300/80"
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

// 节点 visual + header 浓色一档 (group 颜色 saturation 强化区分 body).
// KIND_VISUAL bg 是 -500/15 透明, header 用 -500/30 同色系.
const v = computed(() => {
  const base = KIND_VISUAL[kind.value] ?? {
    icon: 'i-tabler-circle',
    bg: 'bg-muted',
    border: 'border-default',
  }
  // 从 bg-XXX-500/15 抽出颜色名拼 -500/30 当 header bg.
  const m = /^bg-([a-z]+)-\d+\/\d+$/.exec(base.bg)
  const headerBg = m ? `bg-${m[1]}-500/30` : 'bg-elevated/50'
  return { ...base, headerBg }
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

// type lookup for data pins: 含 static spec.dataIn/Out + dynamic (Expr.Inputs/Subgraph 等).
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

// label 颜色: exec 用 default 灰白, data 用 type 颜色 (浅一档好读).
function labelColor(p: PinEntry): string {
  if (p.kind === 'exec') return ''
  return TYPE_COLOR[p.type] ?? '#9ca3af'
}

// Handle 样式: exec 三角形 (clip-path), data 圆形 + type 颜色.
function handleStyle(p: PinEntry, i: number): Record<string, string> {
  const top = HEADER_H + i * ROW_H + ROW_H / 2 - 5 + 'px'
  if (p.kind === 'exec') {
    return {
      top,
      background: 'var(--ui-text-default, #f5f5f5)',
      clipPath: 'polygon(0% 0%, 100% 50%, 0% 100%)',
      borderRadius: '0',
      border: 'none',
      width: '10px',
      height: '10px',
    }
  }
  return {
    top,
    background: TYPE_COLOR[p.type] ?? '#9ca3af',
    border: 'none',
    width: '9px',
    height: '9px',
  }
}

const preview = computed(() => {
  const cfg = props.data?.config ?? {}
  const skip = new Set(['n'])
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

const HEADER_H = 30
const ROW_H = 20

const bodyHeight = computed(() => Math.max(leftPins.value.length, rightPins.value.length) * ROW_H)
</script>

<style scoped>
.container-node {
  position: relative;
  border-radius: 8px;
  border-width: 1px;
  font-size: 12px;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.35), 0 0 0 1px rgba(255, 255, 255, 0.02);
  transition: box-shadow 180ms ease, transform 180ms ease;
}
.container-node.is-selected {
  box-shadow: 0 0 0 2px var(--ui-primary, #6366f1), 0 6px 18px rgba(0, 0, 0, 0.4);
}
.container-node.is-disabled {
  opacity: 0.5;
  filter: grayscale(0.8);
}
.container-node.is-running {
  box-shadow: 0 0 0 2px rgba(52, 211, 153, 0.8), 0 0 28px rgba(52, 211, 153, 0.6);
  animation: pulse-running 1.5s ease-in-out infinite;
}

.node-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  height: 30px;
  border-top-left-radius: 7px;
  border-top-right-radius: 7px;
}

.node-body {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 4px 0;
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
  height: 20px;
  font-size: 10.5px;
  line-height: 1;
  user-select: none;
  pointer-events: none;
}
.pin-row-right {
  justify-content: flex-end;
}
.pin-label {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.2px;
}

.node-preview {
  padding: 5px 10px;
  border-top-width: 1px;
  color: var(--ui-text-dimmed);
  font-size: 10px;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.node-subgraph {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border-top-width: 1px;
  color: var(--ui-text-dimmed);
  font-size: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

/* Handle base — vue-flow 默认带一些 style 用 :deep + ! 覆盖. */
:deep(.vue-flow__handle.handle-base) {
  z-index: 5;
  transition: transform 120ms ease, box-shadow 120ms ease;
}
:deep(.vue-flow__handle.handle-base:hover) {
  transform: translateY(-50%) scale(1.4);
  box-shadow: 0 0 6px currentColor;
}

@keyframes pulse-running {
  0%,
  100% {
    box-shadow: 0 0 0 2px rgba(52, 211, 153, 0.7), 0 0 16px rgba(52, 211, 153, 0.45);
  }
  50% {
    box-shadow: 0 0 0 2px rgba(52, 211, 153, 1), 0 0 32px rgba(52, 211, 153, 0.75);
  }
}
</style>
