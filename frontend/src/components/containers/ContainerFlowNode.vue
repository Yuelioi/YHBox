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
    :style="{ minWidth: '240px', maxWidth: '360px' }"
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
        :title="t('editor.canvas.node_disabled_tooltip')"
      />
      <span
        v-if="isRunning"
        class="size-1.5 rounded-full bg-primary animate-pulse shrink-0"
      />
    </div>

    <!-- Body: grid 严格对齐. 同 row index 的 left/right 自动同 Y.
         left/right pin 数不等时按 max(len) 占行, 短的一列下方空 grid cell. -->
    <div class="node-body" :style="{ minHeight: bodyHeight + 'px' }">
      <template v-for="i in maxRows" :key="'row-' + i">
        <div class="pin-row pin-row-left">
          <template v-if="leftPins[i - 1]">
            <span
              class="pin-label truncate"
              :style="{ color: labelColor(leftPins[i - 1]) }"
            >{{ leftPins[i - 1].label }}</span>
            <PinLiteral
              v-if="showInlineLiteral(leftPins[i - 1])"
              class="pin-inline-input nodrag"
              :type="leftPins[i - 1].type"
              :widget-kind="fieldFor(leftPins[i - 1].id)?.widgetKind"
              :options="fieldFor(leftPins[i - 1].id)?.options"
              :model-value="inlineLiteralValue(leftPins[i - 1].id)"
              @update:model-value="(v: any) => onInlineLiteralUpdate(leftPins[i - 1].id, v)"
              @mousedown.stop
              @click.stop
            />
          </template>
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

    <!-- $变量引用 (表达式/脚本里写的 $hp) — v4 可见性补救: 不连线也一眼看出读了哪些变量 -->
    <div v-if="dollarRefs.length > 0" class="node-footer" :class="v.border">
      <div class="preview-row truncate">
        <span class="preview-key">vars</span>
        <span class="preview-val font-mono">{{ dollarRefs.map((r) => '$' + r).join('  ') }}</span>
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
      <span class="truncate font-mono">{{ props.data?.config?.SubgraphID || t('editor.canvas.subgraph_no_id') }}</span>
      <span
        v-if="boundSubgraphNodeCount !== null"
        class="ml-auto shrink-0 text-fuchsia-300/80 font-medium"
      >{{ t('containers.node_count', { n: boundSubgraphNodeCount }) }}</span>
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
      :class="[
        'handle-base',
        p.kind === 'exec' ? 'handle-exec' : 'handle-data',
        p.isError ? 'handle-exec-error' : '',
      ]"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
import { useI18n } from 'vue-i18n'
import { Handle, Position } from '@vue-flow/core'
import { useExecutionStore } from '@/stores/execution'
import { pinsFor, KIND_VISUAL, KIND_LABEL_ZH, PIN_SPECS, resolveSubgraphCallExecOut } from './pinSpec'
import PinLiteral from './inline/PinLiteral.vue'
import { unconnectedDataInPins, ContainerCanvasApiKey } from '@/composables/containerEditor/pinLiterals'
import { getSpec } from './nodeRegistry/registry'
import { TYPE_COLOR } from './nodeRegistry/index'
import type { PinType, FieldSchema } from './nodeRegistry/index'
import { useContainerEditorStore } from '@/stores/containerEditor'

const { t, te } = useI18n()
const execStore = useExecutionStore()
const editorStore = useContainerEditorStore()

// Pin label i18n lookup. key = node.<kind>.input/output.<pin>.label.
// 没注册 (e.g. dynamic Subgraph pin / fallback kind) → 返 raw pin name 字面值.
function pinLabel(pinName: string, dir: 'in' | 'out'): string {
  const k = `node.${kind.value}.${dir === 'in' ? 'input' : 'output'}.${pinName}.label`
  if (te(k)) return t(k)
  // 输出数据字段译名走共享字典 (跟 Inspector outLabel 同源, 画布也不显英文 pin 名)。
  if (dir === 'out') {
    const common = `inspector.output.field.${pinName}`
    if (te(common)) return t(common)
  }
  return pinName
}

const props = defineProps<{
  id: string
  data: { kind: string; config?: Record<string, any>; disabled?: boolean; label?: string }
  selected?: boolean
}>()

const kind = computed(() => props.data?.kind ?? '')
// KIND_LABEL_ZH[k] 是 i18n key 字符串, 经 t() 渲染; 未注册 kind 时 fallback 到 kind 字面.
function kindLabel(k: string): string {
  const key = KIND_LABEL_ZH[k]
  return key ? t(key) : k
}
// Subgraph 调用节点没填自定义 label 时, 主标题显示绑定子图的名字 (而非泛泛的 "调用子图") —
// 子图的语义就是它的身份. 查不到子图 / 无名则退回类型名.
const boundSubgraphLabel = computed<string | null>(() => {
  if (kind.value !== 'Subgraph') return null
  const sgID = props.data?.config?.SubgraphID
  if (!sgID) return null
  return editorStore.subgraphById(String(sgID))?.label || null
})
const displayLabel = computed(() => {
  if (props.data?.label) return props.data.label
  if (boundSubgraphLabel.value) return boundSubgraphLabel.value
  return kindLabel(kind.value)
})
// 副标题: 主标题显示的是自定义名 / 子图名 (≠ 默认类型名) 时, 用副标题补出类型名.
const kindSubtitle = computed(() =>
  displayLabel.value !== kindLabel(kind.value) ? kindLabel(kind.value) : null,
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

// 画布内联 pin literal — view 通过 ContainerCanvasApiKey provide; 测试/孤立渲染时为 null。
const canvasApi = inject(ContainerCanvasApiKey, null)

// 未连线 + scalar 的 data-in pin 名集合 → 这些 pin 行渲染内联 input。
// geometry(widget==='geometry') / point(widget==='point'): 走各自结构化内联 widget。
// 裸 point pin (无 point schema): 文本兜底会 String(obj)→"[object Object]", 排除。
// 排除 code(widgetKind==='code'): 多行脚本塞画布内联小框没法编, 走 Inspector 的 CodeInput/modal。
const inlineLiteralPins = computed<Set<string>>(() => {
  const edges = canvasApi?.edges.value ?? []
  const dataIn = PIN_SPECS[kind.value]?.dataIn ?? {}
  const ps = unconnectedDataInPins(kind.value, dataIn, props.data?.config ?? null, edges, props.id)
  return new Set(
    ps
      .filter((p) => {
        // code (多行脚本) 走 Inspector modal — 画布无法内联编辑.
        if (fieldFor(p.name)?.widgetKind === 'code') return false
        // geometry → 走 GeometryWidget; point 有 point schema → 走 PointWidget.
        // 两者都走结构化内联; 只有「裸 point pin (无 point schema)」才排除.
        if (p.type === 'point' && fieldFor(p.name)?.schema?.widget !== 'point') return false
        return true
      })
      .map((p) => p.name),
  )
})

function showInlineLiteral(p: PinEntry): boolean {
  return p.kind === 'data' && p.dir === 'in' && inlineLiteralPins.value.has(p.id)
}
// 查 pin 对应的 widget 元数据 (widgetKind/options) — 让画布内联框跟 Inspector 一样出下拉。
// 动态 input (Expr config.Inputs[]) 在 fields 里查不到 → undefined, PinLiteral 走 type fallback。
function fieldFor(pin: string): FieldSchema | undefined {
  return getSpec(kind.value)?.fields?.find((f) => f.key === pin)
}
function inlineLiteralValue(pin: string): unknown {
  // literal 优先 + 顶层 config fallback (镜像后端 PinValue / Inspector getLiteral) —
  // 让尚未迁移的旧数据 (值在顶层 config) 也能在画布内联框显示。
  const lit = props.data?.config?.literal as Record<string, unknown> | undefined
  if (lit && pin in lit) return lit[pin]
  return props.data?.config?.[pin]
}
function onInlineLiteralUpdate(pin: string, v: unknown) {
  canvasApi?.setPinLiteral(props.id, pin, v)
}

const boundSubgraphNodeCount = computed<number | null>(() => {
  if (kind.value !== 'Subgraph') return null
  const sgID = props.data?.config?.SubgraphID
  if (!sgID) return null
  const sg = editorStore.subgraphById(String(sgID))
  return sg?.graph?.nodes?.length ?? null
})

const execOutPinsForRender = computed<{ id: string; label: string; isError: boolean }[]>(() => {
  // 子图调用节点 (Subgraph / CollapsedNode) 的 exec-out = callee outputPins decl,
  // pin id 用 decl ID (runtime 按它路由), 显示名用 decl name.
  if (kind.value === 'Subgraph' || kind.value === 'CollapsedNode') {
    const decls = resolveSubgraphCallExecOut(
      { config: props.data?.config as any },
      editorStore.subgraphList,
    )
    const out = decls.map((d) => ({ id: d.id, label: d.name, isError: false }))
    // Spec 的静态 Fail 出口 (region 兜底) 不在子图 outputPins 里, 单独补.
    for (const id of getSpec(kind.value)?.errorOut ?? []) {
      out.push({ id, label: t('common.fail_pin'), isError: true })
    }
    return out
  }
  // Semantic==='error' 的失败出口 (Fail) → 红引脚.
  const errorOut = new Set(getSpec(kind.value)?.errorOut ?? [])
  return pins.value.execOut.map((id: string) => ({
    id,
    label: errorOut.has(id) ? t('common.fail_pin') : pinLabel(id, 'out'),
    isError: errorOut.has(id),
  }))
})

// $变量引用 — 从表达式/脚本源码正则提取 (展示用, 权威解析在后端 validator)。去重保序。
const dollarRefs = computed<string[]>(() => {
  if (!getSpec(kind.value)?.dynamicInputs) return []
  const lit = props.data?.config?.literal as Record<string, unknown> | undefined
  const src = `${lit?.Expression ?? ''}\n${lit?.Code ?? ''}`
  const seen = new Set<string>()
  for (const m of src.matchAll(/\$([A-Za-z_][A-Za-z0-9_]*)/g)) {
    seen.add(m[1])
  }
  return [...seen]
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
  /** 失败出口 (Semantic==='error') — exec 引脚渲染成红色. */
  isError?: boolean
}

const leftPins = computed<PinEntry[]>(() => [
  ...pins.value.execIn.map((p): PinEntry => ({ id: p, label: pinLabel(p, 'in'), kind: 'exec', type: 'any', dir: 'in' })),
  ...pins.value.dataIn.map(
    (p): PinEntry => ({ id: p, label: pinLabel(p, 'in'), kind: 'data', type: dataTypeMap.value.in[p] ?? 'any', dir: 'in' }),
  ),
])
const rightPins = computed<PinEntry[]>(() => [
  ...execOutPinsForRender.value.map(
    (p): PinEntry => ({ id: p.id, label: p.label, kind: 'exec', type: 'any', dir: 'out', isError: p.isError }),
  ),
  ...pins.value.dataOut.map(
    (p): PinEntry => ({ id: p, label: pinLabel(p, 'out'), kind: 'data', type: dataTypeMap.value.out[p] ?? 'any', dir: 'out' }),
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
      // 失败出口 (Fail / Semantic==='error') 用 error 红, 普通 exec 用浅灰.
      background: p.isError ? 'var(--ui-error)' : '#e5e7eb',
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
  // capture: 输出捕获绑定 (config.capture) 在 Inspector「输出」组展示, 不在节点体当原始 JSON 行渲染。
  const skip = new Set(['n', 'literal', 'capture'])
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
const ROW_H = 28

const maxRows = computed(() => Math.max(leftPins.value.length, rightPins.value.length))
const bodyHeight = computed(() => maxRows.value * ROW_H)
</script>

<style scoped>
.container-node {
  position: relative;
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
  /* selected 用 breathing 动画 (替代静态 ring). */
  animation: selected-breathe 2.6s ease-in-out infinite;
  z-index: 50;
  /* 阻断后面节点穿透 — backdrop-filter 模糊背后节点的内容, 即使 tailwind alpha tint 仍透,
     看到的也只是模糊色块 (字辨不出) 而非清晰文字. 保留 group 色 tint 身份感. */
  backdrop-filter: blur(16px) saturate(140%) brightness(0.85);
  -webkit-backdrop-filter: blur(16px) saturate(140%) brightness(0.85);
}
.container-node.is-disabled {
  opacity: 0.45;
  filter: grayscale(0.85) blur(0.2px);
}
.container-node.is-running {
  animation: pulse-running 1.6s ease-in-out infinite;
}
.container-node.is-running::after {
  /* 扫描线效果 — 顶部主色流光横扫 */
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 12px;
  background: linear-gradient(
    180deg,
    transparent 0%,
    color-mix(in oklab, var(--ui-primary) 18%, transparent) 50%,
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
  grid-auto-rows: 28px;
  padding: 6px 0 8px 0;
}
.pin-row {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  font-size: 12px;
  line-height: 1;
  user-select: none;
  pointer-events: none;
  min-width: 0; /* 防 truncate 失效 */
}

/* 内联 pin literal input — .pin-row 是 pointer-events:none, input 要单独 auto 才能交互。
   handle 在行左缘 (x=0), input 在 padding-left:16px + label 之后, 不压 handle 命中区。
   nodrag class + @mousedown.stop 防打字/点击触发节点拖动 / 画布平移 / 误拖连线。 */
.pin-inline-input {
  pointer-events: auto;
  flex: 1;
  /* min-width 必须够大才能"逼"长 label 截断让位 —— flex 只在空间不足时才压缩 label,
     min-width 太小(≈原剩余宽)等于没用. 88px 保证长 label 行也有可用编辑区, 上限仍 110px.
     配合 .pin-label 的 min-width:0 (overflow:hidden 已让其可缩, 这里显式标注意图). */
  min-width: 88px;
  max-width: 110px;
  margin-left: 4px;
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
  /* truncate (white-space:nowrap + ellipsis) 在 flex 里必须配 min-width:0 才真截断,
     否则长 label 撑满、把同行的内联输入框挤到几乎为 0 (见 .pin-inline-input). */
  min-width: 0;
  font-family:
    'JetBrains Mono', 'Cascadia Code', 'Consolas', ui-monospace, SFMono-Regular, Menlo,
    'Microsoft YaHei', 'PingFang SC', monospace;
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
  font-family: 'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, 'Microsoft YaHei', 'PingFang SC', monospace;
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
/* 失败出口 (Fail / Semantic==='error') 红引脚 — 红色辉光对齐普通 exec 的视觉权重.
   background 由 handleStyle inline 设 #f87171; 此处给红色 glow 增强. */
:deep(.vue-flow__handle.handle-exec-error) {
  filter: drop-shadow(0 0 2px rgba(248, 113, 113, 0.6));
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
      0 0 0 2px color-mix(in oklab, var(--ui-primary) 85%, transparent),
      0 0 20px color-mix(in oklab, var(--ui-primary) 60%, transparent),
      0 8px 22px -8px rgba(0, 0, 0, 0.55);
  }
  50% {
    box-shadow:
      0 0 0 3px var(--ui-primary),
      0 0 44px color-mix(in oklab, var(--ui-primary) 95%, transparent),
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
