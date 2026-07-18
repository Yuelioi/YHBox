import { describe, expect, it } from 'vitest'
import { isKeyChordType, keyChordFromKeyboardEvent } from './keyChord'

describe('workflow key chord authoring', () => {
  it('normalizes browser key events to the KeyCode contract', () => {
    expect(
      keyChordFromKeyboardEvent({ code: 'KeyA', ctrlKey: true, shiftKey: true, altKey: false }),
    ).toEqual(['CTRL', 'SHIFT', 'A'])
    expect(
      keyChordFromKeyboardEvent({
        code: 'ArrowLeft',
        ctrlKey: false,
        shiftKey: false,
        altKey: true,
      }),
    ).toEqual(['ALT', 'LEFT'])
    expect(
      keyChordFromKeyboardEvent({
        code: 'Backspace',
        ctrlKey: false,
        shiftKey: false,
        altKey: false,
      }),
    ).toEqual(['BACKSPACE'])
  })

  it('waits for a non-modifier and rejects keys outside the contract', () => {
    expect(
      keyChordFromKeyboardEvent({
        code: 'ControlLeft',
        ctrlKey: true,
        shiftKey: false,
        altKey: false,
      }),
    ).toBeNull()
    expect(
      keyChordFromKeyboardEvent({
        code: 'Numpad1',
        ctrlKey: false,
        shiftKey: false,
        altKey: false,
      }),
    ).toBeNull()
  })

  it('recognizes only List<KeyCode> ports', () => {
    expect(
      isKeyChordType({
        kind: 'list',
        element: {
          kind: 'ref',
          ref: {
            typeId: 'https://schemas.yotta.dev/types/automation/key-code/v1',
            semanticDigest: 'sha256:test',
          },
        },
      }),
    ).toBe(true)
    expect(
      isKeyChordType({
        kind: 'list',
        element: {
          kind: 'ref',
          ref: {
            typeId: 'https://schemas.yotta.dev/types/core/string/v1',
            semanticDigest: 'sha256:test',
          },
        },
      }),
    ).toBe(false)
  })
})
