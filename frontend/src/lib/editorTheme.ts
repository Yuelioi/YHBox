// 编辑器共享外观: 补全提示 tooltip 的 VSCode 风格主题 (ExprInput / CodeInput / EditorModal
// 三处编辑器统一引入) + CodeMirror 查找面板的中文 phrases。
import { EditorView } from '@codemirror/view'
import { EditorState, type Extension } from '@codemirror/state'

// 补全下拉对齐 VSCode 暗色观感: 可读字号 (12px)、行高、明确的选中底色、
// detail 右对齐灰字、独立滚动。伤眼的 11px 密排是它要替换的对象。
export const completionTooltipTheme: Extension = EditorView.theme({
  '.cm-tooltip.cm-tooltip-autocomplete': {
    backgroundColor: '#1f2428',
    border: '1px solid #3c4147',
    borderRadius: '6px',
    boxShadow: '0 6px 16px rgba(0,0,0,.45)',
    overflow: 'hidden',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul': {
    fontFamily: 'ui-monospace, monospace',
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
    color: '#d4d4d4',
  },
  '.cm-tooltip.cm-tooltip-autocomplete > ul > li[aria-selected]': {
    backgroundColor: '#094771',
    color: '#ffffff',
  },
  '.cm-completionLabel': { flexShrink: '0' },
  '.cm-completionDetail': {
    marginLeft: 'auto',
    fontStyle: 'normal',
    fontSize: '11px',
    color: '#9da5b0',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    maxWidth: '24em',
  },
  '.cm-completionIcon': { display: 'none' }, // 默认字符图标对不齐, 信息量低 — 去掉
  // 补全详情侧栏 (info) 同步放大
  '.cm-tooltip.cm-completionInfo': {
    backgroundColor: '#1f2428',
    border: '1px solid #3c4147',
    borderRadius: '6px',
    fontSize: '12px',
    padding: '6px 10px',
    maxWidth: '26em',
  },
})

// CodeMirror 查找/替换面板的中文文案 (phrases key 是固定英文原文)。
export const zhSearchPhrases: Extension = EditorState.phrases.of({
  'Find': '查找',
  'Replace': '替换',
  'next': '下一个',
  'previous': '上一个',
  'all': '全部',
  'match case': '区分大小写',
  'by word': '全词匹配',
  'regexp': '正则',
  'replace': '替换',
  'replace all': '全部替换',
  'close': '关闭',
})
