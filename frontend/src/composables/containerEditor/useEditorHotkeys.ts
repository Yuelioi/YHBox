// 全局快捷键注册.
//
// EDITOR_KEYS 进 hotkey registry: webview only (inAppOnly), readonly.
// 用户在 Settings → 快捷键 '编辑器' 分组看得见, 跟其它 source 撞 normalized
// 时 lastError 写回 entry 表里能看见. 派发不变: 仍硬编码 key 比较,
// registry 只挂可见性 + 冲突检查.

import { onActivated, onDeactivated, type Ref } from 'vue'
import { backend } from '@/lib/backend'

interface UseEditorHotkeysOpts {
  commandPaletteOpen: Ref<boolean>
  nodeSearchOpen: Ref<boolean>
  settingsOpen: Ref<boolean>
  nodeExplorerOpen: Ref<boolean>
  dirty: Ref<boolean>
  onSave: () => Promise<unknown>
  undo: () => void
  redo: () => void
}

// 7 editor in-app key. 跟 onKeydown 7 if 分支对得上.
// label 是 i18n key, SettingsHotkeys.vue t() 渲染.
const EDITOR_KEYS: ReadonlyArray<{ key: string; label: string; hotkeyStr: string }> = [
  { key: 'editor.command-palette', label: 'hotkeys.label.editor.commandPalette', hotkeyStr: 'Ctrl+K' },
  { key: 'editor.node-search',     label: 'hotkeys.label.editor.nodeSearch',     hotkeyStr: 'Ctrl+F' },
  { key: 'editor.save',            label: 'hotkeys.label.editor.save',           hotkeyStr: 'Ctrl+S' },
  { key: 'editor.open-settings',   label: 'hotkeys.label.editor.openSettings',   hotkeyStr: 'Ctrl+,' },
  { key: 'editor.undo',            label: 'hotkeys.label.editor.undo',           hotkeyStr: 'Ctrl+Z' },
  { key: 'editor.redo',            label: 'hotkeys.label.editor.redo',           hotkeyStr: 'Ctrl+Shift+Z' },
  { key: 'editor.toggle-explorer', label: 'hotkeys.label.editor.toggleExplorer', hotkeyStr: 'Tab' },
]
const READONLY_REASON = 'hotkeys.readonly.editorBuiltin'

function isTypingTarget(e: KeyboardEvent): boolean {
  const t = e.target as HTMLElement | null
  const tag = t?.tagName?.toLowerCase()
  return tag === 'input' || tag === 'textarea' || tag === 'select' || !!t?.isContentEditable
}

export function useEditorHotkeys(opts: UseEditorHotkeysOpts) {
  function onKeydown(e: KeyboardEvent) {
    const mod = e.ctrlKey || e.metaKey

    if (mod && e.key.toLowerCase() === 'k') {
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.commandPaletteOpen.value = true
      return
    }
    if (mod && e.key.toLowerCase() === 'f') {
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.nodeSearchOpen.value = !opts.nodeSearchOpen.value
      return
    }
    if (mod && e.key.toLowerCase() === 's') {
      e.preventDefault()
      if (opts.dirty.value) void opts.onSave()
      return
    }
    if (mod && e.key === ',') {
      e.preventDefault()
      opts.settingsOpen.value = true
      return
    }
    if (mod && !e.shiftKey && e.key.toLowerCase() === 'z') {
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.undo()
      return
    }
    if (mod && ((e.shiftKey && e.key.toLowerCase() === 'z') || (!e.shiftKey && e.key.toLowerCase() === 'y'))) {
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.redo()
      return
    }
    if (e.key === 'Tab') {
      if (opts.nodeExplorerOpen.value) {
        e.preventDefault()
        opts.nodeExplorerOpen.value = false
        return
      }
      if (isTypingTarget(e)) return
      e.preventDefault()
      opts.nodeExplorerOpen.value = true
    }
  }

  // keep-alive 内 (App.vue:26 include="ContainerEditorView" max:3): onActivated 切回时跑,
  // onDeactivated 切走时跑. 不用 mount/unmount — keep-alive 缓存的 instance 不会 unmount,
  // 用 mount/unmount 会让 register 失衡 (切到 settings 不卸, 再切回又 register 撞 duplicate).
  onActivated(async () => {
    for (const k of EDITOR_KEYS) {
      await backend.hotkeys.registerEditor(k.key, k.label, k.hotkeyStr, READONLY_REASON)
    }
    window.addEventListener('keydown', onKeydown)
  })
  onDeactivated(() => {
    for (const k of EDITOR_KEYS) {
      void backend.hotkeys.unregister(k.key)
    }
    window.removeEventListener('keydown', onKeydown)
  })
}
