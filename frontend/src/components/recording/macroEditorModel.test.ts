import { describe, expect, it } from 'vitest'
import type { MacroAction } from '@/lib/backend'
import {
  analyzeMacroActions,
  canonicalBrowserKey,
  duplicateMacroRows,
  duplicateMacroAction,
  insertMacroActions,
  moveMacroAction,
  moveMacroRows,
  patchMacroRow,
  projectMacroRows,
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

  it('projects recorded press-wait-release atoms as simple actions without changing storage', () => {
    const point = { x: 0.4, y: 0.6, unit: 'ratio' as const }
    const source = [
      action('kd', { kind: 'key-down', key: 'W' }),
      action('kw', { kind: 'sleep', durationUs: 250_000 }),
      action('ku', { kind: 'key-up', key: 'W' }),
      action('md', { kind: 'mouse-down', button: 'left', point }),
      action('mw', { kind: 'sleep', durationUs: 80_000 }),
      action('mu', {
        kind: 'mouse-up',
        button: 'left',
        point: { x: 0.401, y: 0.599, unit: 'ratio' },
      }),
    ]

    const simple = projectMacroRows(source, true)
    const atomic = projectMacroRows(source, false)

    expect(simple.map((row) => row.kind)).toEqual(['key-press', 'click'])
    expect(simple.map((row) => row.durationUs)).toEqual([250_000, 80_000])
    expect(simple.map((row) => row.actionIds)).toEqual([
      ['kd', 'kw', 'ku'],
      ['md', 'mw', 'mu'],
    ])
    expect(atomic).toHaveLength(6)
    expect(source.map((item) => item.id)).toEqual(['kd', 'kw', 'ku', 'md', 'mw', 'mu'])
  })

  it('keeps a mouse drag atomic instead of presenting it as a click', () => {
    const source = [
      action('md', {
        kind: 'mouse-down',
        button: 'left',
        point: { x: 0.4, y: 0.6, unit: 'ratio' },
      }),
      action('wait', { kind: 'sleep', durationUs: 80_000 }),
      action('mu', {
        kind: 'mouse-up',
        button: 'left',
        point: { x: 0.5, y: 0.7, unit: 'ratio' },
      }),
    ]

    expect(projectMacroRows(source, true).map((row) => row.kind)).toEqual([
      'mouse-down',
      'sleep',
      'mouse-up',
    ])
  })

  it('edits a simple key press through its atomic source actions', () => {
    const source = [
      action('kd', { kind: 'key-down', key: 'W' }),
      action('wait', { kind: 'sleep', durationUs: 100_000 }),
      action('ku', { kind: 'key-up', key: 'W' }),
    ]
    const [row] = projectMacroRows(source, true)
    if (!row) throw new Error('missing projected row')

    const updated = patchMacroRow(source, row, { key: 'E', durationUs: 450_000 })

    expect(updated).toEqual([
      action('kd', { kind: 'key-down', key: 'E' }),
      action('wait', { kind: 'sleep', durationUs: 450_000 }),
      action('ku', { kind: 'key-up', key: 'E' }),
    ])
  })

  it('inserts, duplicates, and moves selected rows as complete blocks', () => {
    const source = [
      action('a', { kind: 'sleep', durationUs: 10_000 }),
      action('b', { kind: 'sleep', durationUs: 20_000 }),
      action('c', { kind: 'sleep', durationUs: 30_000 }),
      action('d', { kind: 'sleep', durationUs: 40_000 }),
    ]
    const rows = projectMacroRows(source, true)
    const inserted = insertMacroActions(source, 1, [
      action('new', { kind: 'sleep', durationUs: 15_000 }),
    ])
    const duplicated = duplicateMacroRows(source, [rows[1]!, rows[2]!], (id) => `copy-${id}`)
    const moved = moveMacroRows(source, [rows[1]!, rows[2]!], 4)

    expect(inserted.map((item) => item.id)).toEqual(['a', 'b', 'new', 'c', 'd'])
    expect(duplicated.map((item) => item.id)).toEqual(['a', 'b', 'c', 'copy-b', 'copy-c', 'd'])
    expect(moved.map((item) => item.id)).toEqual(['a', 'd', 'b', 'c'])
  })
})
