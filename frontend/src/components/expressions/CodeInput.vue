<!-- Script 代码编辑器 (widget kind 'code', PinInput 分发) — CodeMirror 6 + JS 语法:
     高亮 + 补全 (节点函数/糖函数/动态输入名/vars.get 串内变量名)。
     语法错由后端 validator (SCRIPT_PARSE_ERROR) 权威报。
     右上放大按钮弹 EditorModal 大编辑器: 工具栏/片段/参考面板 (节点用法可展开)/新建变量。 -->
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
      ref="editorModalRef"
      v-model:open="modalOpen"
      :model-value="modelValue"
      :title="t('inspector.code_editor_title')"
      :extensions="modalExtensions"
      :reference="referenceItems"
      :snippets="SNIPPETS"
      commentable
      @update:model-value="(v: string) => emit('update:modelValue', v)"
    >
      <template #panel-actions>
        <UButton
          v-if="declaredVars"
          icon="i-tabler-variable-plus"
          variant="ghost"
          color="neutral"
          size="xs"
          :title="t('inspector.editor_new_var')"
          @click="newVarOpen = true"
        />
      </template>
    </EditorModal>
    <NewVarModal
      v-model:open="newVarOpen"
      initial-name=""
      :existing-var-names="(declaredVars ?? []).map((v) => v.name)"
      @confirm="onNewVar"
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
  type InsertItem,
} from '@/lib/scriptCompletions'
import type { VarType } from '@/lib/variableRef'
import NewVarModal from '@/components/containers/NewVarModal.vue'
import EditorModal, { type RefItem } from './EditorModal.vue'

const { t, te } = useI18n()

const props = defineProps<{
  modelValue: string
  placeholder?: string
  /** 动态输入名 (config.Inputs[] 声明) — 进补全, 让脚本里直接引用。 */
  inputNames?: string[]
  /** 容器变量 (名+类型) — vars.get 串内补全 + 参考面板「变量」组 + 新建变量。 */
  declaredVars?: { name: string; type: VarType }[]
}>()

const emit = defineEmits<{
  'update:modelValue': [v: string]
  'declare-var': [args: { name: string; type: VarType; default: unknown }]
}>()

const host = ref<HTMLElement | null>(null)
const modalOpen = ref(false)
const newVarOpen = ref(false)
const editorModalRef = ref<InstanceType<typeof EditorModal> | null>(null)
let view: EditorView | null = null

const registry = useNodeRegistryStore()

function kindLabel(kind: string): string {
  const key = `node.${kind}.label`
  return te(key) ? t(key) : ''
}

const varNames = computed<string[]>(() => (props.declaredVars ?? []).map((v) => v.name))

const completionOptions = computed<Completion[]>(() => [
  ...nodeFnCompletions(registry.scriptBindableKinds, registry.specs, kindLabel),
  ...SUGAR_COMPLETIONS,
  ...(props.inputNames ?? []).map((n) => ({ label: n, type: 'variable' as const })),
])

// 常用片段 (工具栏下拉) — caretBack 把光标放进条件括号 / 循环体。
const SNIPPETS: InsertItem[] = [
  { label: 'if', insert: 'if () {\n  \n}', caretBack: 8 },
  { label: 'if / else', insert: 'if () {\n  \n} else {\n  \n}', caretBack: 20 },
  { label: 'for', insert: 'for (let i = 0; i < 10; i++) {\n  \n}', caretBack: 2 },
  { label: 'while', insert: 'while () {\n  \n}', caretBack: 8 },
  { label: 'try / catch', insert: 'try {\n  \n} catch (e) {\n  \n}', caretBack: 19 },
]

// 糖函数展开说明 (i18n key 用 _ 替代 ".": vue-i18n 把 "." 当层级分隔)。
function sugarDocs(label: string): string {
  const key = `script.fn.${label.replace(/\./g, '_')}`
  return te(key) ? t(key) : ''
}

// 节点 pin 参数表 (展开详情): 名 / i18n 人话 label / 类型 / 必填。
function nodeParams(kind: string): RefItem['params'] {
  const spec = registry.specs.get(kind)
  return (spec?.inputs ?? [])
    .filter((i) => i.type !== 'Exec')
    .map((i) => {
      const labelKey = `node.${kind}.input.${i.name}.label`
      return {
        name: i.name,
        label: te(labelKey) ? t(labelKey) : '',
        type: i.type,
        required: !!i.required,
      }
    })
}

function nodeText(kind: string, field: 'description' | 'example'): string {
  const key = `node.${kind}.${field}`
  return te(key) ? t(key) : ''
}

// 放大编辑的参考面板: 糖函数 / 可调节点 (可展开用法+参数) / 变量 / 本节点动态输入。
const referenceItems = computed<RefItem[]>(() => [
  ...SUGAR_ITEMS.map((it) => ({
    ...it,
    group: t('inspector.editor_ref_group_fns'),
    docs: sugarDocs(it.label) || undefined,
  })),
  ...nodeFnItems(registry.scriptBindableKinds, registry.specs, kindLabel).map((it) => ({
    ...it,
    group: t('inspector.editor_ref_group_nodes'),
    docs: nodeText(it.label, 'description') || undefined,
    example: nodeText(it.label, 'example') || undefined,
    params: nodeParams(it.label),
  })),
  ...(props.declaredVars ?? []).map((v) => ({
    label: v.name,
    detail: `vars.get("${v.name}")`,
    desc: v.type,
    insert: `vars.get("${v.name}")`,
    caretBack: 0,
    group: t('inspector.editor_ref_group_vars'),
  })),
  ...(props.inputNames ?? []).map((n) => ({
    label: n,
    insert: n,
    caretBack: 0,
    group: t('inspector.editor_ref_group_inputs'),
  })),
])

// 新建变量 (面板内直接建, 免去切侧栏的割裂): 声明上抛 + 顺手插一句 vars.get。
function onNewVar(a: { name: string; type: VarType; default: unknown }) {
  emit('declare-var', a)
  editorModalRef.value?.insert({ label: a.name, insert: `vars.get("${a.name}")`, caretBack: 0 })
}

// modal 的扩展工厂 — 不带 onChange (draft 由 modal 自管, 确认才回写)。
function modalExtensions() {
  return scriptEditorExtensions({
    completions: () => completionOptions.value,
    varNames: () => varNames.value,
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
        varNames: () => varNames.value,
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
