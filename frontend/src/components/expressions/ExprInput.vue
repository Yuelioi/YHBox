<!-- Expr 表达式编辑器 (widget kind 'expr', PinInput 分发) — CodeMirror 6 内核:
     语法高亮 + 函数补全 (签名+说明) + 带位置红线 (lintExpr 启发式)。
     权威校验仍在后端 validator (EXPR_* 节点红错); 这里是打字时的快速反馈。 -->
<template>
  <div>
    <div
      ref="host"
      class="bg-elevated/80 border border-default rounded-md overflow-hidden focus-within:border-emerald-500"
    />
    <p v-if="errorText" class="text-[10px] text-rose-300/90 mt-0.5">{{ errorText }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorView, keymap, placeholder as cmPlaceholder } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { StreamLanguage, syntaxHighlighting, HighlightStyle } from '@codemirror/language'
import { autocompletion, completionKeymap, acceptCompletion, type CompletionContext, type CompletionResult } from '@codemirror/autocomplete'
import { linter, type Diagnostic } from '@codemirror/lint'
import { tags } from '@lezer/highlight'
import { allExprFunctions, exprFnNames, lintExpr, type ExprDiagnostic } from '@/lib/exprFunctions'

const { t, te } = useI18n()

const props = defineProps<{
  modelValue: string
  placeholder?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const host = ref<HTMLElement | null>(null)
let view: EditorView | null = null

// ── 语法高亮: 手写 stream tokenizer (语法就一行表达式, 不上 Lezer grammar) ──

const exprLanguage = StreamLanguage.define({
  token(stream) {
    if (stream.eatSpace()) return null
    if (stream.match(/^"(?:[^"\\]|\\.)*"?/)) return 'string'
    if (stream.match(/^\d+(\.\d+)?/)) return 'number'
    if (stream.match(/^[a-zA-Z_][a-zA-Z0-9_]*/)) {
      const word = stream.current()
      if (word === 'true' || word === 'false' || word === 'null') return 'atom'
      if (exprFnNames().has(word)) return 'keyword' // 内置函数名
      return 'variableName' // 动态输入引用
    }
    if (stream.match(/^[+\-*/%<>=!&|?:,.]+/)) return 'operator'
    if (stream.match(/^[()]/)) return 'bracket'
    stream.next()
    return null
  },
})

// 配色对齐 pin 类型色 (types.go: string 绿 / number 蓝 / bool 黄系).
const exprHighlight = HighlightStyle.define([
  { tag: tags.string, color: '#4ade80' },
  { tag: tags.number, color: '#60a5fa' },
  { tag: tags.atom, color: '#facc15' },
  { tag: tags.keyword, color: '#c084fc' },
  { tag: tags.variableName, color: '#e2e8f0' },
  { tag: tags.operator, color: '#94a3b8' },
])

// ── 补全: 函数 (sig + i18n 说明, 上屏光标落括号内) + 字面量 ──

function fnDesc(name: string): string {
  const key = `expression.fn.${name}.desc`
  return te(key) ? t(key) : '' // 新函数没配说明 → 空, 不糊 raw key
}

function exprCompletions(ctx: CompletionContext): CompletionResult | null {
  const word = ctx.matchBefore(/[a-zA-Z_][a-zA-Z0-9_]*/)
  if (!word && !ctx.explicit) return null
  const from = word ? word.from : ctx.pos
  return {
    from,
    validFor: /^[a-zA-Z0-9_]*$/,
    options: [
      ...allExprFunctions().map(f => ({
        label: f.name,
        displayLabel: f.sig,
        type: 'function',
        detail: fnDesc(f.name),
        apply: (v: EditorView, _c: unknown, applyFrom: number, applyTo: number) => {
          const insert = `${f.name}()`
          const caret = applyFrom + insert.length - (f.maxArgs === 0 ? 0 : 1)
          v.dispatch({ changes: { from: applyFrom, to: applyTo, insert }, selection: { anchor: caret } })
        },
      })),
      ...['true', 'false', 'null'].map(l => ({ label: l, type: 'keyword' as const })),
    ],
  }
}

// ── lint: 启发式诊断 → 红波浪线 + hover 提示; 框下另留首条红字 ──

function diagMessage(d: ExprDiagnostic): string {
  return te(d.messageKey) ? t(d.messageKey, d.params ?? {}) : d.messageKey
}

function cmLintSource(v: EditorView): Diagnostic[] {
  return lintExpr(v.state.doc.toString(), exprFnNames()).map(d => ({
    from: d.from,
    to: Math.max(d.to, d.from),
    severity: 'error' as const,
    message: diagMessage(d),
  }))
}

const errorText = computed<string>(() => {
  const first = lintExpr(props.modelValue ?? '', exprFnNames())[0]
  return first ? diagMessage(first) : ''
})

// ── EditorView 生命周期 + v-model 双向 ──

const theme = EditorView.theme({
  '&': { backgroundColor: 'transparent', fontSize: '11px' },
  '&.cm-focused': { outline: 'none' },
  '.cm-content': { fontFamily: 'ui-monospace, monospace', minHeight: '3.9em', padding: '4px 0' },
  '.cm-line': { padding: '0 6px' },
  '.cm-tooltip': { fontSize: '11px' },
}, { dark: true })

onMounted(() => {
  view = new EditorView({
    parent: host.value!,
    doc: props.modelValue ?? '',
    extensions: [
      history(),
      exprLanguage,
      syntaxHighlighting(exprHighlight),
      autocompletion({ override: [exprCompletions] }),
      linter(cmLintSource, { delay: 300 }),
      cmPlaceholder(props.placeholder ?? ''),
      theme,
      keymap.of([{ key: 'Tab', run: acceptCompletion }, ...completionKeymap, ...defaultKeymap, ...historyKeymap]),
      EditorView.lineWrapping,
      EditorView.updateListener.of((u) => {
        if (u.docChanged) emit('update:modelValue', u.state.doc.toString())
      }),
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
