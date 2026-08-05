import type { TypeExpression } from '../../../../contracts/node/current/authoring-projection'

export const keyCodeTypeId = 'https://schemas.yotta.dev/types/automation/key-code/v1'

type KeyboardEventShape = Pick<KeyboardEvent, 'code' | 'ctrlKey' | 'shiftKey' | 'altKey'>

export function isKeyChordType(expression: TypeExpression): boolean {
  return (
    expression.kind === 'list' &&
    expression.element.kind === 'ref' &&
    expression.element.ref.typeId === keyCodeTypeId
  )
}

export function keyChordFromKeyboardEvent(event: KeyboardEventShape): string[] | null {
  if (isModifierCode(event.code)) return null
  const key = contractKeyCode(event.code)
  if (!key) return null
  const chord: string[] = []
  if (event.ctrlKey) chord.push('CTRL')
  if (event.shiftKey) chord.push('SHIFT')
  if (event.altKey) chord.push('ALT')
  chord.push(key)
  return chord
}

export function modifierChordFromKeyboardEvent(event: KeyboardEventShape): string[] | null {
  const released = contractModifierKeyCode(event.code)
  if (!released) return null
  const chord: string[] = []
  if (event.ctrlKey || released === 'CTRL') chord.push('CTRL')
  if (event.shiftKey || released === 'SHIFT') chord.push('SHIFT')
  if (event.altKey || released === 'ALT') chord.push('ALT')
  return chord
}

function isModifierCode(code: string): boolean {
  return /^(?:Control|Shift|Alt|Meta)(?:Left|Right)$/.test(code)
}

function contractModifierKeyCode(code: string): 'CTRL' | 'SHIFT' | 'ALT' | null {
  if (/^Control(?:Left|Right)$/.test(code)) return 'CTRL'
  if (/^Shift(?:Left|Right)$/.test(code)) return 'SHIFT'
  if (/^Alt(?:Left|Right)$/.test(code)) return 'ALT'
  return null
}

function contractKeyCode(code: string): string | null {
  if (/^Key[A-Z]$/.test(code)) return code.slice(3)
  if (/^Digit[0-9]$/.test(code)) return code.slice(5)
  if (/^F(?:[1-9]|1[0-2])$/.test(code)) return code
  const keys: Record<string, string> = {
    Escape: 'ESC',
    Space: 'SPACE',
    Enter: 'ENTER',
    Tab: 'TAB',
    Backspace: 'BACKSPACE',
    Delete: 'DELETE',
    Insert: 'INSERT',
    Home: 'HOME',
    End: 'END',
    PageUp: 'PGUP',
    PageDown: 'PGDN',
    ArrowUp: 'UP',
    ArrowDown: 'DOWN',
    ArrowLeft: 'LEFT',
    ArrowRight: 'RIGHT',
    Comma: ',',
    Period: '.',
    CapsLock: 'CAPSLOCK',
  }
  return keys[code] ?? null
}
