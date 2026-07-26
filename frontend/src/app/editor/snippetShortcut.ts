import type { WorkflowSnippetSummary } from '@/lib/backend'

const reserved = new Set([
  'Ctrl+A',
  'Ctrl+C',
  'Ctrl+D',
  'Ctrl+F',
  'Ctrl+S',
  'Ctrl+V',
  'Ctrl+X',
  'Ctrl+Y',
  'Ctrl+Z',
  'Ctrl+Shift+Z',
])

export type SnippetShortcutIssue = 'duplicate' | 'invalid' | 'reserved' | ''

export function shortcutFromKeyboardEvent(event: KeyboardEvent): string {
  if (event.repeat || isModifierCode(event.code)) return ''
  const key = shortcutKeyFromCode(event.code)
  if (!key) return ''
  const parts: string[] = []
  if (event.ctrlKey) parts.push('Ctrl')
  if (event.shiftKey) parts.push('Shift')
  if (event.altKey) parts.push('Alt')
  if (event.metaKey) parts.push('Meta')
  parts.push(key)
  return parts.join('+')
}

export function snippetShortcutIssue(
  shortcut: string,
  snippetID: string,
  existing: readonly Pick<WorkflowSnippetSummary, 'id' | 'shortcut'>[],
): SnippetShortcutIssue {
  const value = shortcut.trim()
  if (!value) return ''
  if (reserved.has(value)) return 'reserved'
  const parts = value.split('+')
  const key = parts.at(-1) ?? ''
  const hasModifier = parts.length > 1
  if ((!hasModifier && !/^F([1-9]|1[0-2])$/.test(key)) || !isSupportedShortcutKey(key)) {
    return 'invalid'
  }
  return existing.some(
    (item) =>
      item.id !== snippetID && item.shortcut?.toLocaleLowerCase() === value.toLocaleLowerCase(),
  )
    ? 'duplicate'
    : ''
}

function isModifierCode(code: string): boolean {
  return /^(Control|Shift|Alt|Meta)(Left|Right)$/.test(code)
}

function shortcutKeyFromCode(code: string): string {
  if (/^Key[A-Z]$/.test(code)) return code.slice(3)
  if (/^Digit[0-9]$/.test(code)) return code.slice(5)
  if (/^F([1-9]|1[0-2])$/.test(code)) return code
  return (
    (
      {
        Space: 'Space',
        Enter: 'Enter',
        Tab: 'Tab',
        Delete: 'Delete',
        Insert: 'Insert',
        Home: 'Home',
        End: 'End',
        PageUp: 'PgUp',
        PageDown: 'PgDn',
        ArrowUp: 'Up',
        ArrowDown: 'Down',
        ArrowLeft: 'Left',
        ArrowRight: 'Right',
        Period: '.',
        Comma: ',',
      } as Record<string, string>
    )[code] ?? ''
  )
}

function isSupportedShortcutKey(key: string): boolean {
  return (
    /^[A-Z0-9]$/.test(key) ||
    /^F([1-9]|1[0-2])$/.test(key) ||
    [
      'Space',
      'Enter',
      'Tab',
      'Delete',
      'Insert',
      'Home',
      'End',
      'PgUp',
      'PgDn',
      'Up',
      'Down',
      'Left',
      'Right',
      '.',
      ',',
    ].includes(key)
  )
}
