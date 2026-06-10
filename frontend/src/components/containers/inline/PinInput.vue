<template>
  <!-- Inspector 端 widget-aware pin 编辑器 — 按 backend widget kind 选控件, 写回 config.literal[pin]。
       画布端的 scalar 内联编辑仍用 PinLiteral.vue (本组件不上画布)。 -->
  <!-- checkbox / bool -->
  <UCheckbox
    v-if="kind === 'checkbox' || (kind === '' && type === 'bool')"
    :model-value="!!modelValue"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  />

  <!-- number / slider -->
  <UInputNumber
    v-else-if="kind === 'number' || kind === 'slider' || (kind === '' && type === 'number')"
    :model-value="numModel"
    :min="min"
    :max="max"
    :step="step"
    size="sm"
    class="w-full"
    @update:model-value="(v: number) => emit('update:modelValue', Number.isFinite(v) ? v : 0)"
  />

  <!-- static dropdown (枚举, options 内联) -->
  <USelect
    v-else-if="kind === 'dropdown' && (options?.length ?? 0) > 0"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :items="selectItems"
    size="sm"
    class="w-full"
    @update:model-value="(v: any) => emit('update:modelValue', String(v))"
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

  <!-- expr (Expr Expression) — 函数补全 + 即时语法红错 + 放大编辑 modal -->
  <ExprInput
    v-else-if="kind === 'expr'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :placeholder="placeholder"
    :input-names="inputNames"
    @update:model-value="(v: string) => emit('update:modelValue', v)"
  />

  <!-- code (Script.Code) — JS 编辑器: 节点函数/糖函数/动态输入/变量补全 + 放大编辑 modal -->
  <CodeInput
    v-else-if="kind === 'code'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :placeholder="placeholder"
    :input-names="inputNames"
    :declared-vars="declaredVars"
    @update:model-value="(v: string) => emit('update:modelValue', v)"
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
    @update:model-value="(v: string) => emit('update:modelValue', v)"
  />

  <!-- key-capture: 聚焦后按物理键自动填 vk (KeyPress/KeyHold* 的 VK 字段) -->
  <KeyCapture
    v-else-if="kind === 'key-capture'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    @update:model-value="(v: string) => emit('update:modelValue', v)"
  />

  <!-- color-preset: 视觉注册中心色块选择器 (CommentBox.Color 等), 存 palette key -->
  <ColorPalettePicker
    v-else-if="kind === 'color-preset'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    @update:model-value="(v: string) => emit('update:modelValue', v)"
  />

  <!-- icon-preset: 视觉注册中心图标选择器 (CommentBox.Icon 等), 存完整 tabler 名 -->
  <IconPicker
    v-else-if="kind === 'icon-preset'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    @update:model-value="(v: string) => emit('update:modelValue', v)"
  />

  <!-- list pin — wire-only, 不渲染可编辑 input 防手输垃圾 literal -->
  <span
    v-else-if="type === 'list'"
    class="text-xs text-dimmed italic"
  >{{ t('containers.listPinWireOnly') }}</span>

  <!-- text / password / duration / async-dropdown / 默认 → 文本框
       async-dropdown 的候选源 (templateKeys/clipIDs/subgraphIDs) 多由 bespoke section 处理;
       走到这里的 (e.g. WaitTemplate.Template) 当字符串 key 编辑 (跟旧 literal section 一致)。 -->
  <UInput
    v-else
    :model-value="modelValue == null ? '' : String(modelValue)"
    size="sm"
    class="w-full"
    :type="kind === 'password' ? 'password' : 'text'"
    :placeholder="placeholder"
    @update:model-value="(v: any) => emit('update:modelValue', String(v))"
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
import type { PinType } from '../pinSpec'
import type { VarType } from '@/lib/variableRef'

const { t } = useI18n()

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
  /** 动态输入名 (config.Inputs[] 声明) — 仅 code widget 用, 进脚本补全。 */
  inputNames?: string[]
  /** 容器变量 (名+类型) — 仅 code widget 用: vars.get 补全 + 参考面板 + 新建变量。 */
  declaredVars?: { name: string; type: VarType }[]
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: any): void
  (e: 'declare-var', a: { name: string; type: VarType; default: unknown }): void
}>()

const kind = computed(() => props.widgetKind ?? '')

const numModel = computed(() => {
  const n = Number(props.modelValue)
  return Number.isFinite(n) ? n : 0
})

const selectItems = computed(() =>
  (props.options ?? []).map((o) => ({ value: o.value, label: t(o.labelKey) })),
)

// JSON editor: 维护 rawText / jsonValid。仅 valid 时 emit 解析值; invalid 保留原文不回退、不写回。
const rawText = ref('')
const jsonValid = ref(true)
function syncRaw(v: any) {
  if (v == null) {
    rawText.value = ''
  } else if (typeof v === 'string') {
    rawText.value = v
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
    emit('update:modelValue', null)
    return
  }
  try {
    const parsed = JSON.parse(text)
    jsonValid.value = true
    emit('update:modelValue', parsed)
  } catch {
    jsonValid.value = false // 保留 rawText, 不 emit — 半截 JSON 不覆盖已存值
  }
}
</script>
