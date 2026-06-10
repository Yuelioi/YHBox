<!-- Expr 表达式编辑器 (widget kind 'expr', PinInput 分发) — 函数补全下拉 + 即时启发式红错.
     权威校验仍在后端 validator (EXPR_PARSE_ERROR / EXPR_UNKNOWN_FUNCTION / EXPR_FN_ARITY 节点红错);
     这里只做打字时的快速反馈, 启发式宁缺勿滥. -->
<template>
  <div class="relative">
    <UTextarea
      ref="textareaRef"
      :model-value="modelValue"
      :rows="3"
      size="sm"
      class="w-full font-mono"
      :color="errorText ? 'error' : undefined"
      :placeholder="placeholder"
      @update:model-value="onInput"
      @focus="onFocus"
      @blur="onBlur"
      @keydown="onKeydown"
      @keyup="refreshCaret"
      @click="refreshCaret"
    />

    <!-- 补全下拉: 签名 + 一行说明 -->
    <div
      v-if="opened && suggestions.length > 0"
      class="absolute left-0 right-0 top-full mt-1 z-50 bg-elevated border border-default rounded-md shadow-lg max-h-48 overflow-y-auto"
    >
      <button
        v-for="(s, i) in suggestions"
        :key="s.insert"
        type="button"
        class="w-full text-left px-2 py-1 text-xs flex items-center justify-between gap-2 hover:bg-primary/10"
        :class="i === highlightedIdx ? 'bg-primary/10 text-primary' : 'text-default'"
        @mousedown.prevent="onPick(s)"
      >
        <span class="font-mono">{{ s.label }}</span>
        <span class="text-[10px] text-dimmed truncate">{{ s.desc }}</span>
      </button>
    </div>

    <p v-if="errorText" class="text-[10px] text-rose-300/90 mt-0.5">{{ errorText }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { EXPR_FUNCTIONS, tokenAtCaret, unknownFnsIn } from '@/lib/exprFunctions'

const { t } = useI18n()

const props = defineProps<{
  modelValue: string
  placeholder?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const textareaRef = ref<any>(null)
const opened = ref(false)
const highlightedIdx = ref(0)
const touched = ref(false)
const caret = ref(0)

function textareaEl(): HTMLTextAreaElement | null {
  return (textareaRef.value?.$el as HTMLElement | undefined)?.querySelector('textarea') ?? null
}

function refreshCaret() {
  const el = textareaEl()
  if (el) caret.value = el.selectionStart ?? props.modelValue.length
}

interface Suggestion {
  label: string
  desc: string
  insert: string
  /** 插入后光标相对 insert 末尾的回退量 (函数补到括号内). */
  caretBack: number
}

const suggestions = computed<Suggestion[]>(() => {
  const { token } = tokenAtCaret(props.modelValue, caret.value)
  if (token === '') return []
  const q = token.toLowerCase()
  const out: Suggestion[] = []
  for (const f of EXPR_FUNCTIONS) {
    if (!f.name.startsWith(q)) continue
    out.push({
      label: f.sig,
      desc: t(`expression.fn.${f.name}.desc`),
      insert: `${f.name}()`,
      caretBack: f.maxArgs === 0 ? 0 : 1, // 有参函数光标落括号内
    })
  }
  for (const lit of ['true', 'false', 'null']) {
    if (lit.startsWith(q)) out.push({ label: lit, desc: '', insert: lit, caretBack: 0 })
  }
  return out.slice(0, 12)
})

// 即时启发式红错 (touched 后才报, 避免初始空值瞎报): 括号/引号/裸词/尾运算符/未知函数.
const errorText = computed<string>(() => {
  if (!touched.value) return ''
  const v = props.modelValue.trim()
  if (v === '') return ''

  let depth = 0
  let inStr = false
  for (let i = 0; i < v.length; i++) {
    const c = v[i]
    if (c === '"' && v[i - 1] !== '\\') inStr = !inStr
    if (inStr) continue
    if (c === '(') depth++
    else if (c === ')') {
      depth--
      if (depth < 0) return t('expression.error.paren_mismatch')
    }
  }
  if (inStr) return t('expression.error.string_unclosed')
  if (depth !== 0) return t('expression.error.paren_missing', { count: Math.abs(depth), char: depth > 0 ? ')' : '(' })

  // 裸词: 单 identifier 且不是字面量 — 多半是想写字符串忘了引号
  if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(v) && !['true', 'false', 'null'].includes(v)) {
    return t('expression.error.bare_word', { var: v })
  }

  if (/[+\-*/%<>=!&|?:,]\s*$/.test(v)) return t('expression.error.op_end')

  const unknown = unknownFnsIn(v)
  if (unknown.length > 0) return t('expression.error.unknown_fn', { name: unknown[0] })

  return ''
})

function onInput(v: string | number) {
  emit('update:modelValue', String(v ?? ''))
  opened.value = true
  highlightedIdx.value = 0
  nextTick(refreshCaret)
}

function onFocus() {
  opened.value = true
  touched.value = true
  refreshCaret()
}

function onBlur() {
  // 延迟关闭, 让选项的 mousedown 先触发
  setTimeout(() => {
    opened.value = false
  }, 150)
}

function onKeydown(e: KeyboardEvent) {
  if (!opened.value || suggestions.value.length === 0) return
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    highlightedIdx.value = (highlightedIdx.value + 1) % suggestions.value.length
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    highlightedIdx.value = (highlightedIdx.value - 1 + suggestions.value.length) % suggestions.value.length
  } else if (e.key === 'Enter' || e.key === 'Tab') {
    e.preventDefault()
    onPick(suggestions.value[highlightedIdx.value])
  } else if (e.key === 'Escape') {
    opened.value = false
  }
}

async function onPick(s: Suggestion) {
  const cur = props.modelValue
  const { start } = tokenAtCaret(cur, caret.value)
  const end = caret.value
  const next = cur.slice(0, start) + s.insert + cur.slice(end)
  emit('update:modelValue', next)
  opened.value = false
  await nextTick()
  const el = textareaEl()
  if (el) {
    const pos = start + s.insert.length - s.caretBack
    el.focus()
    el.setSelectionRange(pos, pos)
    caret.value = pos
  }
}
</script>
