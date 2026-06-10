<template>
  <!-- geometry widget — 最高优先级, 整棵子树交给 GeometryWidget -->
  <GeometryWidget
    v-if="schema.widget === 'geometry'"
    :model-value="modelValue"
    :field-path="fieldPath"
    @update:model-value="emitVal"
  />

  <!-- object: 有字段列表 → 递归分组 -->
  <div v-else-if="schema.type === 'object'" class="space-y-3">
    <!-- 文本模式切换按钮 (右上角). 标签由调用方渲染 (NodeInspector 外层 label / 父级 childLabel),
         这里不再自渲染否则双标签。 -->
    <div class="flex items-center justify-end">
      <div class="flex items-center gap-1">
        <UTooltip
          v-if="textMode && !jsonOk"
          :text="t('structured_input.json_error_tooltip')"
        >
          <span class="text-[10px] text-error">JSON {{ t('structured_input.json_invalid') }}</span>
        </UTooltip>
        <UButton
          v-if="schema.widget === 'colorRange'"
          data-testid="eyedropper-btn"
          size="xs"
          variant="ghost"
          color="primary"
          icon="i-tabler-color-picker"
          :title="t('structured_input.pick_color')"
          @click="emit('pick-color', fieldPath)"
        />
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          :icon="textMode ? 'i-tabler-forms' : 'i-tabler-code'"
          :disabled="textMode && !jsonOk"
          :title="textMode ? t('structured_input.switch_to_struct') : t('structured_input.switch_to_text')"
          @click="toggleTextMode"
        />
      </div>
    </div>

    <!-- 文本模式 -->
    <div v-if="textMode" class="space-y-1">
      <UTextarea
        :model-value="rawJson"
        :rows="4"
        size="sm"
        class="w-full font-mono text-[11px]"
        :color="jsonOk ? undefined : 'error'"
        @update:model-value="onRawJsonInput"
      />
      <p v-if="!jsonOk" class="text-[10px] text-error leading-snug">{{ jsonError }}</p>
      <UButton
        v-if="!jsonOk"
        size="xs"
        variant="ghost"
        color="neutral"
        @click="abandonTextMode"
      >
        {{ t('structured_input.abandon_changes') }}
      </UButton>
    </div>

    <!-- 结构模式: 递归渲染每个 field -->
    <div v-else class="pl-2 border-l border-default/40 space-y-3">
      <div
        v-for="field in schema.fields ?? []"
        :key="field.key"
        class="space-y-1"
      >
        <label class="block text-[11px] text-toned">
          {{ childLabel(field.key) }}
          <span v-if="field.required && isEmpty(modelValue?.[field.key])" class="text-[10px] text-error ml-1">{{ t('structured_input.field_required') }}</span>
        </label>
        <StructuredInput
          :schema="field.schema"
          :model-value="modelValue?.[field.key]"
          :field-path="fieldPath + '.' + field.key"
          :kind="kind"
          @update:model-value="(v: any) => updateChild(field.key, v)"
        />
      </div>
    </div>
  </div>

  <!-- tuple: 定长数组, 逐位置渲染带标签子项 (DetectColor.Range 等). 结构 ⇄ JSON 双模式同 object. -->
  <div v-else-if="schema.type === 'tuple'" class="space-y-3">
    <div class="flex items-center justify-end">
      <div class="flex items-center gap-1">
        <UTooltip
          v-if="textMode && !jsonOk"
          :text="t('structured_input.json_error_tooltip')"
        >
          <span class="text-[10px] text-error">JSON {{ t('structured_input.json_invalid') }}</span>
        </UTooltip>
        <UButton
          v-if="schema.widget === 'colorRange'"
          data-testid="eyedropper-btn"
          size="xs"
          variant="ghost"
          color="primary"
          icon="i-tabler-color-picker"
          :title="t('structured_input.pick_color')"
          @click="emit('pick-color', fieldPath)"
        />
        <UButton
          size="xs"
          variant="ghost"
          color="neutral"
          :icon="textMode ? 'i-tabler-forms' : 'i-tabler-code'"
          :disabled="textMode && !jsonOk"
          :title="textMode ? t('structured_input.switch_to_struct') : t('structured_input.switch_to_text')"
          @click="toggleTextMode"
        />
      </div>
    </div>

    <!-- 文本模式: 整个数组当 JSON 编辑 -->
    <div v-if="textMode" class="space-y-1">
      <UTextarea
        :model-value="rawJson"
        :rows="4"
        size="sm"
        class="w-full font-mono text-[11px]"
        :color="jsonOk ? undefined : 'error'"
        @update:model-value="onRawJsonInput"
      />
      <p v-if="!jsonOk" class="text-[10px] text-error leading-snug">{{ jsonError }}</p>
      <UButton
        v-if="!jsonOk"
        size="xs"
        variant="ghost"
        color="neutral"
        @click="abandonTextMode"
      >
        {{ t('structured_input.abandon_changes') }}
      </UButton>
    </div>

    <!-- 结构模式: 按位置逐项渲染 (childLabel 走 fieldPath.<key> i18n) -->
    <div v-else class="pl-2 border-l border-default/40 space-y-3">
      <div
        v-for="(field, idx) in schema.fields ?? []"
        :key="field.key"
        class="space-y-1"
      >
        <label class="block text-[11px] text-toned">{{ childLabel(field.key) }}</label>
        <StructuredInput
          :schema="field.schema"
          :model-value="Array.isArray(modelValue) ? modelValue[idx] : undefined"
          :field-path="fieldPath + '.' + field.key"
          :kind="kind"
          @update:model-value="(v: any) => updateTupleChild(idx, v)"
        />
      </div>
    </div>
  </div>

  <!-- scalar: number -->
  <UInputNumber
    v-else-if="schema.type === 'number'"
    :model-value="numModel"
    size="sm"
    class="w-full"
    @update:model-value="(v: number) => emitVal(Number.isFinite(v) ? v : 0)"
  />

  <!-- scalar: string -->
  <UInput
    v-else-if="schema.type === 'string'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    size="sm"
    class="w-full"
    @update:model-value="(v: any) => emitVal(String(v))"
  />

  <!-- scalar: bool -->
  <UCheckbox
    v-else-if="schema.type === 'bool'"
    :model-value="!!modelValue"
    @update:model-value="(v: boolean) => emitVal(v)"
  />

  <!-- enum: USelect -->
  <USelect
    v-else-if="schema.type === 'enum'"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :items="enumItems"
    size="sm"
    class="w-full"
    @update:model-value="(v: any) => emitVal(v)"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { NodeFieldSchema } from '@/components/containers/nodeRegistry/index'
import GeometryWidget from './GeometryWidget.vue'

const { t, te } = useI18n()

const props = defineProps<{
  schema: NodeFieldSchema
  modelValue: any
  fieldPath: string
  kind: string
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', v: any): void
  (e: 'pick-color', fieldPath: string): void
}>()

// ─── emit helper ──────────────────────────────────────────────────────────────
function emitVal(v: any) {
  emit('update:modelValue', v)
}

// ─── label helpers ────────────────────────────────────────────────────────────
// 标签由调用方渲染 (NodeInspector 外层 label / 父级 childLabel); object/tuple 自身不渲染
// 顶层标签, 故无 fieldLabel — 只有给子项用的 childLabel。
function childLabel(key: string): string {
  const childPath = `${props.fieldPath}.${key}`
  const k = `node.${props.kind}.input.${childPath}.label`
  return te(k) ? t(k) : key
}

// ─── number model ─────────────────────────────────────────────────────────────
const numModel = computed(() => {
  const n = Number(props.modelValue)
  return Number.isFinite(n) ? n : 0
})

// ─── enum items ───────────────────────────────────────────────────────────────
const enumItems = computed(() =>
  (props.schema.options ?? []).map((opt) => {
    const optKey = `node.${props.kind}.input.${props.fieldPath}.option.${String(opt.value)}`
    return {
      value: String(opt.value),
      label: te(optKey) ? t(optKey) : String(opt.value),
    }
  }),
)

// ─── object updateChild ───────────────────────────────────────────────────────
function updateChild(key: string, v: any) {
  const next = { ...props.modelValue, [key]: v }
  emitVal(next)
}

// ─── tuple updateChild ──────────────────────────────────────────────────────────
// 每次按 schema.fields 重建满长度数组 (空位填类型零值), 保证数组稠密 —
// 否则稀疏位置会序列化成 null, 后端 parseRange6 之类按位置读会报「不是数字」。
function tupleZero(type?: string): any {
  if (type === 'number') return 0
  if (type === 'string') return ''
  if (type === 'bool') return false
  return null
}
function updateTupleChild(idx: number, v: any) {
  const fields = props.schema.fields ?? []
  const cur = Array.isArray(props.modelValue) ? props.modelValue : []
  const arr = fields.map((f, j) => {
    if (j === idx) return v
    const existing = cur[j]
    if (existing !== undefined && existing !== null) return existing
    return tupleZero(f.schema?.type)
  })
  emitVal(arr)
}

function isEmpty(v: any): boolean {
  return v == null || v === '' || (typeof v === 'object' && Object.keys(v).length === 0)
}

// ─── text mode (object + geometry non-leaf toggle) ────────────────────────────
const textMode = ref(false)
const rawJson = ref('')
const jsonOk = ref(true)
const jsonError = ref('')
/** last known valid value — used to restore on abandon */
const lastValid = ref<any>(undefined)

function syncRawFromValue() {
  try {
    rawJson.value = JSON.stringify(props.modelValue, null, 2)
  } catch {
    rawJson.value = ''
  }
  jsonOk.value = true
  jsonError.value = ''
  lastValid.value = props.modelValue
}

watch(
  () => props.modelValue,
  () => {
    if (!textMode.value) {
      lastValid.value = props.modelValue
    }
  },
  { immediate: true },
)

function toggleTextMode() {
  if (!textMode.value) {
    // entering text mode
    syncRawFromValue()
    textMode.value = true
  } else {
    // leaving text mode (only allowed when JSON is valid)
    if (jsonOk.value) {
      textMode.value = false
    }
  }
}

function abandonTextMode() {
  // restore last valid value (don't emit — value already emitted when it was valid)
  textMode.value = false
  jsonOk.value = true
  jsonError.value = ''
}

function validateAgainstSchema(value: unknown, schema: NodeFieldSchema): string | null {
  if (schema.type === 'object') {
    if (typeof value !== 'object' || value === null || Array.isArray(value)) {
      return 'expected object'
    }
    const obj = value as Record<string, unknown>
    const declaredKeys = new Set((schema.fields ?? []).map((f) => f.key))
    for (const k of Object.keys(obj)) {
      if (!declaredKeys.has(k)) return `unknown key: ${k}`
    }
    for (const f of schema.fields ?? []) {
      if (f.required && (!(f.key in obj) || obj[f.key] == null)) {
        return `required field missing: ${f.key}`
      }
      if (f.key in obj) {
        const childErr = validateAgainstSchema(obj[f.key], f.schema)
        if (childErr) return `${f.key}: ${childErr}`
      }
    }
    return null
  }
  if (schema.type === 'tuple') {
    if (!Array.isArray(value)) return 'expected array'
    const fields = schema.fields ?? []
    for (let i = 0; i < fields.length && i < value.length; i++) {
      const childErr = validateAgainstSchema(value[i], fields[i].schema)
      if (childErr) return `[${i}]: ${childErr}`
    }
    return null
  }
  if (schema.type === 'number') {
    if (typeof value !== 'number') return 'expected number'
    return null
  }
  if (schema.type === 'string') {
    if (typeof value !== 'string') return 'expected string'
    return null
  }
  if (schema.type === 'bool') {
    if (typeof value !== 'boolean') return 'expected boolean'
    return null
  }
  if (schema.type === 'enum') {
    // any primitive is acceptable
    return null
  }
  return null
}

function onRawJsonInput(text: string) {
  rawJson.value = text
  if (text.trim() === '') {
    jsonOk.value = true
    jsonError.value = ''
    emitVal(null)
    return
  }
  try {
    const parsed = JSON.parse(text)
    const err = validateAgainstSchema(parsed, props.schema)
    if (err) {
      jsonOk.value = false
      jsonError.value = err
      return
    }
    jsonOk.value = true
    jsonError.value = ''
    lastValid.value = parsed
    emitVal(parsed)
  } catch (e: any) {
    jsonOk.value = false
    jsonError.value = String(e?.message ?? e)
  }
}
</script>
