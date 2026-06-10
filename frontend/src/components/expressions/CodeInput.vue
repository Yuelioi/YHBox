<!-- Script 代码编辑器 (widget kind 'code', PinInput 分发) — CodeMirror 6 + JS 语法:
     高亮 + 补全 (节点函数/糖函数/动态输入名)。语法错由后端 validator (SCRIPT_PARSE_ERROR) 权威报。
     右上放大按钮弹 EditorModal 大编辑器 (带参考面板)。 -->
<template>
  <div class="relative">
    <div
      ref="host"
      class="bg-elevated/80 border border-default rounded-md overflow-hidden focus-within:border-emerald-500"
    />
    <UButton
      icon="i-tabler-maximize"
      variant="ghost"
      color="neutral"
      size="xs"
      class="absolute top-1 right-1 opacity-60 hover:opacity-100"
      :title="t('inspector.code_expand')"
      @click="modalOpen = true"
    />
    <EditorModal
      v-model:open="modalOpen"
      :model-value="modelValue"
      :title="t('inspector.code_editor_title')"
      :extensions="modalExtensions"
      :reference="referenceItems"
      @update:model-value="(v: string) => emit('update:modelValue', v)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorView } from '@codemirror/view'
import type { Completion } from '@codemirror/autocomplete'
import { useNodeRegistryStore } from '@/stores/nodeRegistry'
import {
  nodeFnCompletions,
  nodeFnItems,
  scriptEditorExtensions,
  SUGAR_COMPLETIONS,
  SUGAR_ITEMS,
} from '@/lib/scriptCompletions'
import EditorModal, { type RefItem } from './EditorModal.vue'

const { t, te } = useI18n()

const props = defineProps<{
  modelValue: string
  placeholder?: string
  /** 动态输入名 (config.Inputs[] 声明) — 进补全, 让脚本里直接引用。 */
  inputNames?: string[]
}>()

const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const host = ref<HTMLElement | null>(null)
const modalOpen = ref(false)
let view: EditorView | null = null

const registry = useNodeRegistryStore()

function kindLabel(kind: string): string {
  const key = `node.${kind}.label`
  return te(key) ? t(key) : ''
}

const completionOptions = computed<Completion[]>(() => [
  ...nodeFnCompletions(registry.scriptBindableKinds, registry.specs, kindLabel),
  ...SUGAR_COMPLETIONS,
  ...(props.inputNames ?? []).map((n) => ({ label: n, type: 'variable' as const })),
])

// 放大编辑的参考面板: 糖函数 / 可调节点 / 本节点动态输入, 点击插入。
const referenceItems = computed<RefItem[]>(() => [
  ...SUGAR_ITEMS.map((it) => ({ ...it, group: t('inspector.editor_ref_group_fns') })),
  ...nodeFnItems(registry.scriptBindableKinds, registry.specs, kindLabel).map((it) => ({
    ...it,
    group: t('inspector.editor_ref_group_nodes'),
  })),
  ...(props.inputNames ?? []).map((n) => ({
    label: n,
    insert: n,
    caretBack: 0,
    group: t('inspector.editor_ref_group_inputs'),
  })),
])

// modal 的扩展工厂 — 不带 onChange (draft 由 modal 自管, 确认才回写)。
function modalExtensions() {
  return scriptEditorExtensions({
    completions: () => completionOptions.value,
    placeholder: props.placeholder,
  })
}

const theme = EditorView.theme({
  '&': { backgroundColor: 'transparent', fontSize: '11px' },
  '&.cm-focused': { outline: 'none' },
  '.cm-content': { fontFamily: 'ui-monospace, monospace', minHeight: '10em', padding: '4px 0' },
  '.cm-line': { padding: '0 6px' },
  '.cm-tooltip': { fontSize: '11px' },
}, { dark: true })

onMounted(() => {
  view = new EditorView({
    parent: host.value!,
    doc: props.modelValue ?? '',
    extensions: [
      ...scriptEditorExtensions({
        completions: () => completionOptions.value,
        placeholder: props.placeholder,
        onChange: (doc) => emit('update:modelValue', doc),
      }),
      theme,
    ],
  })
})

watch(() => props.modelValue, (v) => {
  if (!view) return
  const cur = view.state.doc.toString()
  if (v !== cur) {
    view.dispatch({ changes: { from: 0, to: cur.length, insert: v ?? '' } })
  }
})

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})
</script>
