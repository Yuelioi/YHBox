import { describe, expect, it } from 'vitest'
import type { MacroAction } from '@/lib/backend'
import {
  analyzeMacroActions,
  canonicalBrowserKey,
  duplicateMacroAction,
  moveMacroAction,
} from './macroEditorModel'

function action(id: string, value: Omit<MacroAction, 'id'>): MacroAction {
  return { id, ...value }
}

describe('macro editor model', () => {
  it('keeps overlapping movement keys as independent held states', () => {
    const actions = [
      action('1', { kind: 'key-down', key: 'W' }),
      action('2', { kind: 'key-down', key: 'D' }),
      action('3', { kind: 'sleep', durationUs: 120_000 }),
      action('4', { kind: 'key-up', key: 'W' }),
      action('5', { kind: 'key-up', key: 'D' }),
    ]

    const result = analyzeMacroActions(actions)

    expect(result.issues).toEqual([])
    expect(result.heldAfter).toEqual([
      { keys: ['W'], buttons: [] },
      { keys: ['D', 'W'], buttons: [] },
      { keys: ['D', 'W'], buttons: [] },
      { keys: ['D'], buttons: [] },
      { keys: [], buttons: [] },
    ])
    expect(result.durationUs).toBe(120_000)
  })

  it('reports invalid transitions instead of silently grouping them', () => {
    const result = analyzeMacroActions([
      action('1', { kind: 'key-up', key: 'W' }),
      action('2', { kind: 'key-down', key: 'D' }),
      action('3', { kind: 'key-down', key: 'D' }),
      action('4', {
        kind: 'mouse-down',
        button: 'left',
        point: { x: 0.5, y: 0.5, unit: 'ratio' },
      }),
      action('5', {
        kind: 'click',
        button: 'left',
        durationUs: 50_000,
        point: { x: 0.5, y: 0.5, unit: 'ratio' },
      }),
    ])

    expect(result.issues.map((issue) => issue.code)).toEqual([
      'key-not-down',
      'key-already-down',
      'click-button-held',
      'held-at-end',
    ])
  })

  it('normalizes captured browser keys to the macro contract', () => {
    expect(canonicalBrowserKey('Control')).toBe('CTRL')
    expect(canonicalBrowserKey('ArrowLeft')).toBe('LEFT')
    expect(canonicalBrowserKey('w')).toBe('W')
    expect(canonicalBrowserKey('F12')).toBe('F12')
    expect(canonicalBrowserKey('Meta')).toBe('')
  })

  it('moves and duplicates actions without mutating their source', () => {
    const source = [
      action('1', { kind: 'key-down', key: 'W' }),
      action('2', { kind: 'key-up', key: 'W' }),
    ]

    const moved = moveMacroAction(source, 0, 1)
    const duplicated = duplicateMacroAction(source, 0, 'copy')

    expect(moved.map((item) => item.id)).toEqual(['2', '1'])
    expect(duplicated.map((item) => item.id)).toEqual(['1', 'copy', '2'])
    expect(source.map((item) => item.id)).toEqual(['1', '2'])
  })
})
