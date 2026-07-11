<template>
  <!-- Inspector 端 widget-aware pin 编辑器 — 按 backend widget kind 选控件, 写回 config.literal[pin]。
       画布端的 scalar 内联编辑仍用 PinLiteral.vue (本组件不上画布)。 -->
  <!-- checkbox / bool -->
  <UCheckbox
    v-if="kind === 'checkbox' || (kind === '' && type === 'bool')"
    :model-value="!!modelValue"
    @update:model-value="(v: boolean) => commit(v)"
  />

  <!-- number / slider / duration — duration 值就是毫秒数 (number), 必须走数字框存 number;
       漏了它会掉到末尾文本框 → 存成字符串 → 保存被 LITERAL_TYPE_MISMATCH 拦 (Sleep.Duration 即此坑) -->
  <UInputNumber
    v-else-if="kind === 'number' || kind === 'slider' || kind === 'duration' || (kind === '' && type === 'number')"
    :model-value="numModel"
    :min="min"
    :max="max"
    :step="step"
    :format-options="{ useGrouping: false }"
    size="sm"
    class="w-full"
    @update:model-value="(v: number) => commit(v)"
  />

  <!-- static dropdown (枚举, options 内联) -->
  <USelect
    v-else-if="kind === 'dropdown' && (options?.length ?? 0) > 0"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :items="selectItems"
    size="sm"
    class="w-full"
    @update:model-value="(v: any) => commit(v)"
  />

  <!-- async dropdown (运行时数据源, e.g. ADB devices): 候选懒加载, 同时允许手输兜底。 -->
  <div v-else-if="kind === 'async-dropdown'" class="space-y-1">
    <UInputMenu
      :model-value="modelValue == null ? '' : String(modelValue)"
      :items="asyncSelectItems"
      :create-item="'always'"
      value-key="value"
      size="sm"
      class="w-full"
      :loading="asyncLoading"
      :placeholder="placeholder"
      @update:model-value="onAsyncValue"
      @create="(v: string) => commit(v)"
    />
    <p v-if="asyncError" class="text-[10px] text-error">{{ asyncError }}</p>
  </div>

  <!-- ai-connection: AI 节点连接选择器, 从 settings 读连接列表 (label→id); 空 = 用默认连接 -->
  <USelect
    v-else-if="kind === 'ai-connection'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :items="connectionItems"
    size="sm"
    class="w-full"
    @update:model-value="(v: any) => commit(v)"
  />

  <!-- JSON / rect — textarea + parse, 保留 invalid raw text, 仅 valid 时 commit -->
  <div v-else-if="kind === 'json' || kind === 'rect-editor'" class="space-y-1">
    <UTextarea
      :model-value="rawText"
      :rows="3"
      size="sm"
      class="w-full font-mono text-[11px]"
      :color="jsonValid ? undefined : 'error'"
      :placeholder="placeholder"
      @update:model-value="onJsonInput"
    />
    <p v-if="!jsonValid" class="text-[10px] text-error">{{ t('inspector.pin_input_json_invalid') }}</p>
  </div>

  <!-- expr (Expr Expression) — 函数/$变量补全 + 即时语法红错 + 放大编辑 modal -->
  <ExprInput
    v-else-if="kind === 'expr'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :placeholder="placeholder"
    :input-names="inputNames"
    :declared-vars="declaredVars"
    @update:model-value="(v: string) => commit(v)"
  />

  <!-- code (Script.Code) — JS 编辑器: 节点函数/糖函数/动态输入/变量补全 + 放大编辑 modal -->
  <CodeInput
    v-else-if="kind === 'code'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :placeholder="placeholder"
    :input-names="inputNames"
    :declared-vars="declaredVars"
    @update:model-value="(v: string) => commit(v)"
    @declare-var="(a) => emit('declare-var', a)"
  />

  <!-- textarea -->
  <UTextarea
    v-else-if="kind === 'textarea'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :rows="3"
    size="sm"
    class="w-full font-mono"
    :placeholder="placeholder"
    @update:model-value="(v: string) => commit(v)"
  />

  <!-- key-capture: 聚焦后按物理键自动填 vk (KeyPress/KeyHold* 的 VK 字段) -->
  <KeyCapture
    v-else-if="kind === 'key-capture'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    @update:model-value="(v: string) => commit(v)"
  />

  <!-- color-preset: 视觉注册中心色块选择器 (CommentBox.Color 等), 存 palette key -->
  <ColorPalettePicker
    v-else-if="kind === 'color-preset'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    @update:model-value="(v: string) => commit(v)"
  />

  <!-- icon-preset: 视觉注册中心图标选择器 (CommentBox.Icon 等), 存完整 tabler 名 -->
  <IconPicker
    v-else-if="kind === 'icon-preset'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    @update:model-value="(v: string) => commit(v)"
  />

  <!-- list pin — wire-only, 不渲染可编辑 input 防手输垃圾 literal -->
  <span
    v-else-if="type === 'list'"
    class="text-xs text-dimmed italic"
  >{{ t('containers.listPinWireOnly') }}</span>

  <!-- text / password / 默认 → 文本框 -->
  <UInput
    v-else
    :model-value="modelValue == null ? '' : String(modelValue)"
    size="sm"
    class="w-full"
    :type="kind === 'password' ? 'password' : 'text'"
    :placeholder="placeholder"
    @update:model-value="(v: any) => commit(v)"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import KeyCapture from '@/components/containers/KeyCapture.vue'
import ColorPalettePicker from './ColorPalettePicker.vue'
import IconPicker from './IconPicker.vue'
import ExprInput from '@/components/expressions/ExprInput.vue'
import CodeInput from '@/components/expressions/CodeInput.vue'
import { coerceLiteral } from './coerceLiteral'
import { useSettingsStore } from '@/stores/settings'
import { NodeService } from '@bindings/github.com/yottaapp/yotta/internal/node'
import type { PinType } from '../pinSpec'
import type { VarType } from '@/lib/variableRef'
import {
  asyncOptionPayloadForValue,
  normalizeAsyncDropdownValue,
  type AsyncOption,
} from './asyncDropdown'

const { t } = useI18n()
const settingsStore = useSettingsStore()

// AI 节点连接下拉项: 第一项「用默认」(空值) + settings 里各连接 (label→id)。
const connectionItems = computed(() => [
  { value: '', label: t('node.AI.input.Connection.useDefault') },
  ...(settingsStore.data?.ai?.connections ?? []).map((c) => ({ value: c.id, label: c.label })),
])

const props = defineProps<{
  /** PinType (number/bool/string/point/any/list) — widgetKind 缺失时的 fallback 渲染依据。 */
  type: PinType
  /** 原始 backend widget kind (见 FieldSchema.widgetKind)。空串 = 无 spec 字段 (Expr 动态输入), 走 type fallback。 */
  widgetKind?: string
  modelValue: any
  options?: Array<{ value: string; labelKey: string }>
  placeholder?: string
  /** number/slider 步进与范围 (后端 SliderProps). 不传则 UInputNumber 默认 step=1 (整数). */
  min?: number
  max?: number
  step?: number
  /** async-dropdown 数据源名，对应 backend node.RegisterAsyncSource。 */
  asyncSource?: string
  /** 当前节点 instance ID, 传给 async source 做上下文过滤。 */
  nodeId?: string
  /** 当前节点 kind, 传给 async source 做上下文过滤。 */
  specKind?: string
  /** 当前节点 literal/config 输入快照, 传给 async source。 */
  currentInputs?: Record<string, unknown>
  /** 动态输入名 (config.Inputs[] 声明) — 仅 code widget 用, 进脚本补全。 */
  inputNames?: string[]
  /** 容器变量 (名+类型) — 仅 code widget 用: varname pin 值位补全 + 参考面板 + 新建变量。 */
  declaredVars?: { name: string; type: VarType }[]
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: any): void
  (e: 'declare-var', a: { name: string; type: VarType; default: unknown }): void
  (e: 'async-option-selected', payload: { value: unknown; meta: Record<string, unknown> }): void
}>()

const kind = computed(() => props.widgetKind ?? '')

// 所有写回都过这里: 按 pin 规范类型 coerce, widget 配漏也不会存错类型 (见 coerceLiteral)。
// 旧数据里"数字 pin 存了数字字符串"这类历史脏值, 不在这里静默自改, 走「检查」面板显式「修复」。
function commit(v: unknown) {
  emit('update:modelValue', coerceLiteral(v, props.type))
}

const numModel = computed(() => {
  const n = Number(props.modelValue)
  return Number.isFinite(n) ? n : 0
})

const selectItems = computed(() =>
  (props.options ?? []).map((o) => ({ value: o.value, label: t(o.labelKey) })),
)

const asyncOptions = ref<AsyncOption[]>([])
const asyncLoading = ref(false)
const asyncError = ref('')
let asyncLoadSeq = 0

const asyncSelectItems = computed(() =>
  asyncOptions.value.map((o) => ({
    value: String(o.value ?? ''),
    label: o.label || String(o.value ?? ''),
  })),
)

async function loadAsyncOptions() {
  if (kind.value !== 'async-dropdown' || !props.asyncSource) {
    asyncOptions.value = []
    asyncLoading.value = false
    asyncError.value = ''
    return
  }
  const seq = ++asyncLoadSeq
  asyncLoading.value = true
  asyncError.value = ''
  try {
    const options = await NodeService.AsyncOptions(
      props.nodeId ?? '',
      props.specKind ?? '',
      props.asyncSource,
      props.currentInputs ?? {},
    )
    if (seq !== asyncLoadSeq) return
    asyncOptions.value = (options ?? []) as AsyncOption[]
  } catch (err) {
    if (seq !== asyncLoadSeq) return
    asyncOptions.value = []
    asyncError.value = err instanceof Error ? err.message : String(err)
  } finally {
    if (seq === asyncLoadSeq) asyncLoading.value = false
  }
}

function onAsyncValue(v: unknown) {
  const value = normalizeAsyncDropdownValue(v)
  commit(value)
  const payload = asyncOptionPayloadForValue(asyncOptions.value, value)
  if (payload) emit('async-option-selected', payload)
}

watch(
  () => [
    kind.value,
    props.asyncSource,
    props.nodeId,
    props.specKind,
    JSON.stringify(props.currentInputs ?? {}),
  ],
  () => {
    void loadAsyncOptions()
  },
  { immediate: true },
)

// JSON editor: 维护 rawText / jsonValid。仅 valid 时 emit 解析值; invalid 保留原文不回退、不写回。
const rawText = ref('')
const jsonValid = ref(true)
function syncRaw(v: any) {
  if (v == null) {
    rawText.value = ''
  } else {
    try {
      rawText.value = JSON.stringify(v, null, 2)
    } catch {
      rawText.value = ''
    }
  }
  jsonValid.value = true
}
// 外部值变化 (切节点 / undo) 且当前不是 invalid 编辑态时同步显示。
watch(
  () => props.modelValue,
  (v) => {
    if (jsonValid.value) syncRaw(v)
  },
  { immediate: true },
)
function onJsonInput(text: string) {
  rawText.value = text
  if (text.trim() === '') {
    jsonValid.value = true
    commit(null)
    return
  }
  try {
    const parsed = JSON.parse(text)
    jsonValid.value = true
    commit(parsed)
  } catch {
    jsonValid.value = false // 保留 rawText, 不 emit — 半截 JSON 不覆盖已存值
  }
}
</script>
