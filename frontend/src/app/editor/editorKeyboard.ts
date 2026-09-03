export type EditorKeyboardAction =
  | { kind: 'close-connection-menu' }
  | { kind: 'open-quick-add' }
  | { kind: 'use-snippet'; snippetID: string }
  | { kind: 'add-favorite-node'; nodeTypeId: string }
  | { kind: 'clear-selection' }
  | { kind: 'find-node' }
  | { kind: 'copy-selection' }
  | { kind: 'cut-selection' }
  | { kind: 'paste-selection' }
  | { kind: 'duplicate-selection' }
  | { kind: 'undo' }
  | { kind: 'redo' }
  | { kind: 'remove-selection' }

export interface EditorKeyboardContext {
  connectionMenuOpen: boolean
  canvasPointerInside: boolean
  hasNodeSelection: boolean
  hasSelection: boolean
  matchedSnippetID?: string
  favoriteNodeTypeIds?: string[]
}

export function resolveEditorKeyboardAction(
  event: KeyboardEvent,
  context: EditorKeyboardContext,
): EditorKeyboardAction | null {
  if (event.key === 'Escape' && context.connectionMenuOpen) {
    return { kind: 'close-connection-menu' }
  }
  if (!editorOwnsKeyboardEvent(event)) return null

  if (
    context.canvasPointerInside &&
    event.altKey &&
    !event.ctrlKey &&
    !event.metaKey &&
    !event.shiftKey &&
    /^[1-5]$/.test(event.key)
  ) {
    const nodeTypeId = context.favoriteNodeTypeIds?.[Number(event.key) - 1]
    if (nodeTypeId) return { kind: 'add-favorite-node', nodeTypeId }
  }

  if (
    event.key === 'Tab' &&
    !event.ctrlKey &&
    !event.metaKey &&
    !event.altKey &&
    !event.shiftKey &&
    context.canvasPointerInside
  ) {
    return { kind: 'open-quick-add' }
  }

  const modifier = event.ctrlKey || event.metaKey
  if (modifier && !event.altKey) {
    const key = event.key.toLocaleLowerCase()
    if (key === 'z') return { kind: event.shiftKey ? 'redo' : 'undo' }
    if (key === 'y' && !event.shiftKey) return { kind: 'redo' }
  }

  if (context.canvasPointerInside && context.matchedSnippetID) {
    return { kind: 'use-snippet', snippetID: context.matchedSnippetID }
  }
  if (event.key === 'Escape' && context.hasSelection) return { kind: 'clear-selection' }

  if (modifier && !event.altKey) {
    const key = event.key.toLocaleLowerCase()
    if (key === 'f') return { kind: 'find-node' }
    if (key === 'c' && context.hasNodeSelection) {
      return { kind: 'copy-selection' }
    }
    if (key === 'x' && context.hasNodeSelection) {
      return { kind: 'cut-selection' }
    }
    if (key === 'v') return { kind: 'paste-selection' }
    if (key === 'd' && context.hasNodeSelection) {
      return { kind: 'duplicate-selection' }
    }
  }
  if (
    !event.ctrlKey &&
    !event.metaKey &&
    !event.altKey &&
    !event.shiftKey &&
    (event.key === 'Delete' || event.key === 'Backspace') &&
    context.hasSelection
  ) {
    return { kind: 'remove-selection' }
  }
  return null
}

function editorOwnsKeyboardEvent(event: KeyboardEvent): boolean {
  const target = event.target instanceof Element ? event.target : null
  return !(
    target?.matches('input, textarea, select, [contenteditable="true"]') ||
    target?.closest('[role="dialog"]')
  )
}
