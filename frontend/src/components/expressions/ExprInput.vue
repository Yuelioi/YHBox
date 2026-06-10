<!-- Expr 表达式编辑器 (widget kind 'expr', PinInput 分发) — CodeMirror 6 内核:
     语法高亮 + 函数补全 (签名+说明) + 悬停函数文档 + 带位置红线 (lintExpr 启发式)。
     权威校验仍在后端 validator (EXPR_* 节点红错); 这里是打字时的快速反馈。
     右上放大按钮弹 EditorModal 大编辑器 (带函数参考面板)。扩展本体在 lib/exprEditorExtensions。 -->
<template>
  <div>
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
        v-model:open="modalOpen"
        :model-value="modelValue"
        :title="t('inspector.expr_editor_title')"
        icon="i-tabler-math-function"
        :extensions="modalExtensions"
        :reference="referenceItems"
        :lint-first="lintFirst"
        :lang-label="t('inspector.editor_lang_expr')"
        @update:model-value="(v: string) => emit('update:modelValue', v)"
      />
    </div>
    <p v-if="errorText" class="text-[10px] text-error mt-0.5">{{ errorText }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorView } from '@codemirror/view'
import { allExprFunctions, type ExprDiagnostic } from '@/lib/exprFunctions'
import { exprEditorExtensions, firstExprError } from '@/lib/exprEditorExtensions'
import EditorModal, { type RefItem } from './EditorModal.vue'

const { t, te } = useI18n()

const props = defineProps<{
  modelValue: string
  placeholder?: string
  /** 动态输入名 (config.Inputs[] 声明) — 进补全和参考面板。 */
  inputNames?: string[]
  /** 容器变量 (名+类型) — $ 补全和参考面板「变量」组。 */
  declaredVars?: { name: string; type: string }[]
}>()

const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const host = ref<HTMLElement | null>(null)
const modalOpen = ref(false)
let view: EditorView | null = null

// i18n 注入: 函数说明 + lint 诊断文案 (lib 保持纯函数)。
function fnDesc(name: string): string {
  const key = `expression.fn.${name}.desc`
  return te(key) ? t(key) : '' // 新函数没配说明 → 空, 不糊 raw key
}

function diagMessage(d: ExprDiagnostic): string {
  return te(d.messageKey) ? t(d.messageKey, d.params ?? {}) : d.messageKey
}

const errorText = computed<string>(
  () => firstExprError(props.modelValue ?? '', diagMessage)?.message ?? '',
)

function lintFirst(doc: string): { message: string; from: number } | null {
  return firstExprError(doc, diagMessage)
}

// 放大编辑的参考面板: 变量 ($引用) / 全部表达式函数 / 本节点动态输入, 点击插入。
const referenceItems = computed<RefItem[]>(() => [
  ...(props.declaredVars ?? []).map((v) => ({
    label: `$${v.name}`,
    desc: v.type,
    insert: `$${v.name}`,
    caretBack: 0,
    group: t('inspector.editor_ref_group_vars'),
  })),
  ...allExprFunctions().map((f) => ({
    label: f.name,
    detail: f.sig,
    desc: fnDesc(f.name) || undefined,
    insert: `${f.name}()`,
    caretBack: f.maxArgs === 0 ? 0 : 1,
    group: t('inspector.editor_ref_group_fns'),
  })),
  ...(props.inputNames ?? []).map((n) => ({
    label: n,
    insert: n,
    caretBack: 0,
    group: t('inspector.editor_ref_group_inputs'),
  })),
])

function buildExtensions(opts: { modal?: boolean; onChange?: (doc: string) => void } = {}) {
  return exprEditorExtensions({
    fnDesc,
    diagMessage,
    inputNames: () => props.inputNames ?? [],
    varNames: () => (props.declaredVars ?? []).map((v) => v.name),
    placeholder: props.placeholder,
    minHeight: '3.9em',
    ...opts,
  })
}

// modal 的扩展工厂 — 不带 onChange (draft 由 modal 自管, 确认才回写)。
function modalExtensions() {
  return buildExtensions({ modal: true })
}

// ── 小框 EditorView 生命周期 + v-model 双向 ──

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
