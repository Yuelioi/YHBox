<template>
  <div class="relative">
    <UInput
      ref="inputRef"
      :model-value="modelValue"
      :placeholder="placeholder"
      :size="size"
      :class="{
        'ring-1 ring-amber-500/60': typeWarning && !syntaxError,
        'ring-1 ring-rose-500/60': syntaxError,
      }"
      @update:model-value="onInput"
      @focus="onFocus"
      @blur="onBlur"
      @keydown="onKeydown"
    />

    <!-- Autocomplete dropdown -->
    <div
      v-if="opened && suggestions.length > 0"
      class="absolute left-0 right-0 top-full mt-1 z-50 bg-elevated border border-default rounded-md shadow-lg max-h-48 overflow-y-auto"
    >
      <button
        v-for="(s, i) in suggestions"
        :key="s.value"
        type="button"
        class="w-full text-left px-2 py-1 text-xs font-mono flex items-center justify-between hover:bg-primary/10"
        :class="i === highlightedIdx ? 'bg-primary/10 text-primary' : 'text-default'"
        @mousedown.prevent="onPick(s)"
      >
        <span>{{ s.label }}</span>
        <span class="text-[10px] text-dimmed">{{ s.kind }}</span>
      </button>
    </div>

    <!-- Syntax error (red) > Type warning (amber)，最多展一条 -->
    <p v-if="syntaxError" class="text-[10px] text-rose-300/90 mt-0.5">{{ syntaxError }}</p>
    <p v-else-if="typeWarning" class="text-[10px] text-amber-300/80 mt-0.5">{{ typeWarning }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = withDefaults(
  defineProps<{
    modelValue: string
    /** placeholder text，默认按 expectedType 渲染 */
    placeholder?: string
    /** 期望类型 — 影响 hint 与类型猜测 */
    expectedType?: 'number' | 'bool' | 'string' | 'point'
    /** 当前 Container 已声明的变量名（用于 autocomplete） */
    varNames?: string[]
    /** Action.params 名（InvokeAction 的子表单用） */
    paramNames?: string[]
    /** 允许 $sys.* 补全（默认 true） */
    allowSys?: boolean
    size?: 'sm' | 'md' | 'lg' | 'xs'
  }>(),
  {
    allowSys: true,
    size: 'sm',
  },
)

const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const inputRef = ref<any>(null)
const opened = ref(false)
const highlightedIdx = ref(0)
const touched = ref(false)

const placeholder = computed(() => {
  if (props.placeholder) return props.placeholder
  switch (props.expectedType) {
    case 'number':
      return '<number>'
    case 'bool':
      return '<bool>'
    case 'string':
      return '<string>'
    case 'point':
      return '<point>'
  }
  return '<expr>'
})

interface Suggestion {
  label: string
  value: string
  kind: string
}

// 抓最后一个 token 起点；触发 autocomplete 的条件是 token 以 $ 开头或纯字母（fn 名）
function currentToken(s: string): string {
  // 从末尾往前走，直到遇到分隔符
  let i = s.length
  while (i > 0) {
    const c = s[i - 1]
    if (/[a-zA-Z0-9_.$]/.test(c)) i--
    else break
  }
  return s.slice(i)
}

const FN_NAMES = ['abs', 'min', 'max', 'now']

const suggestions = computed<Suggestion[]>(() => {
  const tok = currentToken(props.modelValue)
  if (tok === '') return []
  const out: Suggestion[] = []

  if (tok.startsWith('$')) {
    // $vars.X / $params.Y / $sys.Z
    const segs = tok.split('.')
    const root = segs[0]
    if (root === '$vars' || root === '$') {
      for (const n of props.varNames ?? []) {
        out.push({ label: `$vars.${n}`, value: `$vars.${n}`, kind: 'var' })
      }
    }
    if (root === '$params' || root === '$') {
      for (const n of props.paramNames ?? []) {
        out.push({ label: `$params.${n}`, value: `$params.${n}`, kind: 'param' })
      }
    }
    if ((root === '$sys' || root === '$') && props.allowSys) {
      const sysKeys = [
        '$sys.runId',
        '$sys.iter',
        '$sys.winnerIdx',
        '$sys.lastTemplate.found',
        '$sys.lastTemplate.point.x',
        '$sys.lastTemplate.point.y',
      ]
      for (const k of sysKeys) out.push({ label: k, value: k, kind: 'sys' })
    }
  } else {
    // 函数 / true / false / null
    for (const f of FN_NAMES) {
      if (f.startsWith(tok)) out.push({ label: `${f}(...)`, value: `${f}(`, kind: 'fn' })
    }
    for (const lit of ['true', 'false', 'null']) {
      if (lit.startsWith(tok)) out.push({ label: lit, value: lit, kind: 'lit' })
    }
  }

  // 子串前缀过滤
  return out.filter((s) => s.value.toLowerCase().includes(tok.toLowerCase())).slice(0, 12)
})

watch(suggestions, () => {
  highlightedIdx.value = 0
})

// 语法错误启发式（无 JS parser；只查最常见的坑）。
// 用户触过输入框（touched=true）才显示，避免 SetVar value 等初始空值瞎报错。
const syntaxError = computed<string>(() => {
  if (!touched.value) return ''
  const v = props.modelValue.trim()
  if (v === '') return ''

  // 1) 括号配对
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

  // 2) bareword 检测：单 token 像标识符但不是 true/false/null/函数名（无后跟 '('）
  //    最常见的踩坑：用户想写字符串字面量但忘了双引号。
  if (/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(v)) {
    if (!['true', 'false', 'null'].includes(v)) {
      return t('expression.error.bare_word', { var: v })
    }
  }

  // 3) 表达式以二元运算符结尾
  if (/[+\-*/%<>=!&|?:,]\s*$/.test(v)) return t('expression.error.op_end')

  return ''
})

// 类型 hint（字面量层级猜测）
const typeWarning = computed<string>(() => {
  if (!props.expectedType) return ''
  const v = props.modelValue.trim()
  if (!v) return ''
  if (v.startsWith('$') || v.includes('(') || /[+\-*/%<>=!&|]/.test(v)) return ''
  // 单字面量
  switch (props.expectedType) {
    case 'number':
      if (!/^-?\d+(\.\d+)?$/.test(v)) return t('expression.error.expect_number')
      break
    case 'bool':
      if (v !== 'true' && v !== 'false') return t('expression.error.expect_bool')
      break
    case 'string':
      if (!(v.startsWith('"') && v.endsWith('"'))) return t('expression.error.expect_string')
      break
  }
  return ''
})

function onInput(v: string | number) {
  emit('update:modelValue', String(v ?? ''))
}

function onFocus() {
  opened.value = true
  touched.value = true
}

function onBlur() {
  // 延迟关闭，让 mousedown 选项先触发
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
    highlightedIdx.value =
      (highlightedIdx.value - 1 + suggestions.value.length) % suggestions.value.length
  } else if (e.key === 'Enter' || e.key === 'Tab') {
    e.preventDefault()
    onPick(suggestions.value[highlightedIdx.value])
  } else if (e.key === 'Escape') {
    opened.value = false
  }
}

async function onPick(s: Suggestion) {
  const cur = props.modelValue
  const tok = currentToken(cur)
  const prefix = cur.slice(0, cur.length - tok.length)
  emit('update:modelValue', prefix + s.value)
  opened.value = false
  await nextTick()
  // 重新打开（用户可能想接着补完函数 args 或链式 $vars）
  opened.value = true
}
</script>
