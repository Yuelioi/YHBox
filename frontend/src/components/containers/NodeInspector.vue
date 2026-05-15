<template>
  <div v-if="!node" class="text-sm text-dimmed">未选中节点</div>

  <div v-else>
    <!-- Header: 大图标 + 中文名 + ID -->
    <header class="flex items-start gap-3 pb-4 mb-4 border-b border-default">
      <div
        class="size-10 rounded-lg flex items-center justify-center shrink-0"
        :class="[visual.bg, visual.border, 'border']"
      >
        <UIcon :name="visual.icon" class="size-5 text-default" />
      </div>
      <div class="min-w-0 flex-1">
        <h3 class="text-sm font-medium text-highlighted leading-tight">{{ label }}</h3>
        <p class="text-[11px] text-dimmed font-mono truncate mt-0.5">
          {{ node.kind }} · {{ node.id }}
        </p>
      </div>
      <UButton
        size="xs"
        variant="ghost"
        color="error"
        icon="i-tabler-trash"
        title="删除节点"
        @click="$emit('delete')"
      />
    </header>

    <!-- 用法说明 -->
    <section
      v-if="description"
      class="mb-5 rounded-md bg-elevated/30 border border-default/40 px-3 py-2.5"
    >
      <div class="flex items-start gap-2">
        <UIcon name="i-tabler-info-circle" class="size-3.5 text-primary shrink-0 mt-0.5" />
        <p class="text-[12px] text-toned leading-relaxed">{{ description }}</p>
      </div>
    </section>

    <!-- 并发警告 -->
    <section
      v-if="concurrencyWarning"
      class="mb-5 rounded-md bg-amber-500/10 border border-amber-500/30 px-3 py-2.5"
    >
      <div class="flex items-start gap-2">
        <UIcon name="i-tabler-alert-triangle" class="size-3.5 text-amber-300 shrink-0 mt-0.5" />
        <div class="text-[12px] text-amber-300">
          <div class="font-medium leading-tight">并发分支写入同一变量</div>
          <div class="text-amber-300/80 mt-1 leading-relaxed">{{ concurrencyWarning }}</div>
        </div>
      </div>
    </section>

    <!-- 屏幕选择工具：根据 kind 显示对应快捷 -->
    <section
      v-if="canPickPoint || canPickRect"
      class="mb-4 rounded-md bg-primary/5 border border-primary/30 p-3 space-y-2"
    >
      <div class="flex items-center gap-2">
        <UIcon name="i-tabler-crosshair" class="size-3.5 text-primary" />
        <span class="text-[11px] text-toned">屏幕拾取</span>
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <UButton
          v-if="canPickPoint"
          size="xs"
          variant="soft"
          color="primary"
          icon="i-tabler-pointer"
          :loading="picking"
          @click="onPickPoint"
        >
          截屏选点
        </UButton>
        <UButton
          v-if="canPickRect"
          size="xs"
          variant="soft"
          color="primary"
          icon="i-tabler-frame"
          :loading="picking"
          @click="onPickRect"
        >
          截屏框选 ROI
        </UButton>
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          icon="i-tabler-pointer"
          @click="onOpenHUD"
        >
          鼠标 HUD
        </UButton>
      </div>
      <p class="text-[10px] text-dimmed leading-snug">
        打开独立窗口截当前游戏画面，{{ canPickRect ? '拖矩形' : '点一下' }}后自动回填字段
      </p>
    </section>

    <!-- Config fields -->
    <section v-if="fields.length > 0">
      <h4 class="text-[10px] uppercase tracking-[0.08em] font-semibold text-dimmed mb-3">配置</h4>
      <div class="space-y-4">
        <div v-for="field in fields" :key="field.key" class="space-y-1.5">
          <label class="block text-xs text-toned">{{ field.label }}</label>
          <ExpressionInput
            v-if="field.type === 'expr'"
            :model-value="getCfg(field.key)"
            :placeholder="field.placeholder"
            :expected-type="field.exprType"
            :var-names="varNames"
            @update:model-value="setCfg(field.key, $event)"
          />
          <USelect
            v-else-if="field.type === 'select'"
            :model-value="getCfg(field.key)"
            :items="field.options ?? []"
            size="md"
            class="w-full"
            :ui="{ content: 'min-w-[280px]' }"
            @update:model-value="setCfg(field.key, String($event))"
          />
          <UInput
            v-else-if="field.type === 'text'"
            :model-value="getCfg(field.key)"
            size="md"
            :placeholder="field.placeholder"
            @update:model-value="setCfg(field.key, String($event))"
          />
          <ActionPicker
            v-else-if="field.type === 'action-picker'"
            :model-value="getCfg(field.key)"
            @update:model-value="setCfg(field.key, $event)"
          />
          <TemplatePicker
            v-else-if="field.type === 'template-picker'"
            :model-value="getCfg(field.key)"
            @update:model-value="setCfg(field.key, $event)"
          />
          <KeyCapture
            v-else-if="field.type === 'key-capture'"
            :model-value="getCfg(field.key)"
            @update:model-value="setCfg(field.key, $event)"
          />
          <p v-if="field.hint" class="text-[11px] text-dimmed leading-snug">{{ field.hint }}</p>
        </div>
      </div>
    </section>

    <p v-else class="text-[12px] text-dimmed">此节点无可配置项。</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { GraphNode } from '@/lib/backend'
import { backend } from '@/lib/backend'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import ExpressionInput from '@/components/expressions/ExpressionInput.vue'
import ActionPicker from './ActionPicker.vue'
import TemplatePicker from './TemplatePicker.vue'
import KeyCapture from './KeyCapture.vue'
import { KIND_LABEL_ZH, KIND_DESCRIPTION, KIND_VISUAL } from './pinSpec'

const props = defineProps<{
  node: GraphNode | null
  varNames?: string[]
  nodes?: GraphNode[]
  edges?: { from: string; to: string }[]
}>()
const emit = defineEmits<{ update: [config: Record<string, any>]; delete: [] }>()

interface Field {
  key: string
  label: string
  type: 'expr' | 'select' | 'text' | 'action-picker' | 'template-picker' | 'key-capture'
  options?: { label: string; value: string }[]
  placeholder?: string
  exprType?: 'number' | 'bool' | 'string' | 'point'
  hint?: string
}

const label = computed(() =>
  props.node ? (KIND_LABEL_ZH[props.node.kind] ?? props.node.kind) : '',
)
const description = computed(() => (props.node ? (KIND_DESCRIPTION[props.node.kind] ?? '') : ''))
const visual = computed(() =>
  props.node
    ? (KIND_VISUAL[props.node.kind] ?? {
        icon: 'i-tabler-circle',
        bg: 'bg-zinc-700/40',
        border: 'border-zinc-500/40',
      })
    : { icon: '', bg: '', border: '' },
)

const SCHEMAS: Record<string, Field[]> = {
  Sleep: [
    { key: 'durationMs', label: '休眠 ms', type: 'expr', placeholder: '1000', exprType: 'number' },
  ],
  Loop: [
    {
      key: 'mode',
      label: '模式',
      type: 'select',
      options: [
        { label: '固定次数 (count)', value: 'count' },
        { label: '条件循环 (while)', value: 'while' },
        { label: '无限循环 (forever)', value: 'forever' },
      ],
    },
    {
      key: 'count',
      label: '次数（count 模式）',
      type: 'expr',
      placeholder: '10',
      exprType: 'number',
    },
    {
      key: 'condition',
      label: '条件表达式（while 模式）',
      type: 'expr',
      placeholder: '$vars.x < 5',
      exprType: 'bool',
    },
  ],
  If: [{ key: 'condition', label: '条件表达式', type: 'expr', exprType: 'bool' }],
  Parallel: [
    { key: 'n', label: '分支数 (2-8)', type: 'expr', placeholder: '2', exprType: 'number' },
  ],
  Race: [{ key: 'n', label: '分支数 (2-8)', type: 'expr', placeholder: '2', exprType: 'number' }],
  Stop: [{ key: 'error', label: '错误信息（可选）', type: 'text' }],
  SetVar: [
    { key: 'varName', label: '变量名', type: 'text' },
    { key: 'value', label: '值（表达式）', type: 'expr' },
  ],
  IncVar: [
    { key: 'varName', label: '变量名', type: 'text' },
    { key: 'delta', label: '增量', type: 'expr', placeholder: '1', exprType: 'number' },
  ],
  WaitTemplate: [
    { key: 'template', label: '模板', type: 'template-picker' },
    { key: 'timeoutMs', label: '超时 ms', type: 'expr', placeholder: '5000', exprType: 'number' },
    {
      key: 'threshold',
      label: '匹配阈值',
      type: 'expr',
      placeholder: '0.85',
      exprType: 'number',
      hint: '0..1，越大越严格',
    },
  ],
  CheckTemplate: [
    { key: 'template', label: '模板', type: 'template-picker' },
    {
      key: 'threshold',
      label: '匹配阈值',
      type: 'expr',
      placeholder: '0.85',
      exprType: 'number',
      hint: '0..1，越大越严格',
    },
  ],
  ClickTemplate: [
    { key: 'template', label: '模板', type: 'template-picker' },
    { key: 'timeoutMs', label: '超时 ms', type: 'expr', placeholder: '5000', exprType: 'number' },
    { key: 'threshold', label: '匹配阈值', type: 'expr', placeholder: '0.85', exprType: 'number' },
    {
      key: 'button',
      label: '鼠标按键',
      type: 'select',
      options: [
        { label: '左键', value: 'left' },
        { label: '中键', value: 'middle' },
        { label: '右键', value: 'right' },
      ],
    },
  ],
  DetectColor: [
    {
      key: 'region',
      label: 'ROI (x,y,w,h 比例)',
      type: 'text',
      placeholder: '0.4,0.55,0.2,0.05',
      hint: '客户区比例 0..1，逗号分隔。留空 = 全屏',
    },
    {
      key: 'mode',
      label: '颜色模式',
      type: 'select',
      options: [
        { label: 'HSV (色相饱和明度)', value: 'hsv' },
        { label: 'RGB (红绿蓝)', value: 'rgb' },
      ],
    },
    {
      key: 'range',
      label: '颜色范围 (6 个值 CSV)',
      type: 'text',
      placeholder: 'HSV: 50,60,67,127,253,255',
      hint: 'HSV: hMin,hMax,sMin,sMax,vMin,vMax  (H: 0-360, S/V: 0-255)。RGB: rMin,rMax,gMin,gMax,bMin,bMax (0-255)',
    },
    {
      key: 'minPixels',
      label: '最少命中像素',
      type: 'expr',
      placeholder: '5',
      exprType: 'number',
      hint: '≥ 此值走 yes 出口；否则 no',
    },
  ],
  InvokeAction: [{ key: 'actionId', label: '动作', type: 'action-picker' }],
  ClickAt: [
    {
      key: 'xRatio',
      label: 'X 比例',
      type: 'expr',
      placeholder: '0.5',
      exprType: 'number',
      hint: '0..1，0=左 1=右；1280×720 屏上 0.5 = 中心 640px',
    },
    {
      key: 'yRatio',
      label: 'Y 比例',
      type: 'expr',
      placeholder: '0.5',
      exprType: 'number',
      hint: '0..1，0=顶 1=底',
    },
    {
      key: 'durationMs',
      label: '按下时长 ms',
      type: 'expr',
      placeholder: '50',
      exprType: 'number',
    },
    {
      key: 'button',
      label: '鼠标按键',
      type: 'select',
      options: [
        { label: '左键', value: 'left' },
        { label: '中键', value: 'middle' },
        { label: '右键', value: 'right' },
      ],
    },
  ],
  KeyPress: [
    { key: 'vk', label: '按键', type: 'key-capture', hint: '点输入框后按下任意键自动捕获' },
    {
      key: 'durationMs',
      label: '按下时长 ms',
      type: 'expr',
      placeholder: '50',
      exprType: 'number',
    },
  ],
  MouseMoveRel: [
    { key: 'dx', label: 'dx (像素)', type: 'expr', exprType: 'number' },
    { key: 'dy', label: 'dy (像素)', type: 'expr', exprType: 'number' },
    {
      key: 'durationMs',
      label: '移动用时 ms',
      type: 'expr',
      placeholder: '200',
      exprType: 'number',
    },
  ],
  Scroll: [
    { key: 'xRatio', label: 'X 比例', type: 'expr', exprType: 'number' },
    { key: 'yRatio', label: 'Y 比例', type: 'expr', exprType: 'number' },
    { key: 'delta', label: '滚动格数', type: 'expr', placeholder: '3', exprType: 'number' },
  ],
  OnEvent: [
    {
      key: 'kind',
      label: '事件类型',
      type: 'select',
      options: [{ label: '模板出现 (template_appeared)', value: 'template_appeared' }],
    },
    { key: 'template', label: '模板', type: 'template-picker' },
    {
      key: 'pollIntervalMs',
      label: '轮询间隔 ms',
      type: 'expr',
      placeholder: '100',
      exprType: 'number',
    },
    {
      key: 'maxConcurrent',
      label: '最大并发子图数',
      type: 'expr',
      placeholder: '1',
      exprType: 'number',
    },
    {
      key: 'retriggerPolicy',
      label: '重触发策略',
      type: 'select',
      options: [
        { label: '丢弃 (drop)', value: 'drop' },
        { label: '排队 (queue)', value: 'queue' },
        { label: '重启 (restart)', value: 'restart' },
      ],
    },
    { key: 'cooldownMs', label: '冷却 ms', type: 'expr', placeholder: '0', exprType: 'number' },
  ],
  Log: [
    {
      key: 'level',
      label: '级别',
      type: 'select',
      options: [
        { label: 'info', value: 'info' },
        { label: 'warn', value: 'warn' },
        { label: 'error', value: 'error' },
      ],
    },
    { key: 'message', label: '消息（表达式）', type: 'expr', exprType: 'string' },
  ],
  Toast: [
    { key: 'title', label: '标题（表达式）', type: 'expr', exprType: 'string' },
    { key: 'message', label: '内容（表达式）', type: 'expr', exprType: 'string' },
    {
      key: 'color',
      label: '颜色',
      type: 'select',
      options: [
        { label: '中性 (neutral)', value: 'neutral' },
        { label: '主色 (primary)', value: 'primary' },
        { label: '成功 (success)', value: 'success' },
        { label: '警告 (warning)', value: 'warning' },
        { label: '错误 (error)', value: 'error' },
      ],
    },
  ],
}

const fields = computed<Field[]>(() => (props.node ? (SCHEMAS[props.node.kind] ?? []) : []))

function getCfg(key: string): string {
  if (!props.node?.config) return ''
  const v = props.node.config[key]
  return v == null ? '' : String(v)
}

function setCfg(key: string, val: string) {
  if (!props.node) return
  const next = { ...props.node.config }
  if (val === '') delete next[key]
  else next[key] = val
  emit('update', next)
}

function setCfgBatch(patch: Record<string, string>) {
  if (!props.node) return
  const next = { ...props.node.config }
  for (const k in patch) {
    if (patch[k] === '') delete next[k]
    else next[k] = patch[k]
  }
  emit('update', next)
}

// ---- ScreenPicker 集成 ----
// 哪些节点能"截屏选点": ClickAt / Scroll（都是 xRatio/yRatio）
// 哪些节点能"截屏框选 ROI": DetectColor（region）
const canPickPoint = computed(() => {
  const k = props.node?.kind ?? ''
  return k === 'ClickAt' || k === 'Scroll'
})
const canPickRect = computed(() => {
  const k = props.node?.kind ?? ''
  return k === 'DetectColor'
})

const picking = ref(false)

function genID(): string {
  return 'pick-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now()
}

interface PickerResult {
  id: string
  mode: string
  payload: any
}

async function openPicker(mode: 'point' | 'rect', onResult: (payload: any) => void) {
  const id = genID()
  picking.value = true
  // 先挂监听，再开 picker 窗口——确保 picker emit 时本端已订阅
  const waiter = awaitWailsEvent<PickerResult>('tools:picker-result', (p) => p?.id === id)
  const r = await backend.tools.openScreenPicker(mode, id)
  if (r === undefined) {
    picking.value = false
    return
  }
  const result = await waiter
  picking.value = false
  if (!result.payload?.cancelled) onResult(result.payload)
}

async function onPickPoint() {
  await openPicker('point', (p) => {
    setCfgBatch({
      xRatio: p.xRatio.toFixed(4),
      yRatio: p.yRatio.toFixed(4),
    })
  })
}

async function onPickRect() {
  await openPicker('rect', (p) => {
    const r = p.region as [number, number, number, number]
    const csv = `${r[0].toFixed(3)},${r[1].toFixed(3)},${r[2].toFixed(3)},${r[3].toFixed(3)}`
    setCfg('region', csv)
  })
}

async function onOpenHUD() {
  await backend.tools.openMouseHUD()
}

// ---- 并发警告：Parallel/Race 子分支写同名 var ----
function reachable(
  startNodeID: string,
  startPin: string,
  allEdges: { from: string; to: string }[],
): string[] {
  const out = new Set<string>()
  const queue: string[] = []
  for (const e of allEdges) {
    if (e.from === `${startNodeID}.${startPin}`) {
      const to = e.to.split('.')[0]
      queue.push(to)
      out.add(to)
    }
  }
  while (queue.length > 0) {
    const cur = queue.shift()!
    for (const e of allEdges) {
      if (e.from.startsWith(cur + '.')) {
        const to = e.to.split('.')[0]
        if (!out.has(to)) {
          out.add(to)
          queue.push(to)
        }
      }
    }
  }
  return [...out]
}

const concurrencyWarning = computed<string>(() => {
  const n = props.node
  if (!n || (n.kind !== 'Parallel' && n.kind !== 'Race')) return ''
  const nodes = props.nodes ?? []
  const edges = props.edges ?? []
  const branchVars = new Map<number, Set<string>>()
  for (const e of edges) {
    if (!e.from.startsWith(n.id + '.branch')) continue
    const pin = e.from.slice(n.id.length + 1)
    const idx = Number(pin.replace('branch', ''))
    if (Number.isNaN(idx)) continue
    const reached = reachable(n.id, pin, edges)
    const vars = new Set<string>()
    for (const id of reached) {
      const node = nodes.find((x) => x.id === id)
      if (!node) continue
      if (node.kind === 'SetVar' || node.kind === 'IncVar') {
        const v = (node.config?.varName as string | undefined) ?? ''
        if (v) vars.add(v)
      }
    }
    branchVars.set(idx, vars)
  }
  const allBranchVars = [...branchVars.values()]
  const conflicts = new Set<string>()
  for (let i = 0; i < allBranchVars.length; i++) {
    for (let j = i + 1; j < allBranchVars.length; j++) {
      for (const v of allBranchVars[i]) {
        if (allBranchVars[j].has(v)) conflicts.add(v)
      }
    }
  }
  if (conflicts.size === 0) return ''
  return `分支同时写入: ${[...conflicts].join(', ')}（结果不确定）`
})
</script>
