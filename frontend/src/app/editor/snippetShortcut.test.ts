import { describe, expect, it } from 'vitest'
import { shortcutFromKeyboardEvent, snippetShortcutIssue } from './snippetShortcut'

describe('snippet shortcuts', () => {
  it('normalizes keyboard events to the persisted shortcut format', () => {
    expect(
      shortcutFromKeyboardEvent(
        new KeyboardEvent('keydown', { code: 'KeyK', ctrlKey: true, shiftKey: true }),
      ),
    ).toBe('Ctrl+Shift+K')
    expect(shortcutFromKeyboardEvent(new KeyboardEvent('keydown', { code: 'F8' }))).toBe('F8')
  })

  it('rejects editor-reserved, unsafe, and duplicate bindings', () => {
    expect(snippetShortcutIssue('Ctrl+C', '', [])).toBe('reserved')
    expect(snippetShortcutIssue('K', '', [])).toBe('invalid')
    expect(snippetShortcutIssue('Ctrl+K', 'two', [{ id: 'one', shortcut: 'Ctrl+K' }])).toBe(
      'duplicate',
    )
    expect(snippetShortcutIssue('Ctrl+K', 'one', [{ id: 'one', shortcut: 'Ctrl+K' }])).toBe('')
  })
})
