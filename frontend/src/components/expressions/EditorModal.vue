<!-- 表达式/代码放大编辑 modal — 语言无关的壳: 调用方经 extensions() 注入语言
     (Expr 走 exprEditorExtensions, Script 走 scriptEditorExtensions, 都带 modal:true 档)。
     工具栏分组 (撤销重做 | 注释/查找 | 片段/扩展 | 整理缩进/折叠) + 左编辑器
     + 底部状态栏 (语法状态可点跳错 + 统计 + 光标 + 语言标签); 右侧可搜索参考面板。
     编辑的是 draft, 确认 (按钮或 Ctrl+Enter) 才回写 update:modelValue; 取消/关闭丢弃。 -->
<template>
  <BaseModal
    :open="open"
    :title="title"
    :icon="icon ?? 'i-tabler-code'"
    size="7xl"
    :tall="maximized"
    :content-class="maximized ? 'sm:max-w-[96vw]' : undefined"
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <template #header-extra>
      <UButton
        :icon="maximized ? 'i-tabler-minimize' : 'i-tabler-maximize'"
        variant="ghost"
        color="neutral"
        size="xs"
        :title="maximized ? t('inspector.editor_restore') : t('inspector.editor_maximize')"
        @click="maximized = !maximized"
      />
    </template>
    <div class="flex gap-3 min-h-0" :class="maximized ? 'h-full' : 'h-[70vh]'">
      <div class="flex-1 min-w-0 flex flex-col gap-1.5">
        <div class="flex items-center shrink-0">
          <div class="flex items-center gap-0.5">
            <UButton icon="i-tabler-arrow-back-up" variant="ghost" color="neutral" size="xs"
              :title="t('inspector.editor_undo')" @click="run(undo)" />
            <UButton icon="i-tabler-arrow-forward-up" variant="ghost" color="neutral" size="xs"
              :title="t('inspector.editor_redo')" @click="run(redo)" />
          </div>
          <span class="h-4 border-l border-default mx-1.5" />
          <div class="flex items-center gap-0.5">
            <UButton v-if="commentable" variant="ghost" color="neutral" size="xs"
              class="font-mono" :title="t('inspector.editor_comment')" @click="run(toggleComment)">//</UButton>
            <UButton icon="i-tabler-list-search" variant="ghost" color="neutral" size="xs"
              :title="t('inspector.editor_search')" @click="run(openSearchPanel)" />
          </div>
          <template v-if="snippetLang || $slots['toolbar-extra']">
            <span class="h-4 border-l border-default mx-1.5" />
            <div class="flex items-center gap-0.5">
              <UButton v-if="snippetLang" icon="i-tabler-template" variant="ghost" color="neutral" size="xs"
                :title="t('inspector.editor_snippets_tip')" @click="openSnippetManager">
                {{ t('inspector.editor_snippets') }}
              </UButton>
              <slot name="toolbar-extra" />
            </div>
          </template>
          <div class="ml-auto flex items-center gap-0.5">
            <UButton v-if="reference?.length" icon="i-tabler-book" variant="ghost"
              :color="refDrawerOpen ? 'primary' : 'neutral'" size="xs"
              :title="t('inspector.editor_ref_toggle')" @click="refDrawerOpen = !refDrawerOpen" />
            <UButton icon="i-tabler-indent-increase" variant="ghost" color="neutral" size="xs"
              :title="t('inspector.editor_indent_tidy')" @click="reindentAll" />
            <template v-if="foldable">
              <UButton icon="i-tabler-fold" variant="ghost" color="neutral" size="xs"
                :title="t('inspector.editor_fold_all')" @click="run(foldAll)" />
              <UButton icon="i-tabler-fold-down" variant="ghost" color="neutral" size="xs"
                :title="t('inspector.editor_unfold_all')" @click="run(unfoldAll)" />
            </template>
          </div>
        </div>
        <div
          ref="host"
          class="flex-1 min-h-0 border border-default rounded-md overflow-hidden focus-within:border-primary/60"
        />
        <div class="flex items-center gap-3 text-[11px] shrink-0 px-0.5">
          <button
            v-if="statusError"
            type="button"
            class="flex items-center gap-1 text-error truncate cursor-pointer hover:underline"
            :title="t('inspector.editor_goto_error')"
            @click="jumpToError"
          >
            <UIcon name="i-tabler-alert-circle" class="size-3.5 shrink-0" />
            <span class="truncate">{{ statusError.message }}</span>
          </button>
          <span v-else class="flex items-center gap-1 text-success/80">
            <UIcon name="i-tabler-check" class="size-3.5 shrink-0" />
            {{ t('inspector.editor_status_ok') }}
          </span>
          <span class="ml-auto text-muted shrink-0 font-mono">{{ cursorText }}</span>
          <span class="text-muted shrink-0">{{ statsText }}</span>
          <span v-if="langLabel" class="text-dimmed shrink-0">{{ langLabel }}</span>
        </div>
      </div>

      <Transition name="ref-drawer">
      <aside
        v-if="reference?.length && refDrawerOpen"
        class="w-80 shrink-0 flex flex-col gap-2 min-h-0 border-l border-default pl-3"
      >
        <div class="flex items-center justify-between shrink-0">
          <span class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted">
            <UIcon name="i-tabler-book" class="size-3.5" />
            {{ t('inspector.editor_ref_toggle') }}
          </span>
          <UButton icon="i-tabler-x" variant="ghost" color="neutral" size="xs" @click="refDrawerOpen = false" />
        </div>
        <UInput
          v-model="search"
          icon="i-tabler-search"
          size="xs"
          class="shrink-0"
          :placeholder="t('inspector.editor_ref_search')"
        />
        <div class="flex-1 min-h-0 overflow-y-auto pr-1">
          <template v-for="group in filteredGroups" :key="group.name">
            <div
              v-if="group.name"
              class="flex items-center gap-1.5 px-1 pt-4 pb-1.5 first:pt-1 sticky top-0 bg-default z-10"
              :class="group.cls || 'text-muted'"
            >
              <span class="size-1.5 rounded-full bg-current shrink-0" />
              <span class="text-[10px] font-semibold uppercase tracking-wider text-toned">{{ group.name }}</span>
              <span class="h-px flex-1 bg-accented/60 ml-1" />
              <span class="text-[10px] font-normal text-dimmed tabular-nums">{{ group.items.length }}</span>
            </div>
            <div
              v-for="it in group.items"
              :key="it.label"
              class="rounded-md"
              :class="isExpanded(it) ? 'bg-elevated/50 ring-1 ring-default my-1' : ''"
            >
              <div class="flex items-center group/row">
                <button
                  type="button"
                  class="flex-1 min-w-0 text-left px-2 py-1.5 rounded-md hover:bg-elevated/60 focus:bg-elevated/60 focus:outline-none"
                  @click="onRowClick(it)"
                >
                  <div class="text-[12px] font-mono truncate leading-snug">
                    <span class="text-highlighted">{{ sigName(it) }}</span><span class="text-dimmed">{{ sigArgs(it) }}</span>
                  </div>
                  <div v-if="it.desc" class="text-[11px] text-muted truncate mt-0.5">{{ it.desc }}</div>
                </button>
                <UIcon
                  v-if="expandable(it)"
                  name="i-tabler-chevron-right"
                  class="size-3 shrink-0 text-dimmed transition-transform duration-150 mr-0.5"
                  :class="isExpanded(it) ? 'rotate-90' : ''"
                />
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
              <div v-if="isExpanded(it)" class="mx-2 pb-2 pt-1.5 space-y-1.5 border-t border-default/60">
                <p v-if="it.docs" class="text-[11px] text-toned leading-snug whitespace-pre-line">{{ it.docs }}</p>
                <div v-if="it.params?.length" class="space-y-0.5">
                  <div class="text-[10px] font-semibold uppercase tracking-wider text-dimmed">{{ t('inspector.editor_params') }}</div>
                  <div
                    v-for="p in it.params"
                    :key="p.name"
                    class="flex items-baseline gap-2 text-[11px] leading-snug"
                  >
                    <span class="font-mono text-highlighted shrink-0">{{ p.name }}</span>
                    <span class="font-mono text-[10px] text-info/80 shrink-0">{{ p.type }}<span v-if="p.required" class="text-error">*</span></span>
                    <span class="text-muted truncate">{{ p.label }}</span>
                    <span v-if="p.options?.length" class="font-mono text-[10px] text-dimmed shrink-0">{{ p.options.join(' | ') }}</span>
                  </div>
                </div>
                <p v-if="it.example" class="text-[10px] text-dimmed leading-snug italic whitespace-pre-line">{{ it.example }}</p>
                <UButton
                  size="xs"
                  variant="soft"
                  color="primary"
                  icon="i-tabler-corner-down-left"
                  @click="insertItem(it)"
                >{{ t('inspector.editor_insert') }}</UButton>
              </div>
            </div>
          </template>
          <div v-if="!filteredGroups.length" class="flex flex-col items-center gap-1.5 py-8 text-muted">
            <UIcon name="i-tabler-search-off" class="size-5 opacity-60" />
            <p class="text-[11px]">{{ t('inspector.editor_ref_empty') }}</p>
          </div>
        </div>
      </aside>
      </Transition>
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

  <SnippetManagerModal
    v-if="snippetLang"
    v-model:open="managerOpen"
    :lang="snippetLang"
    :initial-body="managerCreate ? managerBody : undefined"
  />
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorView, keymap } from '@codemirror/view'
import { Prec, type Extension } from '@codemirror/state'
import { undo, redo, toggleComment, indentSelection } from '@codemirror/commands'
import { foldAll, unfoldAll } from '@codemirror/language'
import { search as cmSearch, searchKeymap, openSearchPanel } from '@codemirror/search'
import { snippet, type Completion } from '@codemirror/autocomplete'
import BaseModal from '@/components/common/BaseModal.vue'
import SnippetManagerModal from './SnippetManagerModal.vue'
import type { InsertItem } from '@/lib/scriptCompletions'
import { searchPanelTheme, zhSearchPhrases } from '@/lib/editorTheme'
import type { CodeSnippetLang } from '@/stores/codeSnippets'

const { t, locale } = useI18n()

/** 参考面板项 — group 是已翻译的展示组名, 同组相邻渲染;
    docs/params/example 是展开详情 (都缺省 = 行不可展开, 点击即插入)。 */
export interface RefItem extends InsertItem {
  group?: string
  /** 组标题的 tailwind 文字色 class (节点分类配色, 对齐画布) — 缺省灰。 */
  groupClass?: string
  /** 展开详情: 用法说明 (节点 description / 函数长说明)。 */
  docs?: string
  /** 展开详情: 参数表 (节点 pin: 名/人话 label/类型/必填)。 */
  params?: { name: string; label: string; type: string; required?: boolean; options?: string[] }[]
  /** 展开详情: 示例。 */
  example?: string
}

const props = defineProps<{
  open: boolean
  modelValue: string
  title: string
  icon?: string
  /** 每次打开时调用, 构建语言扩展 (高亮/补全/lint, modal 档); 不要带 onChange — draft 由本组件管。 */
  extensions: () => Extension[]
  reference?: RefItem[]
  /** 实时状态栏: 首条问题 (文案+位置, 点击跳转), null = 无问题。 */
  lintFirst?: (doc: string) => { message: string; from: number } | null
  /** 工具栏「注释」按钮 (需语言有注释语法 — Script JS 开, Expr 关)。 */
  commentable?: boolean
  /** 工具栏「折叠」按钮组 (语言支持折叠时开 — Script 开, Expr 关)。 */
  foldable?: boolean
  /** 传了就有「片段」下拉 (codeSnippets store 按语言取用户片段 + 新建/管理入口)。 */
  snippetLang?: CodeSnippetLang
  /** 状态栏右侧语言标签 (JavaScript / 表达式)。 */
  langLabel?: string
}>()

const emit = defineEmits<{
  'update:open': [v: boolean]
  'update:modelValue': [v: string]
}>()

const host = ref<HTMLElement | null>(null)
const search = ref('')
const draftDoc = ref('')
const cursorText = ref('1:1')
const maximized = ref(true)
const expandedKeys = ref<Set<string>>(new Set())
let view: EditorView | null = null

// 参考面板抽屉开关 — 默认收起, 状态记本地, 工具栏按钮 + F1 切换。
const REF_DRAWER_KEY = 'yotta.editor.refDrawer'
function loadRefDrawer(): boolean {
  try { return localStorage.getItem(REF_DRAWER_KEY) === '1' } catch { return false }
}
const refDrawerOpen = ref(loadRefDrawer())
watch(refDrawerOpen, (v) => {
  try { localStorage.setItem(REF_DRAWER_KEY, v ? '1' : '0') } catch { /* localStorage 不可用 → 静默 */ }
})

const statusError = computed<{ message: string; from: number } | null>(() =>
  props.lintFirst ? props.lintFirst(draftDoc.value) : null,
)

const statsText = computed<string>(() => {
  const doc = draftDoc.value
  const lines = (doc.match(/\n/g)?.length ?? 0) + 1
  return t('inspector.editor_status_stats', { lines, chars: doc.length })
})

// 用户片段管理入口: 工具栏单图标 — 带非空选区点开 = 直接进新建表单预填 (选区入库);
// 无选区 = 进列表。插入不走这里, 走编辑器内 prefix 补全。
const managerOpen = ref(false)
const managerCreate = ref(false)
const managerBody = ref('')

function openSnippetManager() {
  const sel = view?.state.selection.main
  const hasSel = !!sel && !sel.empty
  managerCreate.value = hasSel
  managerBody.value = hasSel ? view!.state.sliceDoc(sel!.from, sel!.to) : ''
  managerOpen.value = true
}

const filteredGroups = computed<{ name: string; cls?: string; items: RefItem[] }[]>(() => {
  const q = search.value.trim().toLowerCase()
  const hit = (it: RefItem) =>
    !q ||
    it.label.toLowerCase().includes(q) ||
    (it.detail ?? '').toLowerCase().includes(q) ||
    (it.desc ?? '').toLowerCase().includes(q)
  const groups: { name: string; cls?: string; items: RefItem[] }[] = []
  for (const it of props.reference ?? []) {
    if (!hit(it)) continue
    const name = it.group ?? ''
    const last = groups[groups.length - 1]
    if (last && last.name === name) last.items.push(it)
    else groups.push({ name, cls: it.groupClass, items: [it] })
  }
  return groups
})

// 签名拆两段渲染: 函数名亮色, '(' 起的参数串暗色 — 长签名截断时名字仍可扫读。
function sigName(it: RefItem): string {
  const s = it.detail ?? it.label
  const i = s.indexOf('(')
  return i === -1 ? s : s.slice(0, i)
}
function sigArgs(it: RefItem): string {
  const s = it.detail ?? it.label
  const i = s.indexOf('(')
  return i === -1 ? '' : s.slice(i)
}

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
        searchExt(),
        searchPanelTheme,
        keymap.of(searchKeymap),
        Prec.high(keymap.of([{ key: 'Mod-Enter', run: () => { confirm(); return true } }])),
        Prec.high(keymap.of([{ key: 'F1', run: () => { refDrawerOpen.value = !refDrawerOpen.value; return true } }])),
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

// 整理缩进 = 全选跑语言缩进规则, 再把光标放回原处 (缩进变化导致的少量偏移可接受)。
function reindentAll() {
  if (!view) return
  const head = view.state.selection.main.head
  view.dispatch({ selection: { anchor: 0, head: view.state.doc.length } })
  indentSelection(view)
  view.dispatch({ selection: { anchor: Math.min(head, view.state.doc.length) } })
  view.focus()
}

function jumpToError() {
  const err = statusError.value
  if (!view || !err) return
  view.dispatch({
    selection: { anchor: Math.min(err.from, view.state.doc.length) },
    scrollIntoView: true,
  })
  view.focus()
}

// 参考面板/片段/外部 (新建变量) → 插到光标处 (替换选区)。
// 带 snippet 模板的项走占位插入 (Tab 跳格填参数); 否则按 caretBack 落光标。
function insertItem(it: InsertItem) {
  if (!view) return
  const { from, to } = view.state.selection.main
  if (it.snippet) {
    snippet(it.snippet)(view, null as unknown as Completion, from, to)
  } else {
    view.dispatch({
      changes: { from, to, insert: it.insert },
      selection: { anchor: from + it.insert.length - it.caretBack },
    })
  }
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

<style scoped>
.ref-drawer-enter-active,
.ref-drawer-leave-active {
  transition: all 0.15s ease;
}
.ref-drawer-enter-from,
.ref-drawer-leave-to {
  transform: translateX(8px);
  opacity: 0;
}
</style>
