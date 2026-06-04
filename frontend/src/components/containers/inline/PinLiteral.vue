<template>
  <!-- Inline pin literal — type-polymorphic input.
       NodeInspector + 画布节点 (ContainerFlowNode) 用它编辑未连接 data-in pin 的 config.literal[pinName]。
       有 dropdown widget (widgetKind + options) 时优先出下拉, 否则按 type 出 number/bool/text。 -->
  <USelect
    v-if="widgetKind === 'dropdown' && (options?.length ?? 0) > 0"
    class="w-full"
    :model-value="modelValue == null ? '' : String(modelValue)"
    :items="selectItems"
    size="xs"
    @update:model-value="(v: any) => emit('update:modelValue', String(v))"
  />
  <UInput
    v-else-if="type === 'number'"
    type="number"
    :model-value="modelValue ?? 0"
    size="xs"
    @update:model-value="(v: any) => emit('update:modelValue', Number(v) || 0)"
  />
  <UCheckbox
    v-else-if="type === 'bool'"
    :model-value="!!modelValue"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  />
  <UInput
    v-else
    :model-value="modelValue == null ? '' : String(modelValue)"
    size="xs"
    :placeholder="placeholder"
    @update:model-value="(v: any) => emit('update:modelValue', String(v))"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PinType } from '../pinSpec'

const { t } = useI18n()

const props = defineProps<{
  type: PinType
  modelValue: any
  placeholder?: string
  /** 后端 widget kind (见 FieldSchema.widgetKind)。'dropdown' + options 时画布上也出下拉。 */
  widgetKind?: string
  /** dropdown 选项. label 是 i18n key (node.<kind>.input.<name>.option.<value>), 走 t()。 */
  options?: Array<{ value: string; labelKey: string }>
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: any): void }>()

const selectItems = computed(() =>
  (props.options ?? []).map((o) => ({ value: o.value, label: t(o.labelKey) })),
)
</script>
