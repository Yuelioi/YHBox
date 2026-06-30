import { describe, expect, it } from 'vitest'
import { keyEventToVK } from './keyCapture'

function ev(key: string): KeyboardEvent {
  return { key } as KeyboardEvent
}

describe('keyEventToVK', () => {
  it('maps letters, digits, function keys and common special keys to backend VK names', () => {
    expect(keyEventToVK(ev('f'))).toBe('F')
    expect(keyEventToVK(ev('7'))).toBe('7')
    expect(keyEventToVK(ev('F9'))).toBe('F9')
    expect(keyEventToVK(ev(' '))).toBe('Space')
    expect(keyEventToVK(ev('Escape'))).toBe('Esc')
    expect(keyEventToVK(ev('ArrowLeft'))).toBe('Left')
    expect(keyEventToVK(ev('Control'))).toBe('Ctrl')
  })
})
