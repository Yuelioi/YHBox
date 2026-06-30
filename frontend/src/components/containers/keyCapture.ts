// Shared single-key capture mapping for inspector KeyCapture and script editor insertion.
export function keyEventToVK(e: KeyboardEvent): string {
  if (/^[a-zA-Z]$/.test(e.key)) return e.key.toUpperCase()
  if (/^[0-9]$/.test(e.key)) return e.key
  const map: Record<string, string> = {
    ' ': 'Space',
    Enter: 'Enter',
    Tab: 'Tab',
    Escape: 'Esc',
    Backspace: 'Back',
    Delete: 'Del',
    ArrowUp: 'Up',
    ArrowDown: 'Down',
    ArrowLeft: 'Left',
    ArrowRight: 'Right',
    Control: 'Ctrl',
    Shift: 'Shift',
    Alt: 'Alt',
  }
  if (map[e.key]) return map[e.key]
  if (/^F([1-9]|1[0-2])$/.test(e.key)) return e.key
  return e.key
}
