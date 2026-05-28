// 命令面板 commands 列表 builder. 从 ContainerEditorView 抽 (backlog C1).
// 纯 declarative — ref + actions 进 opts, computed<Command[]> 出.
//
// 注意: 调用位置必须在所有 action 声明之后 (onSave / onValidate / ... 不能前向引用).
// open ref 跟 useEditorHotkeys 共享 — hotkey Ctrl+K 设 open=true, palette modal 关 open=false.

import { computed, type ComputedRef, type Ref } from 'vue'
import { useVueFlow } from '@vue-flow/core'
import { useContainerEditorStore } from '@/stores/containerEditor'
import { useExecutionStore } from '@/stores/execution'
import type { SidebarPrefs } from '@/composables/editor/useSidebarPrefs'
import type { AlignMode } from './useGraphLayout'
import type { Command } from '@/components/containers/CommandPalette.vue'

interface UseCommandPaletteOpts {
  // refs
  canUndo: Ref<boolean>
  canRedo: Ref<boolean>
  dirty: Ref<boolean>
  sidebarPrefs: Ref<SidebarPrefs>
  nodeExplorerOpen: Ref<boolean>
  libraryExplorerOpen: Ref<boolean>
  settingsOpen: Ref<boolean>
  nodeSearchOpen: Ref<boolean>
  // actions
  undo: () => void
  redo: () => void
  onCopySelection: () => void
  onPasteSelection: () => Promise<unknown> | void
  onFoldSelection: () => void
  onAlignSelected: (mode: AlignMode) => void
  onAutoLayout: (dir: 'LR' | 'TB') => void
  onSave: () => Promise<unknown>
  onValidate: () => Promise<unknown> | void
  onTryRun: () => Promise<unknown> | void
  onStopRun: () => Promise<unknown> | void
  onAddVar: () => void
}

export function useCommandPalette(opts: UseCommandPaletteOpts): { commands: ComputedRef<Command[]> } {
  const editorStore = useContainerEditorStore()
  const execStore = useExecutionStore()
  const { getSelectedNodes, removeNodes } = useVueFlow()

  const commands = computed<Command[]>(() => {
    const sel = getSelectedNodes.value ?? []
    const selCount = sel.length
    return [
      // ── edit ──
      {
        id: 'edit.copy', label: '复制选中', group: 'edit', icon: 'i-tabler-copy', shortcut: 'Ctrl+C',
        disabled: selCount === 0,
        exec: () => opts.onCopySelection(),
      },
      {
        id: 'edit.paste', label: '粘贴', group: 'edit', icon: 'i-tabler-clipboard', shortcut: 'Ctrl+V',
        exec: () => void opts.onPasteSelection(),
      },
      {
        id: 'edit.delete', label: '删除选中', group: 'edit', icon: 'i-tabler-trash', shortcut: 'Del',
        disabled: selCount === 0,
        exec: () => sel.forEach((n: any) => removeNodes([n.id])),
      },
      {
        id: 'edit.undo', label: '撤销', group: 'edit', icon: 'i-tabler-arrow-back-up', shortcut: 'Ctrl+Z',
        disabled: !opts.canUndo.value,
        exec: () => opts.undo(),
      },
      {
        id: 'edit.redo', label: '重做', group: 'edit', icon: 'i-tabler-arrow-forward-up', shortcut: 'Ctrl+Y',
        disabled: !opts.canRedo.value,
        exec: () => opts.redo(),
      },
      {
        id: 'edit.fold', label: '折叠为子图', group: 'edit', icon: 'i-tabler-package-import',
        keywords: ['fold', 'subgraph', '折叠'],
        disabled: selCount === 0,
        exec: () => opts.onFoldSelection(),
      },
      {
        id: 'edit.align-left', label: '对齐 - 左', group: 'edit', keywords: ['align', '对齐'],
        disabled: selCount < 2,
        exec: () => opts.onAlignSelected('left'),
      },
      {
        id: 'edit.align-right', label: '对齐 - 右', group: 'edit', keywords: ['align', '对齐'],
        disabled: selCount < 2,
        exec: () => opts.onAlignSelected('right'),
      },
      {
        id: 'edit.align-top', label: '对齐 - 顶', group: 'edit', keywords: ['align', '对齐'],
        disabled: selCount < 2,
        exec: () => opts.onAlignSelected('top'),
      },
      {
        id: 'edit.align-bottom', label: '对齐 - 底', group: 'edit', keywords: ['align', '对齐'],
        disabled: selCount < 2,
        exec: () => opts.onAlignSelected('bottom'),
      },
      {
        id: 'edit.align-center-h', label: '水平居中', group: 'edit', keywords: ['align', '对齐', '居中'],
        disabled: selCount < 2,
        exec: () => opts.onAlignSelected('center-h'),
      },
      {
        id: 'edit.align-center-v', label: '垂直居中', group: 'edit', keywords: ['align', '对齐', '居中'],
        disabled: selCount < 2,
        exec: () => opts.onAlignSelected('center-v'),
      },
      {
        id: 'edit.distribute-h', label: '水平等距分布', group: 'edit', keywords: ['distribute', '分布'],
        disabled: selCount < 3,
        exec: () => opts.onAlignSelected('h-equal'),
      },
      {
        id: 'edit.distribute-v', label: '垂直等距分布', group: 'edit', keywords: ['distribute', '分布'],
        disabled: selCount < 3,
        exec: () => opts.onAlignSelected('v-equal'),
      },
      {
        id: 'edit.auto-layout-lr', label: '自动布局 (横向)', group: 'edit',
        icon: 'i-tabler-layout-board-split', shortcut: 'Ctrl+L',
        keywords: ['layout', 'dagre', '布局'],
        exec: () => opts.onAutoLayout('LR'),
      },
      {
        id: 'edit.auto-layout-tb', label: '自动布局 (纵向)', group: 'edit',
        icon: 'i-tabler-layout-rows', shortcut: 'Ctrl+Shift+L',
        keywords: ['layout', 'dagre', '布局'],
        exec: () => opts.onAutoLayout('TB'),
      },

      // ── view ──
      {
        id: 'view.toggle-left-sidebar', label: '折叠/展开 左侧栏', group: 'view',
        keywords: ['sidebar', 'panel', '侧栏'],
        exec: () => { opts.sidebarPrefs.value.leftSidebarCollapsed = !opts.sidebarPrefs.value.leftSidebarCollapsed },
      },
      {
        id: 'view.toggle-inspector', label: '折叠/展开 右 Inspector', group: 'view',
        keywords: ['inspector', 'panel', '检查器'],
        exec: () => { opts.sidebarPrefs.value.inspectorCollapsed = !opts.sidebarPrefs.value.inspectorCollapsed },
      },

      // ── navigate ──
      {
        id: 'navigate.node-explorer', label: '打开 Node Explorer', group: 'navigate',
        icon: 'i-tabler-grid-dots', shortcut: 'Tab',
        keywords: ['explorer', 'node', '节点'],
        exec: () => { opts.nodeExplorerOpen.value = !opts.nodeExplorerOpen.value },
      },
      {
        id: 'navigate.library', label: '打开子图库 Explorer', group: 'navigate',
        icon: 'i-tabler-books',
        keywords: ['library', 'subgraph', '库', '子图'],
        exec: () => { opts.libraryExplorerOpen.value = true },
      },
      {
        id: 'navigate.settings', label: '打开容器设置', group: 'navigate',
        icon: 'i-tabler-settings', shortcut: 'Ctrl+,',
        keywords: ['settings', 'config', '设置'],
        exec: () => { opts.settingsOpen.value = true },
      },
      {
        id: 'navigate.back', label: '跳到主图 / 上一层级', group: 'navigate',
        keywords: ['back', 'up', '返回', '主图'],
        disabled: editorStore.editorPath.length === 0,
        exec: () => editorStore.popPath(),
      },

      // ── run ──
      {
        id: 'run.save', label: '保存', group: 'run',
        icon: 'i-tabler-check', shortcut: 'Ctrl+S',
        keywords: ['save', '保存'],
        disabled: !opts.dirty.value,
        exec: () => void opts.onSave(),
      },
      {
        id: 'run.validate', label: '检查 (Validate)', group: 'run',
        icon: 'i-tabler-checks',
        keywords: ['validate', 'check', '校验', '检查'],
        disabled: opts.dirty.value,
        exec: () => void opts.onValidate(),
      },
      {
        id: 'run.try-run', label: '试运行', group: 'run',
        icon: 'i-tabler-player-play',
        keywords: ['run', 'play', '运行'],
        disabled: opts.dirty.value || execStore.running,
        exec: () => void opts.onTryRun(),
      },
      {
        id: 'run.stop', label: '停止运行', group: 'run',
        icon: 'i-tabler-square',
        keywords: ['stop', 'halt', '停止'],
        disabled: !execStore.running,
        exec: () => void opts.onStopRun(),
      },

      // ── var ──
      {
        id: 'var.add', label: '添加变量', group: 'var',
        icon: 'i-tabler-circle-plus',
        keywords: ['variable', 'add', '变量', '添加'],
        exec: () => opts.onAddVar(),
      },

      // ── navigate: find-node (Ctrl+F) ──
      {
        id: 'navigate.find-node', label: '在画布找节点...', group: 'navigate',
        icon: 'i-tabler-search', shortcut: 'Ctrl+F',
        keywords: ['find', 'search', '搜索', '查找'],
        exec: () => { opts.nodeSearchOpen.value = true },
      },
    ]
  })

  return { commands }
}
