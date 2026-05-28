<!--
NodeInspector — 给定 Spec + 当前 config, 自动渲染整个 inspector form.
Phase 0 验证 declarative metadata schema 够用.
- Widget kind override 优先 spec, 没设 → framework 按 Type/Semantic 选默认
- VisibleWhen AND/OR 控制字段显隐
- Required 红星 + validate() 校验
- Advanced 折叠区
-->
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Spec, InputSpec } from '@bindings/yhbox/internal/node'

const { t } = useI18n()
import { evalVisibleRule } from './VisibleWhen'
import TextWidget from './widgets/TextWidget.vue'
import TextareaWidget from './widgets/TextareaWidget.vue'
import PasswordWidget from './widgets/PasswordWidget.vue'
import NumberWidget from './widgets/NumberWidget.vue'
import SliderWidget from './widgets/SliderWidget.vue'
import CheckboxWidget from './widgets/CheckboxWidget.vue'
import DropdownWidget from './widgets/DropdownWidget.vue'
import AsyncDropdownWidget from './widgets/AsyncDropdownWidget.vue'
import PointEditorWidget from './widgets/PointEditorWidget.vue'
import RectEditorWidget from './widgets/RectEditorWidget.vue'
import JsonEditorWidget from './widgets/JsonEditorWidget.vue'
import DurationWidget from './widgets/DurationWidget.vue'

const props = defineProps<{
  spec: Spec
  config: Record<string, any>
  nodeID: string
}>()
const emit = defineEmits<{ (e: 'update:config', v: Record<string, any>): void }>()

const errors = ref<Record<string, string>>({})

// Widget kind 默认表 (Type → kind), 节点 Widget.Kind override 优先
function resolveKind(input: InputSpec): string {
  if (input.widget?.kind) return input.widget.kind
  switch (input.type) {
    case 'String': return 'text'
    case 'Number': case 'Integer': return 'number'
    case 'Bool': return 'checkbox'
    case 'Point': return 'point-editor'
    case 'Rect': return 'rect-editor'
    case 'Duration': return 'duration'
    case 'JSON': return 'json'
    default: return 'text'
  }
}

function widgetComponent(kind: string) {
  switch (kind) {
    case 'text': return TextWidget
    case 'textarea': return TextareaWidget
    case 'password': return PasswordWidget
    case 'number': return NumberWidget
    case 'slider': return SliderWidget
    case 'checkbox': return CheckboxWidget
    case 'dropdown': return DropdownWidget
    case 'async-dropdown': return AsyncDropdownWidget
    case 'point-editor': return PointEditorWidget
    case 'rect-editor': return RectEditorWidget
    case 'json': return JsonEditorWidget
    case 'duration': return DurationWidget
    default: return null
  }
}

function updateField(name: string, value: any) {
  emit('update:config', { ...props.config, [name]: value })
}

// 取值: config → default → undefined
function valueFor(input: InputSpec): any {
  if (props.config[input.name] !== undefined) return props.config[input.name]
  return input.default
}

// 可见 + 非 exec 的 input
const visibleInputs = computed(() =>
  props.spec.inputs.filter(i =>
    i.type !== 'Exec' && evalVisibleRule(i.visibleWhen, props.config)
  )
)

const basicInputs = computed(() => visibleInputs.value.filter(i => !i.advanced))
const advancedInputs = computed(() => visibleInputs.value.filter(i => i.advanced))
const showAdvanced = ref(false)

// 给 async-dropdown 用: 当前所有 inputs 值快照
const currentInputs = computed<Record<string, any>>(() => {
  const out: Record<string, any> = {}
  for (const i of props.spec.inputs) {
    out[i.name] = valueFor(i)
  }
  return out
})

function validate(): boolean {
  errors.value = {}
  for (const input of visibleInputs.value) {
    if (input.required) {
      const v = valueFor(input)
      if (v === undefined || v === null || v === '') {
        errors.value[input.name] = `${input.displayName || input.name} ${t('common.required')}`
      }
    }
  }
  return Object.keys(errors.value).length === 0
}

defineExpose({ validate })
</script>

<template>
  <div class="node-inspector space-y-3 p-3">
    <div class="border-b border-default pb-2">
      <h3 class="text-sm font-semibold">{{ spec.displayName || spec.kind }}</h3>
      <p v-if="spec.description" class="text-xs text-toned mt-1">{{ spec.description }}</p>
    </div>

    <div v-for="input in basicInputs" :key="input.name" class="space-y-1">
      <label class="text-xs flex items-center gap-1">
        <span>{{ input.displayName || input.name }}</span>
        <span v-if="input.required" class="text-red-500">*</span>
      </label>
      <component
        :is="widgetComponent(resolveKind(input))"
        :input="input"
        :value="valueFor(input)"
        :node-i-d="nodeID"
        :spec-kind="spec.kind"
        :current-inputs="currentInputs"
        @update="(v: any) => updateField(input.name, v)"
      />
      <p v-if="input.doc" class="text-xs text-toned">{{ input.doc }}</p>
      <p v-if="errors[input.name]" class="text-xs text-red-500">{{ errors[input.name] }}</p>
    </div>

    <div v-if="advancedInputs.length > 0" class="pt-2 border-t border-default">
      <UButton
        size="xs"
        variant="ghost"
        :icon="showAdvanced ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
        @click="showAdvanced = !showAdvanced"
      >
        {{ showAdvanced ? t('editorAux.inspector_advanced_hide') : t('editorAux.inspector_advanced_show') }}{{ t('editorAux.inspector_advanced', { count: advancedInputs.length }) }}
      </UButton>
      <div v-if="showAdvanced" class="space-y-3 mt-2">
        <div v-for="input in advancedInputs" :key="input.name" class="space-y-1">
          <label class="text-xs flex items-center gap-1">
            <span>{{ input.displayName || input.name }}</span>
            <span v-if="input.required" class="text-red-500">*</span>
          </label>
          <component
            :is="widgetComponent(resolveKind(input))"
            :input="input"
            :value="valueFor(input)"
            :node-i-d="nodeID"
            :spec-kind="spec.kind"
            :current-inputs="currentInputs"
            @update="(v: any) => updateField(input.name, v)"
          />
          <p v-if="input.doc" class="text-xs text-toned">{{ input.doc }}</p>
          <p v-if="errors[input.name]" class="text-xs text-red-500">{{ errors[input.name] }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
