// 编辑器共享外观与基础扩展 (ExprInput / CodeInput / EditorModal 三处统一引入):
// 语法 token 色 = VSCode Dark+ 移植 (代码内容色, 独立于 app 主题);
// chrome (编辑面/gutter/浮层/查找面板) = NuxtUI semantic CSS 变量 (--ui-*),
// 跟 app 同一套灰阶/主色 — 不许再写死 hex, 否则 modal/卡片里出现"第二种灰"。
// 另含编辑手感基础件 (自动配对/括号高亮/Tab 缩进/选中词同款/多选区) + 查找面板中文 phrases。
// 例外 token: $变量 保持橙色徽标 (与画布变量认知一致, 不随 Dark+)。
import {
  EditorView,
  keymap,
  drawSelection,
  lineNumbers,
  highlightActiveLine,
  highlightActiveLineGutter,
  tooltips,
} from '@codemirror/view'
import { EditorState, type Extension } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import {
  HighlightStyle,
  bracketMatching,
  indentOnInput,
  indentUnit,
  syntaxHighlighting,
} from '@codemirror/language'
import {
  acceptCompletion,
  closeBrackets,
  closeBracketsKeymap,
  completionKeymap,
} from '@codemirror/autocomplete'
import { highlightSelectionMatches, selectNextOccurrence } from '@codemirror/search'
import { lintGutter } from '@codemirror/lint'
import { tags } from '@lezer/highlight'
import { i18n } from '@/i18n'

export const EDITOR_FONT = `'JetBrains Mono Variable', ui-monospace, Consolas, 'Courier New', monospace`

// ── 语法配色: VSCode Dark+ (Expr / Script 共用一份) ──

export const editorHighlightStyle = HighlightStyle.define([
  { tag: tags.controlKeyword, color: '#c586c0' },
  { tag: tags.keyword, color: '#569cd6' },
  { tag: [tags.bool, tags.null, tags.atom, tags.self], color: '#569cd6' },
  { tag: [tags.string, tags.special(tags.string)], color: '#ce9178' },
  { tag: tags.number, color: '#b5cea8' },
  {
    tag: [tags.comment, tags.lineComment, tags.blockComment],
    color: '#6a9955',
    fontStyle: 'italic',
  },
  { tag: [tags.function(tags.variableName), tags.function(tags.propertyName)], color: '#dcdcaa' },
  { tag: [tags.variableName, tags.definition(tags.variableName)], color: '#9cdcfe' },
  { tag: [tags.propertyName, tags.definition(tags.propertyName)], color: '#9cdcfe' },
  { tag: tags.special(tags.variableName), color: '#fb923c' }, // $变量引用 (橙, 产品特有)
  { tag: [tags.typeName, tags.className, tags.namespace], color: '#4ec9b0' },
  { tag: tags.regexp, color: '#d16969' },
  { tag: tags.escape, color: '#d7ba7d' },
  { tag: tags.operator, color: '#d4d4d4' },
  { tag: [tags.punctuation, tags.bracket, tags.separator], color: '#d4d4d4' },
  { tag: tags.labelName, color: '#c8c8c8' },
  { tag: tags.invalid, color: '#f44747' },
])

// ── chrome: 编辑面/光标/选区/当前行/括号/gutter/tooltip — 全走 --ui-* semantic 变量;
//    选区/选中词用 primary 的 color-mix, 与全局 ::selection 同源 ──

const chromeTheme = EditorView.theme(
  {
    '&': { backgroundColor: 'var(--ui-bg)', color: 'var(--ui-text)' },
    '&.cm-focused': { outline: 'none' },
    // 连字关掉: === 渲成三横长等号对脚本新手是误导, VSCode 默认也不开
    '.cm-scroller': { fontFamily: EDITOR_FONT, overflow: 'auto', fontVariantLigatures: 'none' },
    '.cm-content': { caretColor: 'var(--ui-text-toned)' },
    '.cm-cursor, .cm-dropCursor': {
      borderLeftColor: 'var(--ui-text-toned)',
      borderLeftWidth: '2px',
    },
    '.cm-selectionBackground, .cm-content ::selection': {
      backgroundColor: 'color-mix(in oklab, var(--ui-primary) 20%, transparent)',
    },
    '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground': {
      backgroundColor: 'color-mix(in oklab, var(--ui-primary) 32%, transparent)',
    },
    '.cm-selectionMatch': {
      backgroundColor: 'color-mix(in oklab, var(--ui-primary) 12%, transparent)',
    },
    '.cm-activeLine': { backgroundColor: '#ffffff0a' },
    '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--ui-text-toned)' },
    '.cm-gutters': {
      backgroundColor: 'var(--ui-bg)',
      color: 'var(--ui-text-dimmed)',
      border: 'none',
    },
    '.cm-lineNumbers .cm-gutterElement': { paddingLeft: '14px', paddingRight: '6px' },
    '&.cm-focused .cm-matchingBracket': {
      backgroundColor: 'color-mix(in oklab, var(--ui-primary) 12%, transparent)',
      outline: '1px solid var(--ui-text-dimmed)',
    },
    '&.cm-focused .cm-nonmatchingBracket': {
      outline: '1px solid color-mix(in oklab, var(--ui-error) 60%, transparent)',
    },
    '.cm-foldGutter .cm-gutterElement': { color: 'var(--ui-text-dimmed)', cursor: 'pointer' },
    '.cm-foldPlaceholder': {
      backgroundColor: 'var(--ui-bg-elevated)',
      border: '1px solid var(--ui-border-accented)',
      color: 'var(--ui-text)',
      borderRadius: '3px',
      padding: '0 6px',
      margin: '0 2px',
    },
    '.cm-gutter-lint': { width: '8px' },
    '.cm-gutter-lint .cm-gutterElement': { padding: '2px 0 0 2px' },
    '.cm-tooltip': {
      backgroundColor: 'var(--ui-bg-elevated)',
      border: '1px solid var(--ui-border-accented)',
      color: 'var(--ui-text)',
      // 浮层挂到 document.body (见 baseEditorExtensions 的 tooltips parent) — z-index 必须盖过
      // 放大编辑 modal (Nuxt UI z-[100]), 否则签名/hover 浮层会渲染到模态下方看不见。
      zIndex: '1000',
    },
    '.cm-placeholder': { color: 'var(--ui-text-dimmed)' },
    // $变量徽标: theme 规则带 scope 前缀, 特异性盖过 HighlightStyle 的 token 色
    '.cm-yh-dollar': {
      color: '#fb923c',
      backgroundColor: 'rgba(251,146,60,.09)',
      borderRadius: '3px',
    },
    '.cm-tooltip .cm-yh-doc': {
      padding: '6px 10px',
      maxWidth: '30em',
      fontSize: '12px',
      lineHeight: '1.5',
    },
    '.cm-yh-doc-sig': { fontFamily: EDITOR_FONT, color: '#dcdcaa' },
    '.cm-yh-doc-sig-active': { color: '#dcdcaa', fontWeight: 'bold' },
    '.cm-yh-doc-desc': { color: 'var(--ui-text-muted)', marginTop: '2px' },
    '.cm-yh-doc-param': { display: 'flex', gap: '8px', marginTop: '2px', fontSize: '11px' },
    '.cm-yh-doc-param-name': { fontFamily: EDITOR_FONT, color: '#9cdcfe', minWidth: '6em' },
    '.cm-yh-doc-param-type': { fontFamily: EDITOR_FONT, color: '#4ec9b0' },
    '.cm-yh-doc-param-label': { color: 'var(--ui-text-muted)' },
    '.cm-yh-doc-param-enum': { fontFamily: EDITOR_FONT, color: '#d7ba7d', opacity: '0.85' },
    '.cm-yh-doc-snippet-body': {
      fontFamily: EDITOR_FONT,
      whiteSpace: 'pre-wrap',
      marginTop: '4px',
      fontSize: '11px',
      color: 'var(--ui-text-muted)',
    },
  },
  { dark: true },
)

// 字号/行距/留白分两档: 放大编辑 (modal) 13px 宽松; 卡片内小框 12px 紧凑。
const modalSizeTheme = EditorView.theme({
  '&': { fontSize: '13px', height: '100%' },
  '.cm-scroller': { lineHeight: '1.6' },
  '.cm-content': { padding: '10px 0' },
  '.cm-line': { padding: '0 14px' },
  '.cm-tooltip': { fontSize: '12px' },
})

function smallSizeTheme(minHeight: string): Extension {
  return EditorView.theme({
    '&': { fontSize: '12px' },
    '.cm-scroller': { lineHeight: '1.55' },
    '.cm-content': { minHeight, padding: '8px 0' },
    '.cm-line': { padding: '0 8px' },
    '.cm-tooltip': { fontSize: '12px' },
  })
}

// ── 补全下拉 (VSCode suggest widget 观感) ──

export const completionTooltipTheme: Extension = EditorView.theme({
  '.cm-tooltip.cm-tooltip-autocomplete': {
    backgroundColor: 'var(--ui-bg-elevated)',
    border: '1px solid var(--ui-border-accented)',
    borderRadius: '6px',
    boxShadow: '0 6px 16px rgba(0,0,0,.45)',
    overflow: 'hidden',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul': {
    fontFamily: EDITOR_FONT,
    fontSize: '12px',
    maxHeight: '18em',
    minWidth: '22em',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li': {
    display: 'flex',
    alignItems: 'baseline',
    gap: '10px',
    padding: '4px 10px',
    lineHeight: '1.45',
    color: 'var(--ui-text)',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]': {
    backgroundColor: 'color-mix(in oklab, var(--ui-primary) 22%, var(--ui-bg-elevated))',
    color: 'var(--ui-text-highlighted)',
  },
  '.cm-completionLabel': { flexShrink: '0' },
  '.cm-completionDetail': {
    marginLeft: 'auto',
    fontStyle: 'normal',
    fontSize: '11px',
    color: 'var(--ui-text-muted)',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    maxWidth: '24em',
  },
  '.cm-completionIcon': {
    display: 'inline-block',
    width: '7px',
    height: '7px',
    borderRadius: '50%',
    padding: '0',
    marginRight: '2px',
    opacity: '1',
    alignSelf: 'center',
  },
  '.cm-completionIcon::after': { content: "''" }, // 去掉默认字符图标
  '.cm-completionIcon-function': { backgroundColor: '#dcdcaa' },
  '.cm-completionIcon-variable': { backgroundColor: '#9cdcfe' },
  '.cm-completionIcon-keyword': { backgroundColor: '#569cd6' },
  '.cm-completionIcon-snippet': { backgroundColor: '#d7ba7d' },
  '.cm-tooltip.cm-completionInfo': {
    backgroundColor: 'var(--ui-bg-elevated)',
    border: '1px solid var(--ui-border-accented)',
    borderRadius: '6px',
    fontSize: '12px',
    padding: '6px 10px',
    maxWidth: '26em',
  },
})

// ── 查找/替换面板: CodeMirror 默认是裸原生控件, 按 app 暗色 token 重排 ──

export const searchPanelTheme: Extension = EditorView.theme({
  '.cm-panels': { backgroundColor: 'transparent', border: 'none' },
  '.cm-panel.cm-search': {
    backgroundColor: 'var(--ui-bg-elevated)',
    borderBottom: '1px solid var(--ui-border-accented)',
    padding: '8px 34px 8px 10px',
    display: 'flex',
    flexWrap: 'wrap',
    alignItems: 'center',
    gap: '6px',
    fontSize: '11px',
    color: 'var(--ui-text-muted)',
    position: 'relative',
  },
  '.cm-panel.cm-search input[type="text"], .cm-panel.cm-search input:not([type])': {
    backgroundColor: 'var(--ui-bg)',
    border: '1px solid var(--ui-border-accented)',
    borderRadius: '4px',
    padding: '3px 8px',
    fontSize: '12px',
    color: 'var(--ui-text)',
    outline: 'none',
    width: '14em',
  },
  '.cm-panel.cm-search input:focus': { borderColor: 'var(--ui-primary)' },
  '.cm-panel.cm-search button.cm-button': {
    backgroundImage: 'none',
    backgroundColor: 'var(--ui-bg-accented)',
    border: '1px solid var(--ui-border-accented)',
    borderRadius: '4px',
    padding: '3px 10px',
    fontSize: '11px',
    color: 'var(--ui-text)',
    cursor: 'pointer',
  },
  '.cm-panel.cm-search button.cm-button:hover': {
    backgroundColor: 'color-mix(in oklab, var(--ui-bg-accented) 88%, white)',
  },
  '.cm-panel.cm-search label': {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '4px',
    fontSize: '11px',
    color: 'var(--ui-text-muted)',
    whiteSpace: 'nowrap',
  },
  '.cm-panel.cm-search input[type="checkbox"]': { accentColor: 'var(--ui-primary)' },
  '.cm-panel.cm-search button[name="close"]': {
    position: 'absolute',
    top: '6px',
    right: '8px',
    color: 'var(--ui-text-muted)',
    fontSize: '16px',
    cursor: 'pointer',
    background: 'none',
    border: 'none',
  },
  '.cm-panel.cm-search button[name="close"]:hover': { color: 'var(--ui-text-highlighted)' },
})

// CodeMirror 查找/替换面板的中文文案 (phrases key 是固定英文原文)。
export const zhSearchPhrases: Extension = EditorState.phrases.of({
  Find: i18n.global.t('editorSearch.find'),
  Replace: i18n.global.t('editorSearch.replace'),
  next: i18n.global.t('editorSearch.next'),
  previous: i18n.global.t('editorSearch.previous'),
  all: i18n.global.t('editorSearch.all'),
  'match case': i18n.global.t('editorSearch.match_case'),
  'by word': i18n.global.t('editorSearch.by_word'),
  regexp: i18n.global.t('editorSearch.regexp'),
  replace: i18n.global.t('editorSearch.replace'),
  'replace all': i18n.global.t('editorSearch.replace_all'),
  close: i18n.global.t('editorSearch.close'),
})

// ── 共享基础扩展: 主题 + 编辑手感, 语言无关 (语言/补全/lint 由各语言工厂叠加) ──

export interface BaseEditorOpts {
  /** 放大编辑 modal 档: 行号/当前行/lint gutter/大字号; 缺省小框档。 */
  modal?: boolean
  /** 小框档的最小高度 (modal 档忽略, 占满容器)。 */
  minHeight?: string
}

export function baseEditorExtensions(opts: BaseEditorOpts = {}): Extension[] {
  return [
    chromeTheme,
    // position:fixed 逃 overflow-hidden; parent:body 再逃放大编辑 modal 的 transform/overflow 包含块
    // (光标在第一行时签名/hover 浮层 above 出框, 只挂 body 才彻底不被裁)。z-index 见 chromeTheme .cm-tooltip。
    // modal 档再把浮层限制在编辑器矩形内 → above 放不下 (第一行) 时 CM 自动翻到行下方, 不压工具栏;
    // 小框档不限制 (浮层本就要逃出小框, 用默认视口空间)。
    tooltips({
      position: 'fixed',
      parent: document.body,
      ...(opts.modal
        ? { tooltipSpace: (view: EditorView) => view.dom.getBoundingClientRect() }
        : {}),
    }),
    opts.modal ? modalSizeTheme : smallSizeTheme(opts.minHeight ?? '3.9em'),
    syntaxHighlighting(editorHighlightStyle),
    completionTooltipTheme,
    history(),
    drawSelection(),
    EditorState.allowMultipleSelections.of(true),
    closeBrackets(),
    bracketMatching(),
    indentOnInput(),
    indentUnit.of('  '),
    highlightSelectionMatches(),
    // gutter 顺序即扩展顺序: [lint 标记] [行号] (折叠由 script 工厂追加在更后 → 行号右侧)
    // 不挂 scrollPastEnd — 它给滚动区垫一屏虚拟空白, 三行代码也常驻滚动条
    ...(opts.modal
      ? [lintGutter(), lineNumbers(), highlightActiveLine(), highlightActiveLineGutter()]
      : [lineNumbers()]),
    keymap.of([
      // Tab 先补全上屏, 没有补全时缩进 (indentWithTab 含 Shift-Tab 反缩进)
      { key: 'Tab', run: acceptCompletion },
      ...closeBracketsKeymap,
      ...completionKeymap,
      indentWithTab,
      { key: 'Mod-d', run: selectNextOccurrence, preventDefault: true },
      ...defaultKeymap,
      ...historyKeymap,
    ]),
    EditorView.lineWrapping,
  ]
}
