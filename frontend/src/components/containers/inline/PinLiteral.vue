<template>
  <!-- Inline pin literal — type-polymorphic input.
       NodeInspector + 画布节点 (ContainerFlowNode) 用它编辑未连接 data-in pin 的 config.literal[pinName]。
       有 dropdown widget (widgetKind + options) 时优先出下拉, 否则按 type 出 number/bool/text。 -->
  <!-- USelect (Nuxt UI) 设了 inheritAttrs:false, 会丢掉父级 fallthrough 的原生
       @mousedown/@click 监听 → 画布上 vue-flow 把按下当节点拖动。套一层普通 div
       承接 nodrag + stop (含 pointerdown), 保证下拉可正常点开而不触发拖动。 -->
  <div
    v-if="widgetKind === 'dropdown' && (options?.length ?? 0) > 0"
    class="w-full nodrag"
    @pointerdown.stop
    @mousedown.stop
    @click.stop
  >
    <USelect
      class="w-full"
      :model-value="modelValue == null ? '' : String(modelValue)"
      :items="selectItems"
      size="xs"
      @update:model-value="(v: any) => commit(v)"
    />
  </div>
  <!-- list pin — wire-only, 不渲染可编辑 input 防手输垃圾 literal -->
  <span v-else-if="type === 'list'" class="text-xs text-dimmed italic">{{
    t('containers.listPinWireOnly')
  }}</span>
  <UInput
    v-else-if="type === 'number'"
    type="number"
    :model-value="modelValue ?? 0"
    size="xs"
    @update:model-value="(v: any) => commit(v)"
  />
  <UCheckbox
    v-else-if="type === 'bool'"
    :model-value="!!modelValue"
    @update:model-value="(v: boolean) => commit(v)"
  />
  <UInput
    v-else
    :model-value="modelValue == null ? '' : String(modelValue)"
    size="xs"
    :placeholder="placeholder"
    @update:model-value="(v: any) => commit(v)"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { coerceLiteral } from './coerceLiteral'
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

// 写回统一过 coerce: 按 pin 规范类型收口, 不靠各分支自己 String()/Number() (会漏)。
function commit(v: unknown) {
  emit('update:modelValue', coerceLiteral(v, props.type))
}

const selectItems = computed(() =>
  (props.options ?? []).map((o) => ({ value: o.value, label: t(o.labelKey) })),
)
</script>
