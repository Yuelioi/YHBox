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

  <!-- textarea (e.g. Expr Expression) -->
  <UTextarea
    v-else-if="kind === 'textarea'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :rows="3"
    size="sm"
    class="w-full font-mono"
    :placeholder="placeholder"
    @update:model-value="(v: string) => emit('update:modelValue', v)"
  />

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
import type { PinType } from '../pinSpec'

const { t } = useI18n()

const props = defineProps<{
  /** PinType (number/bool/string/point/any) — widgetKind 缺失时的 fallback 渲染依据。 */
  type: PinType
  /** 原始 backend widget kind (见 FieldSchema.widgetKind)。空串 = 无 spec 字段 (Expr 动态输入), 走 type fallback。 */
  widgetKind?: string
  modelValue: any
  options?: Array<{ value: string; labelKey: string }>
  placeholder?: string
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: any): void }>()

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
