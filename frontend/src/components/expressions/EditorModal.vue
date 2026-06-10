<!-- 表达式/代码放大编辑 modal — 语言无关的壳: 调用方经 extensions() 注入语言
     (Expr 走 exprEditorExtensions, Script 走 scriptEditorExtensions)。
     左编辑器 + 底部状态栏 (lint 首错 + 光标行列); 右侧可搜索参考面板, 点击插入光标处。
     编辑的是 draft, 确认 (按钮或 Ctrl+Enter) 才回写 update:modelValue; 取消/关闭丢弃。 -->
<template>
  <BaseModal
    :open="open"
    :title="title"
    :icon="icon ?? 'i-tabler-code'"
    size="5xl"
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <div class="flex gap-3 h-[68vh] min-h-0">
      <div class="flex-1 min-w-0 flex flex-col gap-1.5">
        <div
          ref="host"
          class="flex-1 min-h-0 bg-elevated/80 border border-default rounded-md overflow-hidden focus-within:border-emerald-500"
        />
        <div class="flex items-center gap-2 text-[10px] shrink-0 px-0.5">
          <span v-if="statusError" class="text-rose-300/90 truncate">{{ statusError }}</span>
          <span class="ml-auto text-muted shrink-0 font-mono">{{ cursorText }}</span>
        </div>
      </div>

      <aside v-if="reference?.length" class="w-72 shrink-0 flex flex-col gap-2 min-h-0">
        <UInput
          v-model="search"
          icon="i-tabler-search"
          size="xs"
          :placeholder="t('inspector.editor_ref_search')"
        />
        <div class="flex-1 min-h-0 overflow-y-auto pr-1">
          <template v-for="group in filteredGroups" :key="group.name">
            <div
              v-if="group.name"
              class="text-[10px] text-muted px-1 pt-2.5 pb-0.5 first:pt-0.5 sticky top-0 bg-default"
            >
              {{ group.name }}
            </div>
            <button
              v-for="it in group.items"
              :key="it.label"
              type="button"
              class="w-full text-left px-2 py-1 rounded hover:bg-elevated/80 focus:bg-elevated/80 focus:outline-none"
              @click="insertItem(it)"
            >
              <div class="text-[11px] font-mono text-highlighted truncate">{{ it.detail ?? it.label }}</div>
              <div v-if="it.desc" class="text-[10px] text-muted truncate">{{ it.desc }}</div>
            </button>
          </template>
          <p v-if="!filteredGroups.length" class="text-[11px] text-muted px-1 py-2">
            {{ t('inspector.editor_ref_empty') }}
          </p>
        </div>
      </aside>
    </div>

    <template #footer>
      <span class="text-[10px] text-muted mr-auto">{{ t('inspector.editor_confirm_hint') }}</span>
      <UButton variant="ghost" color="neutral" @click="emit('update:open', false)">
        {{ t('common.cancel') }}
      </UButton>
      <UButton color="primary" @click="confirm">
        {{ t('common.confirm') }}
      </UButton>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorView, keymap, lineNumbers } from '@codemirror/view'
import { Prec, type Extension } from '@codemirror/state'
import BaseModal from '@/components/common/BaseModal.vue'
import type { InsertItem } from '@/lib/scriptCompletions'

const { t } = useI18n()

/** 参考面板项 — group 是已翻译的展示组名 (函数/节点/输入口), 同组相邻渲染。 */
export interface RefItem extends InsertItem {
  group?: string
}

const props = defineProps<{
  open: boolean
  modelValue: string
  title: string
  icon?: string
  /** 每次打开时调用, 构建语言扩展 (高亮/补全/lint); 不要带 onChange — draft 由本组件管。 */
  extensions: () => Extension[]
  reference?: RefItem[]
  /** 实时状态栏: 返回首条错误文案, 空串 = 无错 (Expr lint 用)。 */
  lintStatus?: (doc: string) => string
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  'update:modelValue': [v: string]
}>()

const host = ref<HTMLElement | null>(null)
const search = ref('')
const draftDoc = ref('')
const cursorText = ref('1:1')
let view: EditorView | null = null

const theme = EditorView.theme({
  '&': { backgroundColor: 'transparent', fontSize: '12px', height: '100%' },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { overflow: 'auto' },
  '.cm-content': { fontFamily: 'ui-monospace, monospace', padding: '6px 0' },
  '.cm-line': { padding: '0 8px' },
  '.cm-gutters': { backgroundColor: 'transparent', border: 'none' },
  '.cm-tooltip': { fontSize: '12px' },
}, { dark: true })

const statusError = computed<string>(() =>
  props.lintStatus ? props.lintStatus(draftDoc.value) : '',
)

const filteredGroups = computed<{ name: string; items: RefItem[] }[]>(() => {
  const q = search.value.trim().toLowerCase()
  const hit = (it: RefItem) =>
    !q ||
    it.label.toLowerCase().includes(q) ||
    (it.detail ?? '').toLowerCase().includes(q) ||
    (it.desc ?? '').toLowerCase().includes(q)
  const groups: { name: string; items: RefItem[] }[] = []
  for (const it of props.reference ?? []) {
    if (!hit(it)) continue
    const name = it.group ?? ''
    const last = groups[groups.length - 1]
    if (last && last.name === name) last.items.push(it)
    else groups.push({ name, items: [it] })
  }
  return groups
})

// modal 内容随 open 挂/卸 (UModal 懒渲染) — editor 跟着 open 建/毁, 开时灌当前值当 draft。
watch(() => props.open, async (open) => {
  if (open) {
    await nextTick()
    if (!host.value) return
    search.value = ''
    draftDoc.value = props.modelValue ?? ''
    view = new EditorView({
      parent: host.value,
      doc: props.modelValue ?? '',
      extensions: [
        ...props.extensions(),
        lineNumbers(),
        theme,
        Prec.high(keymap.of([{ key: 'Mod-Enter', run: () => { confirm(); return true } }])),
        EditorView.updateListener.of((u) => {
          if (u.docChanged) draftDoc.value = u.state.doc.toString()
          if (u.selectionSet || u.docChanged) {
            const head = u.state.selection.main.head
            const line = u.state.doc.lineAt(head)
            cursorText.value = `${line.number}:${head - line.from + 1}`
          }
        }),
      ],
    })
    view.focus()
  } else {
    view?.destroy()
    view = null
  }
})

// 参考面板点击 → 插到光标处 (替换选区), 光标按 caretBack 落进括号/引号内。
function insertItem(it: RefItem) {
  if (!view) return
  const { from, to } = view.state.selection.main
  view.dispatch({
    changes: { from, to, insert: it.insert },
    selection: { anchor: from + it.insert.length - it.caretBack },
  })
  view.focus()
}

function confirm() {
  if (view) emit('update:modelValue', view.state.doc.toString())
  emit('update:open', false)
}

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})
</script>
