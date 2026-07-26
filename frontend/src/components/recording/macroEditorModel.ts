import type { MacroAction } from '@/lib/backend'

export type MacroEditorIssue = {
  code:
    | 'key-already-down'
    | 'key-not-down'
    | 'button-already-down'
    | 'button-not-down'
    | 'click-button-held'
    | 'held-at-end'
  index: number
  key?: string
  button?: string
}

export function analyzeMacroActions(actions: MacroAction[]) {
  const heldKeys = new Set<string>()
  const heldButtons = new Set<string>()
  const heldAfter: Array<{ keys: string[]; buttons: string[] }> = []
  const issues: MacroEditorIssue[] = []
  let durationUs = 0
  for (const [index, action] of actions.entries()) {
    if (action.kind === 'key-down') {
      if (heldKeys.has(action.key ?? ''))
        issues.push({ code: 'key-already-down', index, key: action.key })
      else heldKeys.add(action.key ?? '')
    } else if (action.kind === 'key-up') {
      if (!heldKeys.delete(action.key ?? ''))
        issues.push({ code: 'key-not-down', index, key: action.key })
    } else if (action.kind === 'mouse-down') {
      if (heldButtons.has(action.button ?? '')) issues.push({ code: 'button-already-down', index })
      else heldButtons.add(action.button ?? '')
    } else if (action.kind === 'mouse-up') {
      if (!heldButtons.delete(action.button ?? '')) issues.push({ code: 'button-not-down', index })
    } else if (action.kind === 'click' && heldButtons.has(action.button ?? '')) {
      issues.push({
        code: 'click-button-held',
        index,
        button: action.button,
      })
    }
    if (action.kind === 'sleep' || action.kind === 'click') durationUs += action.durationUs ?? 0
    heldAfter.push({ keys: [...heldKeys].sort(), buttons: [...heldButtons].sort() })
  }
  if (heldKeys.size || heldButtons.size)
    issues.push({ code: 'held-at-end', index: actions.length - 1 })
  return { issues, heldAfter, durationUs }
}

export function canonicalBrowserKey(value: string): string {
  const named: Record<string, string> = {
    Control: 'CTRL',
    Escape: 'ESC',
    ' ': 'SPACE',
    PageUp: 'PGUP',
    PageDown: 'PGDN',
    ArrowLeft: 'LEFT',
    ArrowUp: 'UP',
    ArrowRight: 'RIGHT',
    ArrowDown: 'DOWN',
  }
  if (named[value]) return named[value]
  if (/^[a-z0-9]$/i.test(value)) return value.toUpperCase()
  if (/^F([1-9]|1[0-9]|2[0-4])$/.test(value)) return value
  return [
    'Backspace',
    'Tab',
    'Enter',
    'Shift',
    'Alt',
    'CapsLock',
    'End',
    'Home',
    'Insert',
    'Delete',
    ',',
    '.',
  ].includes(value)
    ? value.toUpperCase()
    : ''
}

export function moveMacroAction(actions: MacroAction[], from: number, to: number): MacroAction[] {
  const next = actions.map(cloneMacroAction)
  if (from < 0 || from >= next.length || to < 0 || to >= next.length || from === to) return next
  const [action] = next.splice(from, 1)
  if (action) next.splice(to, 0, action)
  return next
}

export function duplicateMacroAction(
  actions: MacroAction[],
  index: number,
  id: string,
): MacroAction[] {
  const next = actions.map(cloneMacroAction)
  const source = next[index]
  if (source) next.splice(index + 1, 0, { ...cloneMacroAction(source), id })
  return next
}

export function cloneMacroAction(action: MacroAction): MacroAction {
  return { ...action, point: action.point ? { ...action.point } : undefined }
}
