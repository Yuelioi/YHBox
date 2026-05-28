// 全局快捷键注册. 从 ContainerEditorView 抽出 (backlog C1).
// 顺手 dedup 5 处 isTypingTarget 重复.

import { onMounted, onUnmounted, type Ref } from 'vue'

interface UseEditorHotkeysOpts {
  // refs (composable 内部读写)
  commandPaletteOpen: Ref<boolean>
  nodeSearchOpen: Ref<boolean>
  settingsOpen: Ref<boolean>
  nodeExplorerOpen: Ref<boolean>
  dirty: Ref<boolean>
  // actions (onSave 返回 Promise<boolean> 也兼容)
  onSave: () => Promise<unknown>
  undo: () => void
  redo: () => void
}

// 文本输入聚焦时不抢键 (input / textarea / select / contentEditable)
function isTypingTarget(e: KeyboardEvent): boolean {
  const t = e.target as HTMLElement | null
  const tag = t?.tagName?.toLowerCase()
  return tag === 'input' || tag === 'textarea' || tag === 'select' || !!t?.isContentEditable
}

export function useEditorHotkeys(opts: UseEditorHotkeysOpts) {
  function onKeydown(e: KeyboardEvent) {
    const mod = e.ctrlKey || e.metaKey

    // Ctrl+K → 命令面板
    if (mod && e.key.toLowerCase() === 'k') {
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.commandPaletteOpen.value = true
      return
    }
    // Ctrl+F → 画布节点搜索
    if (mod && e.key.toLowerCase() === 'f') {
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.nodeSearchOpen.value = !opts.nodeSearchOpen.value
      return
    }
    // Ctrl+S → 保存
    if (mod && e.key.toLowerCase() === 's') {
      e.preventDefault()
      if (opts.dirty.value) void opts.onSave()
      return
    }
    // Ctrl+, → settings (Mac Cmd+, also works)
    if (mod && e.key === ',') {
      e.preventDefault()
      opts.settingsOpen.value = true
      return
    }
    // Ctrl+Z → undo
    if (mod && !e.shiftKey && e.key.toLowerCase() === 'z') {
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.undo()
      return
    }
    // Ctrl+Shift+Z / Ctrl+Y → redo
    if (mod && ((e.shiftKey && e.key.toLowerCase() === 'z') || (!e.shiftKey && e.key.toLowerCase() === 'y'))) {
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.redo()
      return
    }
    // Tab → toggle NodeExplorer
    if (e.key === 'Tab') {
      // explorer 已开 — 任意位置 Tab 关 (含内部 search input, 故不过 isTypingTarget)
      if (opts.nodeExplorerOpen.value) {
        e.preventDefault()
        opts.nodeExplorerOpen.value = false
        return
      }
      // 未开 — 打开但跳过文本输入态
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.nodeExplorerOpen.value = true
    }
  }

  onMounted(() => window.addEventListener('keydown', onKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onKeydown))
}
