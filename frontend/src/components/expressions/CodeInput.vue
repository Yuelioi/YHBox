<!-- Script 代码编辑器 (widget kind 'code', PinInput 分发) — CodeMirror 6 + JS 语法:
     高亮 + 补全 (节点函数/糖函数/动态输入名/vars.get 串内变量名) + 悬停函数文档
     + 快速反馈 lint (语法错/未声明 $变量; 权威仍是后端 SCRIPT_PARSE_ERROR)。
     右上放大按钮弹 EditorModal 大编辑器: 工具栏/片段/折叠/参考面板/新建变量。 -->
<template>
  <div class="relative">
    <div
      ref="host"
      class="border border-default rounded-md overflow-hidden focus-within:border-primary/60"
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
      :lint-first="lintFirst"
      lang-label="JavaScript"
      commentable
      foldable
      @update:model-value="(v: string) => emit('update:modelValue', v)"
    >
      <template #toolbar-extra>
        <UButton
          v-if="declaredVars"
          icon="i-tabler-variable-plus"
          variant="ghost"
          color="neutral"
          size="xs"
          :title="t('inspector.editor_new_var')"
          @click="newVarOpen = true"
        >
          {{ t('inspector.editor_new_var') }}
        </UButton>
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
  scriptDollarRefs,
  scriptEditorExtensions,
  scriptSyntaxErrors,
  SUGAR_COMPLETIONS,
  SUGAR_ITEMS,
  type InsertItem,
} from '@/lib/scriptCompletions'
import type { HoverDoc } from '@/lib/editorHover'
import type { VarType } from '@/lib/variableRef'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { ALL_NODE_GROUPS, groupLabelColor } from '@/composables/editor/useNodeGroupColor'
import NewVarModal from '@/components/containers/NewVarModal.vue'
import EditorModal, { type RefItem } from './EditorModal.vue'

const { t, te } = useI18n()

const props = defineProps<{
  modelValue: string
  placeholder?: string
  /** 动态输入名 (config.Inputs[] 声明) — 进补全, 让脚本里直接引用。 */
  inputNames?: string[]
  /** 容器变量 (名+类型) — vars.get 串内补全 + $引用 lint + 参考面板「变量」组 + 新建变量。 */
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
  // $hp live getter (脚本里等价 vars.get("hp"), 实时读)
  ...(props.declaredVars ?? []).map((v) => ({
    label: `$${v.name}`,
    type: 'variable' as const,
    detail: v.type,
  })),
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

// 悬停文档: $变量 → 类型; 糖函数 → 签名+说明; 节点函数 → 签名+中文名/说明+参数表。
function hoverDoc(word: string): HoverDoc | null {
  if (word.startsWith('$')) {
    const v = (props.declaredVars ?? []).find((x) => x.name === word.slice(1))
    return v ? { sig: word, desc: v.type } : null
  }
  const sugar = SUGAR_ITEMS.find((s) => s.label === word)
  if (sugar) return { sig: sugar.detail ?? word, desc: sugarDocs(word) || undefined }
  if (registry.scriptBindableKinds.includes(word)) {
    const spec = registry.specs.get(word)
    if (!spec) return null
    const pins = (spec.inputs ?? []).filter((i) => i.type !== 'Exec').map((i) => i.name).join(', ')
    const desc = [kindLabel(word), nodeText(word, 'description')].filter(Boolean).join(' — ')
    return { sig: `${word}({${pins}})`, desc: desc || undefined, params: nodeParams(word) }
  }
  return null
}

const lintMessages = {
  syntaxError: (line: number) => t('inspector.editor_syntax_error_line', { line }),
  unknownVar: (name: string) => t('inspector.editor_unknown_var', { name }),
}

// modal 状态栏首条问题: 语法错优先, 其次未声明 $变量。
function lintFirst(doc: string): { message: string; from: number } | null {
  const syn = scriptSyntaxErrors(doc)[0]
  if (syn) return { message: lintMessages.syntaxError(syn.line), from: syn.from }
  const { refs, defined } = scriptDollarRefs(doc)
  const known = new Set(varNames.value)
  const bad = refs.find((r) => !known.has(r.name) && !defined.has(r.name))
  return bad ? { message: lintMessages.unknownVar(bad.name), from: bad.from } : null
}

// 节点按画布同款分类分组 (nodeGroup.* 标签 + 分类配色), 组内按 kind 排; 分类顺序对齐面板。
function groupedNodeItems(): RefItem[] {
  const items = nodeFnItems(registry.scriptBindableKinds, registry.specs, kindLabel).map((it) => {
    const g = getSpec(it.label)?.group ?? 'system'
    const gKey = `nodeGroup.${g}`
    return {
      ...it,
      group: te(gKey) ? t(gKey) : g,
      groupClass: groupLabelColor(g),
      docs: nodeText(it.label, 'description') || undefined,
      example: nodeText(it.label, 'example') || undefined,
      params: nodeParams(it.label),
      _g: g,
    }
  })
  const order = new Map(ALL_NODE_GROUPS.map((g, i) => [g as string, i]))
  items.sort(
    (a, b) => (order.get(a._g) ?? 99) - (order.get(b._g) ?? 99) || a.label.localeCompare(b.label),
  )
  return items
}

// 放大编辑的参考面板: 变量 / 糖函数 / 本节点动态输入 / 节点 (按分类, 可展开用法+参数)。
const referenceItems = computed<RefItem[]>(() => [
  ...(props.declaredVars ?? []).map((v) => ({
    label: `$${v.name}`,
    desc: v.type,
    insert: `$${v.name}`,
    caretBack: 0,
    group: t('inspector.editor_ref_group_vars'),
  })),
  ...SUGAR_ITEMS.map((it) => ({
    ...it,
    group: t('inspector.editor_ref_group_fns'),
    docs: sugarDocs(it.label) || undefined,
  })),
  ...(props.inputNames ?? []).map((n) => ({
    label: n,
    insert: n,
    caretBack: 0,
    group: t('inspector.editor_ref_group_inputs'),
  })),
  ...groupedNodeItems(),
])

// 新建变量 (工具栏直接建, 免去切侧栏的割裂): 声明上抛 + 顺手插一个 $引用。
function onNewVar(a: { name: string; type: VarType; default: unknown }) {
  emit('declare-var', a)
  editorModalRef.value?.insert({ label: a.name, insert: `$${a.name}`, caretBack: 0 })
}

function buildExtensions(opts: { modal?: boolean; onChange?: (doc: string) => void }) {
  return scriptEditorExtensions({
    completions: () => completionOptions.value,
    varNames: () => varNames.value,
    hoverDoc,
    lintMessages,
    placeholder: props.placeholder,
    minHeight: '10em',
    ...opts,
  })
}

// modal 的扩展工厂 — 不带 onChange (draft 由 modal 自管, 确认才回写)。
function modalExtensions() {
  return buildExtensions({ modal: true })
}

onMounted(() => {
  view = new EditorView({
    parent: host.value!,
    doc: props.modelValue ?? '',
    extensions: buildExtensions({ onChange: (doc) => emit('update:modelValue', doc) }),
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
