import type { TypeExpression } from '../../../../contracts/node/3.1/authoring-projection'

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

function isModifierCode(code: string): boolean {
  return /^(?:Control|Shift|Alt|Meta)(?:Left|Right)$/.test(code)
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
