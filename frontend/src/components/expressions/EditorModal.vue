<!-- 表达式/代码放大编辑 modal — 语言无关的壳: 调用方经 extensions() 注入语言
     (Expr 走 exprEditorExtensions, Script 走 scriptEditorExtensions)。
     工具栏 (撤销/重做/注释/查找替换/片段) + 左编辑器 + 底部状态栏 (lint 首错 + 光标行列);
     右侧可搜索参考面板: 行点击展开用法 (说明/参数/示例), 行尾按钮插入光标处。
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
        <div class="flex items-center gap-0.5 shrink-0">
          <UButton icon="i-tabler-arrow-back-up" variant="ghost" color="neutral" size="xs"
            :title="t('inspector.editor_undo')" @click="run(undo)" />
          <UButton icon="i-tabler-arrow-forward-up" variant="ghost" color="neutral" size="xs"
            :title="t('inspector.editor_redo')" @click="run(redo)" />
          <UButton v-if="commentable" variant="ghost" color="neutral" size="xs"
            class="font-mono" :title="t('inspector.editor_comment')" @click="run(toggleComment)">//</UButton>
          <UButton icon="i-tabler-list-search" variant="ghost" color="neutral" size="xs"
            :title="t('inspector.editor_search')" @click="run(openSearchPanel)" />
          <UDropdownMenu v-if="snippets?.length" :items="snippetMenuItems">
            <UButton icon="i-tabler-template" variant="ghost" color="neutral" size="xs"
              trailing-icon="i-tabler-chevron-down" :title="t('inspector.editor_snippets')">
              {{ t('inspector.editor_snippets') }}
            </UButton>
          </UDropdownMenu>
        </div>
        <div
          ref="host"
          class="flex-1 min-h-0 bg-elevated/80 border border-default rounded-md overflow-hidden focus-within:border-emerald-500"
        />
        <div class="flex items-center gap-2 text-[10px] shrink-0 px-0.5">
          <span v-if="statusError" class="text-rose-300/90 truncate">{{ statusError }}</span>
          <span class="ml-auto text-muted shrink-0 font-mono">{{ cursorText }}</span>
        </div>
      </div>

      <aside v-if="reference?.length" class="w-80 shrink-0 flex flex-col gap-2 min-h-0">
        <div class="flex items-center gap-1.5 shrink-0">
          <UInput
            v-model="search"
            icon="i-tabler-search"
            size="xs"
            class="flex-1 min-w-0"
            :placeholder="t('inspector.editor_ref_search')"
          />
          <slot name="panel-actions" />
        </div>
        <div class="flex-1 min-h-0 overflow-y-auto pr-1">
          <template v-for="group in filteredGroups" :key="group.name">
            <div
              v-if="group.name"
              class="text-[10px] text-muted px-1 pt-2.5 pb-0.5 first:pt-0.5 sticky top-0 bg-default z-10"
            >
              {{ group.name }}
            </div>
            <div v-for="it in group.items" :key="it.label" class="rounded" :class="isExpanded(it) ? 'bg-elevated/60' : ''">
              <div class="flex items-center group/row">
                <button
                  type="button"
                  class="flex-1 min-w-0 text-left px-2 py-1 rounded hover:bg-elevated/80 focus:bg-elevated/80 focus:outline-none"
                  @click="onRowClick(it)"
                >
                  <div class="text-[12px] font-mono text-highlighted truncate">{{ it.detail ?? it.label }}</div>
                  <div v-if="it.desc" class="text-[10px] text-muted truncate">{{ it.desc }}</div>
                </button>
                <UButton
                  icon="i-tabler-corner-down-left"
                  variant="ghost"
                  color="neutral"
                  size="xs"
                  class="shrink-0 opacity-0 group-hover/row:opacity-70 hover:!opacity-100"
                  :title="t('inspector.editor_insert')"
                  @click="insertItem(it)"
                />
              </div>
              <div v-if="isExpanded(it)" class="px-2 pb-2 space-y-1.5">
                <p v-if="it.docs" class="text-[11px] text-toned leading-snug whitespace-pre-line">{{ it.docs }}</p>
                <div v-if="it.params?.length" class="space-y-0.5">
                  <div class="text-[10px] text-muted">{{ t('inspector.editor_params') }}</div>
                  <div
                    v-for="p in it.params"
                    :key="p.name"
                    class="flex items-baseline gap-2 text-[11px] leading-snug"
                  >
                    <span class="font-mono text-highlighted shrink-0">{{ p.name }}</span>
                    <span class="font-mono text-[10px] text-dimmed shrink-0">{{ p.type }}{{ p.required ? ' *' : '' }}</span>
                    <span class="text-muted truncate">{{ p.label }}</span>
                  </div>
                </div>
                <p v-if="it.example" class="text-[10px] text-dimmed leading-snug italic whitespace-pre-line">{{ it.example }}</p>
              </div>
            </div>
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
import { undo, redo, toggleComment } from '@codemirror/commands'
import { search as cmSearch, searchKeymap, openSearchPanel } from '@codemirror/search'
import BaseModal from '@/components/common/BaseModal.vue'
import type { InsertItem } from '@/lib/scriptCompletions'
import { zhSearchPhrases } from '@/lib/editorTheme'

const { t, locale } = useI18n()

/** 参考面板项 — group 是已翻译的展示组名, 同组相邻渲染;
    docs/params/example 是展开详情 (都缺省 = 行不可展开, 点击即插入)。 */
export interface RefItem extends InsertItem {
  group?: string
  /** 展开详情: 用法说明 (节点 description / 函数长说明)。 */
  docs?: string
  /** 展开详情: 参数表 (节点 pin: 名/人话 label/类型/必填)。 */
  params?: { name: string; label: string; type: string; required?: boolean }[]
  /** 展开详情: 示例。 */
  example?: string
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
  /** 工具栏「注释」按钮 (需语言有注释语法 — Script JS 开, Expr 关)。 */
  commentable?: boolean
  /** 工具栏「片段」下拉 (if/for/while/try 之类模板)。 */
  snippets?: InsertItem[]
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  'update:modelValue': [v: string]
}>()

const host = ref<HTMLElement | null>(null)
const search = ref('')
const draftDoc = ref('')
const cursorText = ref('1:1')
const expandedKeys = ref<Set<string>>(new Set())
let view: EditorView | null = null

const theme = EditorView.theme({
  '&': { backgroundColor: 'transparent', fontSize: '12px', height: '100%' },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { overflow: 'auto' },
  '.cm-content': { fontFamily: 'ui-monospace, monospace', padding: '6px 0' },
  '.cm-line': { padding: '0 8px' },
  '.cm-gutters': { backgroundColor: 'transparent', border: 'none' },
  '.cm-tooltip': { fontSize: '12px' },
  '.cm-panels': { backgroundColor: 'transparent', border: 'none' },
  '.cm-panel.cm-search': { fontSize: '11px', padding: '6px 8px' },
}, { dark: true })

const statusError = computed<string>(() =>
  props.lintStatus ? props.lintStatus(draftDoc.value) : '',
)

const snippetMenuItems = computed(() =>
  (props.snippets ?? []).map((it) => [{ label: it.label, onSelect: () => insertItem(it) }]),
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

function refKey(it: RefItem): string {
  return `${it.group ?? ''}/${it.label}`
}
function isExpanded(it: RefItem): boolean {
  return expandedKeys.value.has(refKey(it))
}
function expandable(it: RefItem): boolean {
  return !!(it.docs || it.params?.length || it.example)
}
// 行点击: 有详情 → 展开/收起; 无详情 → 直接插入 (行尾按钮永远是插入)。
function onRowClick(it: RefItem) {
  if (!expandable(it)) {
    insertItem(it)
    return
  }
  const key = refKey(it)
  const next = new Set(expandedKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  expandedKeys.value = next
}

// modal 内容随 open 挂/卸 (UModal 懒渲染) — editor 跟着 open 建/毁, 开时灌当前值当 draft。
watch(() => props.open, async (open) => {
  if (open) {
    await nextTick()
    if (!host.value) return
    search.value = ''
    expandedKeys.value = new Set()
    draftDoc.value = props.modelValue ?? ''
    view = new EditorView({
      parent: host.value,
      doc: props.modelValue ?? '',
      extensions: [
        ...props.extensions(),
        lineNumbers(),
        searchExt(),
        keymap.of(searchKeymap),
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

function searchExt(): Extension {
  const base = cmSearch({ top: true })
  return locale.value.startsWith('zh') ? [base, zhSearchPhrases] : base
}

function run(cmd: (view: EditorView) => boolean) {
  if (!view) return
  cmd(view)
  view.focus()
}

// 参考面板/片段/外部 (新建变量) → 插到光标处 (替换选区), 光标按 caretBack 落进括号/引号内。
function insertItem(it: InsertItem) {
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

defineExpose({ insert: insertItem })
</script>
