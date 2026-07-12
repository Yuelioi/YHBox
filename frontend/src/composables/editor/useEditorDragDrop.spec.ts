import { describe, it, expect } from 'vitest'
import { startEditorDrag, readDragPayload, MIME, type EditorDragPayload } from './useEditorDragDrop'

function fakeDataTransfer(): DataTransfer {
  const map = new Map<string, string>()
  return {
    setData: (k: string, v: string) => map.set(k, v),
    getData: (k: string) => map.get(k) ?? '',
    effectAllowed: 'none',
  } as unknown as DataTransfer
}

describe('useEditorDragDrop', () => {
  it('startEditorDrag stores payload + sets effectAllowed', () => {
    const dt = fakeDataTransfer()
    const e = { dataTransfer: dt, altKey: false } as unknown as DragEvent
    const payload: EditorDragPayload = {
      type: 'var',
      ref: { name: 'x', type: 'number' },
      modifier: 'none',
    }
    startEditorDrag(payload, e)
    expect(dt.effectAllowed).toBe('copy')
    expect(JSON.parse(dt.getData(MIME))).toEqual(payload)
  })

  it('startEditorDrag auto-detects Alt modifier for var payload', () => {
    const dt = fakeDataTransfer()
    const e = { dataTransfer: dt, altKey: true } as unknown as DragEvent
    const payload: EditorDragPayload = {
      type: 'var',
      ref: { name: 'x', type: 'number' },
      modifier: 'none',
    }
    startEditorDrag(payload, e)
    expect(JSON.parse(dt.getData(MIME)).modifier).toBe('alt')
  })

  it('readDragPayload returns null on missing MIME', () => {
    const dt = fakeDataTransfer()
    const e = { dataTransfer: dt } as unknown as DragEvent
    expect(readDragPayload(e)).toBeNull()
  })

  it('readDragPayload parses var payload', () => {
    const dt = fakeDataTransfer()
    const payload: EditorDragPayload = {
      type: 'var',
      ref: { name: 'y', type: 'string' },
      modifier: 'none',
    }
    dt.setData(MIME, JSON.stringify(payload))
    const e = { dataTransfer: dt } as unknown as DragEvent
    expect(readDragPayload(e)).toEqual(payload)
  })
})
