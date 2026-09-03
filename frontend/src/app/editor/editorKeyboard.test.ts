import { describe, expect, it } from 'vitest'
import { resolveEditorKeyboardAction, type EditorKeyboardContext } from './editorKeyboard'

const context: EditorKeyboardContext = {
  connectionMenuOpen: false,
  canvasPointerInside: true,
  hasNodeSelection: true,
  hasSelection: true,
}

describe('editor keyboard policy', () => {
  it('maps conventional history shortcuts to undo and redo', () => {
    expect(
      resolveEditorKeyboardAction(
        new KeyboardEvent('keydown', { key: 'z', ctrlKey: true }),
        context,
      ),
    ).toEqual({ kind: 'undo' })
    expect(
      resolveEditorKeyboardAction(
        new KeyboardEvent('keydown', { key: 'z', ctrlKey: true, shiftKey: true }),
        context,
      ),
    ).toEqual({ kind: 'redo' })
    expect(
      resolveEditorKeyboardAction(
        new KeyboardEvent('keydown', { key: 'y', ctrlKey: true }),
        context,
      ),
    ).toEqual({ kind: 'redo' })
  })

  it('supports Command history shortcuts without stealing native text editing', () => {
    expect(
      resolveEditorKeyboardAction(
        new KeyboardEvent('keydown', { key: 'z', metaKey: true }),
        context,
      ),
    ).toEqual({ kind: 'undo' })

    const input = document.createElement('input')
    let action: ReturnType<typeof resolveEditorKeyboardAction> = { kind: 'undo' }
    input.addEventListener('keydown', (event) => {
      action = resolveEditorKeyboardAction(event, context)
    })
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'z', ctrlKey: true, bubbles: true }))
    expect(action).toBeNull()
  })

  it('maps the existing canvas commands and preserves their selection guards', () => {
    expect(
      resolveEditorKeyboardAction(
        new KeyboardEvent('keydown', { key: 'f', ctrlKey: true }),
        context,
      ),
    ).toEqual({ kind: 'find-node' })
    expect(
      resolveEditorKeyboardAction(new KeyboardEvent('keydown', { key: 'c', ctrlKey: true }), {
        ...context,
        hasNodeSelection: false,
      }),
    ).toBeNull()
    expect(
      resolveEditorKeyboardAction(new KeyboardEvent('keydown', { key: 'v', ctrlKey: true }), {
        ...context,
        hasNodeSelection: false,
        hasSelection: false,
      }),
    ).toEqual({ kind: 'paste-selection' })
    expect(
      resolveEditorKeyboardAction(new KeyboardEvent('keydown', { key: 'Delete' }), context),
    ).toEqual({ kind: 'remove-selection' })
  })

  it('opens quick add and dispatches configured snippet shortcuts only inside the canvas', () => {
    expect(
      resolveEditorKeyboardAction(new KeyboardEvent('keydown', { key: 'Tab' }), context),
    ).toEqual({ kind: 'open-quick-add' })
    expect(
      resolveEditorKeyboardAction(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }), {
        ...context,
        matchedSnippetID: 'snippet-one',
      }),
    ).toEqual({ kind: 'use-snippet', snippetID: 'snippet-one' })
    expect(
      resolveEditorKeyboardAction(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }), {
        ...context,
        canvasPointerInside: false,
        matchedSnippetID: 'snippet-one',
      }),
    ).toBeNull()
  })

  it('uses Alt+1 through Alt+5 for configured favorite nodes only on the canvas', () => {
    expect(
      resolveEditorKeyboardAction(new KeyboardEvent('keydown', { key: '2', altKey: true }), {
        ...context,
        favoriteNodeTypeIds: ['node-a', 'node-b'],
      }),
    ).toEqual({ kind: 'add-favorite-node', nodeTypeId: 'node-b' })
    expect(
      resolveEditorKeyboardAction(new KeyboardEvent('keydown', { key: '2', altKey: true }), {
        ...context,
        canvasPointerInside: false,
        favoriteNodeTypeIds: ['node-a', 'node-b'],
      }),
    ).toBeNull()
  })

  it('lets Escape close the connection menu before the dialog focus guard', () => {
    const input = document.createElement('input')
    let action: ReturnType<typeof resolveEditorKeyboardAction> = null
    input.addEventListener('keydown', (event) => {
      action = resolveEditorKeyboardAction(event, { ...context, connectionMenuOpen: true })
    })
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(action).toEqual({ kind: 'close-connection-menu' })
  })
})
