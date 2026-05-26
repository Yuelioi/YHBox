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
    <!-- Selected 实色底: 阻断后面节点 alpha bg 穿透. 必须在 children 之前以便 paint order
         = parent bg → solid → header/body/footer. pointer-events:none 不挡 vue-flow 操作. -->
    <div v-if="selected" class="node-solid-bg" aria-hidden="true" />

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

    <!-- Body: grid 严格对齐. 同 row index 的 left/right 自动同 Y.
         left/right pin 数不等时按 max(len) 占行, 短的一列下方空 grid cell. -->
    <div class="node-body" :style="{ minHeight: bodyHeight + 'px' }">
      <template v-for="i in maxRows" :key="'row-' + i">
        <div class="pin-row pin-row-left">
          <span
            v-if="leftPins[i - 1]"
            class="pin-label truncate"
            :style="{ color: labelColor(leftPins[i - 1]) }"
          >{{ leftPins[i - 1].label }}</span>
        </div>
        <div class="pin-row pin-row-right">
          <span
            v-if="rightPins[i - 1]"
            class="pin-label truncate"
            :style="{ color: labelColor(rightPins[i - 1]) }"
          >{{ rightPins[i - 1].label }}</span>
        </div>
      </template>
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

const HEADER_H = 42
const BODY_PAD_TOP = 6
const ROW_H = 22

const maxRows = computed(() => Math.max(leftPins.value.length, rightPins.value.length))
const bodyHeight = computed(() => maxRows.value * ROW_H)
</script>

<style scoped>
.container-node {
  position: relative;
  /* isolation: isolate 创独立 stacking context, .node-solid-bg 用 z-index: -1 不会
     跑出节点边界 (escape 到隔壁节点下方). 没这个 z:-1 会冒到 vue-flow viewport 后. */
  isolation: isolate;
  border-radius: 12px;
  border-width: 1px;
  font-size: 12px;
  /* radial subtle gradient: 节点中心略亮向边缘衰减, 立体感 */
  background-image: radial-gradient(
    circle at 30% 0%,
    rgba(255, 255, 255, 0.04),
    transparent 60%
  );
  box-shadow:
    0 10px 30px -10px rgba(0, 0, 0, 0.7),
    0 2px 6px -1px rgba(0, 0, 0, 0.35),
    inset 0 1px 0 0 rgba(255, 255, 255, 0.06),
    inset 0 -1px 0 0 rgba(0, 0, 0, 0.3);
  transition:
    box-shadow 220ms ease,
    transform 220ms cubic-bezier(0.4, 0, 0.2, 1),
    border-color 180ms ease;
}
.container-node::before {
  /* 整边 1px 高光描边 — conic gradient 模拟金属边缘 */
  content: '';
  position: absolute;
  inset: -1px;
  border-radius: 13px;
  padding: 1px;
  background: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.18) 0%,
    rgba(255, 255, 255, 0.04) 35%,
    rgba(255, 255, 255, 0.02) 70%,
    rgba(255, 255, 255, 0.12) 100%
  );
  -webkit-mask:
    linear-gradient(#fff 0 0) content-box,
    linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
  pointer-events: none;
  z-index: 1;
}
/* hover lift 去掉 — 用户反馈卡. 静止视觉就够, 只 Handle hover 有反馈. */
.container-node.is-selected {
  /* selected 用 breathing 动画 (替代静态 ring) — keyframes 内含 ring + halo + outer glow,
     alpha/radius 变化产生呼吸感. */
  animation: selected-breathe 2.6s ease-in-out infinite;
  /* selected 节点提到最上层避免被旁边 unselected 节点遮挡 (vue-flow 默认 selected 已有, 这里
     兜底). */
  z-index: 50;
}
/* Selected 节点实色底 — 半透 tailwind bg-color (alpha 15%) 让后面堆叠节点的内容透过来
   (Pin label / config preview 互相穿插无法分辨). 加一层 var(--ui-bg-default) 实色 div +
   z-index: -1: CSS paint order 是 (parent bg → negative-z children → in-flow children),
   所以实色盖住 parent alpha tint, 同时 header/body/footer (static in-flow) 又盖在实色上.
   父加 isolation: isolate 防 z:-1 escape 到隔壁节点下方. */
.node-solid-bg {
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: var(--ui-bg-default);
  pointer-events: none;
  z-index: -1;
}
.container-node.is-disabled {
  opacity: 0.45;
  filter: grayscale(0.85) blur(0.2px);
}
.container-node.is-running {
  animation: pulse-running 1.6s ease-in-out infinite;
}
.container-node.is-running::after {
  /* 扫描线效果 — 顶部 emerald 流光横扫 */
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 12px;
  background: linear-gradient(
    180deg,
    transparent 0%,
    rgba(52, 211, 153, 0.18) 50%,
    transparent 100%
  );
  background-size: 100% 30%;
  background-repeat: no-repeat;
  animation: scan-line 1.8s linear infinite;
  pointer-events: none;
  z-index: 2;
  mix-blend-mode: screen;
}

.node-header {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 14px 9px 18px;
  height: 42px;
  border-top-left-radius: 11px;
  border-top-right-radius: 11px;
  overflow: hidden;
  /* header 加 linear gradient (135°) 层次感 */
  background-image: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.08) 0%,
    transparent 55%,
    rgba(0, 0, 0, 0.2) 100%
  );
  /* 分割: 底部 1px 高光 + 1px 暗影 (双线立体感) */
  box-shadow:
    inset 0 -1px 0 0 rgba(0, 0, 0, 0.4),
    0 1px 0 0 rgba(255, 255, 255, 0.06);
}
.header-accent {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  /* 圆角跟 header 顶左对齐, 避免直角矩形伸出 radius 外 */
  border-top-left-radius: 11px;
  /* 纯色 group accent, 不再 shimmer 流动 — 外围呼吸已经够 */
}
.header-icon {
  width: 18px;
  height: 18px;
  color: rgba(255, 255, 255, 0.95);
  filter: drop-shadow(0 0 4px currentColor);
}
.header-text {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 1px;
  line-height: 1.1;
}
.header-label {
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0.4px;
  color: rgba(255, 255, 255, 0.97);
  font-family:
    system-ui, -apple-system, 'Segoe UI Variable Display', 'SF Pro Display',
    'PingFang SC', 'Microsoft YaHei', sans-serif;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}
.header-sub {
  font-size: 10.5px;
  color: rgba(255, 255, 255, 0.55);
  font-weight: 500;
  letter-spacing: 0.3px;
  font-family:
    system-ui, -apple-system, 'Segoe UI Variable Text', 'SF Pro Text', 'PingFang SC',
    sans-serif;
}

/* Body: grid 严格对齐 — left/right col 同 row index 在同 Y, 不依赖 flex padding 微差.
   grid-auto-rows 22px 保证每行 height 一致, 子元素不允许撑高. */
.node-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-auto-rows: 22px;
  padding: 6px 0 8px 0;
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
  min-width: 0; /* 防 truncate 失效 */
}
.pin-row-left {
  padding-left: 16px;
  padding-right: 4px;
  justify-content: flex-start;
  grid-column: 1;
}
.pin-row-right {
  padding-left: 4px;
  padding-right: 16px;
  justify-content: flex-end;
  grid-column: 2;
}
.pin-label {
  font-family:
    'JetBrains Mono', 'Cascadia Code', 'Consolas', ui-monospace, SFMono-Regular, Menlo,
    monospace;
  letter-spacing: 0.3px;
  font-weight: 500;
  font-feature-settings: 'liga' 0, 'calt' 0;
}

.node-footer {
  padding: 7px 14px;
  background: rgba(0, 0, 0, 0.22);
  color: var(--ui-text-dimmed);
  font-size: 10.5px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  /* 跟 header 同款双线 inset 分割 (顶 1px 暗影 + 顶 1px 高光, 立体感) */
  box-shadow:
    inset 0 1px 0 0 rgba(255, 255, 255, 0.06),
    inset 0 2px 0 0 rgba(0, 0, 0, 0.35);
  /* 删 border-top, 完全用 inset shadow */
  border: none !important;
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

/* Handles — 默认静态. 选中节点后 data pin idle breathing + exec pin halo 强化. */
:deep(.vue-flow__handle.handle-base) {
  z-index: 5;
  transition:
    transform 180ms cubic-bezier(0.34, 1.56, 0.64, 1),
    box-shadow 180ms ease,
    filter 180ms ease;
}
:deep(.vue-flow__handle.handle-exec) {
  filter: drop-shadow(0 0 2px rgba(255, 255, 255, 0.25));
}
.is-selected :deep(.vue-flow__handle.handle-data) {
  animation: data-idle 3s ease-in-out infinite;
}
.is-selected :deep(.vue-flow__handle.handle-exec) {
  filter: drop-shadow(0 0 5px rgba(6, 182, 212, 0.7));
}
/* hover: 不用 transform scale (vue-flow 自己 translate(-50%, -50%) 居中, scale 后 %
   translate 是 scaled box 的 %, 会 1.6x 偏位). 改 box-shadow ring + filter glow 模拟
   放大反馈, geometry 不变 — Handle 位置永远跟 row label 对齐. */
:deep(.vue-flow__handle.handle-base:hover) {
  z-index: 10;
}
:deep(.vue-flow__handle.handle-exec:hover) {
  filter: drop-shadow(0 0 12px rgba(255, 255, 255, 1))
    drop-shadow(0 0 4px rgba(255, 255, 255, 0.6));
}
:deep(.vue-flow__handle.handle-data:hover) {
  box-shadow:
    0 0 0 4px rgba(255, 255, 255, 0.18),
    0 0 16px currentColor !important;
  animation: none;
}

@keyframes pulse-running {
  0%,
  100% {
    box-shadow:
      0 0 0 2px rgba(52, 211, 153, 0.85),
      0 0 20px rgba(52, 211, 153, 0.6),
      0 8px 22px -8px rgba(0, 0, 0, 0.55);
  }
  50% {
    box-shadow:
      0 0 0 3px rgba(52, 211, 153, 1),
      0 0 44px rgba(52, 211, 153, 0.95),
      0 8px 22px -8px rgba(0, 0, 0, 0.55);
  }
}
@keyframes selected-breathe {
  0%,
  100% {
    box-shadow:
      0 0 0 1.5px #06b6d4,
      0 0 0 4px rgba(6, 182, 212, 0.14),
      0 0 18px rgba(6, 182, 212, 0.28),
      0 12px 30px -10px rgba(0, 0, 0, 0.6);
  }
  50% {
    box-shadow:
      0 0 0 1.5px #06b6d4,
      0 0 0 8px rgba(6, 182, 212, 0.28),
      0 0 38px rgba(6, 182, 212, 0.6),
      0 12px 30px -10px rgba(0, 0, 0, 0.6);
  }
}
@keyframes scan-line {
  0% {
    background-position: 0% 0%;
  }
  100% {
    background-position: 0% 100%;
  }
}
@keyframes data-idle {
  0%,
  100% {
    opacity: 0.85;
  }
  50% {
    opacity: 1;
  }
}
</style>
