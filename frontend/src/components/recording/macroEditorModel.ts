import type { MacroAction, MacroActionKind, MacroDocument } from '@/lib/backend'

export type MacroEditorRowKind = MacroActionKind | 'key-press'

export interface MacroEditorRow {
  id: string
  kind: MacroEditorRowKind
  source: 'atomic' | 'key-hold' | 'mouse-hold'
  startIndex: number
  endIndex: number
  actionIds: string[]
  key?: string
  button?: MacroAction['button']
  from?: MacroAction['from']
  point?: MacroAction['point']
  notches?: number
  durationUs?: number
  motion?: MacroAction['motion']
}

export type MacroEditorRowPatch = Pick<
  MacroAction,
  'key' | 'button' | 'from' | 'point' | 'notches' | 'durationUs' | 'motion'
>

export type MacroEditorIssue = {
  code:
    | 'key-already-down'
    | 'key-not-down'
    | 'button-already-down'
    | 'button-not-down'
    | 'pointer-action-button-held'
    | 'auto-move-mode'
    | 'auto-move-duration'
    | 'held-at-end'
  index: number
  key?: string
  button?: string
}

export function analyzeMacroActions(actions: MacroAction[]) {
  const heldKeys = new Set<string>()
  const heldButtons = new Set<string>()
  const heldAfter: Array<{ keys: string[]; buttons: string[] }> = []
  const issues: MacroEditorIssue[] = []
  let durationUs = 0
  for (const [index, action] of actions.entries()) {
    if (action.kind === 'key-down') {
      if (heldKeys.has(action.key ?? ''))
        issues.push({ code: 'key-already-down', index, key: action.key })
      else heldKeys.add(action.key ?? '')
    } else if (action.kind === 'key-up') {
      if (!heldKeys.delete(action.key ?? ''))
        issues.push({ code: 'key-not-down', index, key: action.key })
    } else if (action.kind === 'mouse-down') {
      if (heldButtons.has(action.button ?? '')) issues.push({ code: 'button-already-down', index })
      else heldButtons.add(action.button ?? '')
    } else if (action.kind === 'mouse-up') {
      if (!heldButtons.delete(action.button ?? '')) issues.push({ code: 'button-not-down', index })
    } else if (
      (action.kind === 'click' || action.kind === 'drag') &&
      heldButtons.has(action.button ?? '')
    ) {
      issues.push({
        code: 'pointer-action-button-held',
        index,
        button: action.button,
      })
    }
    if (
      action.kind === 'sleep' ||
      action.kind === 'click' ||
      action.kind === 'move' ||
      action.kind === 'drag'
    )
      durationUs += action.durationUs ?? 0
    heldAfter.push({ keys: [...heldKeys].sort(), buttons: [...heldButtons].sort() })
  }
  if (heldKeys.size || heldButtons.size)
    issues.push({ code: 'held-at-end', index: actions.length - 1 })
  return { issues, heldAfter, durationUs }
}

export function analyzeMacroDocument(document: MacroDocument) {
  const analysis = analyzeMacroActions(document.actions)
  const issues = [...analysis.issues]
  const autoMove = document.meta.autoMove
  if (!['instant', 'linear', 'bezier'].includes(autoMove.mode)) {
    issues.unshift({ code: 'auto-move-mode', index: -1 })
  } else if (
    autoMove.durationMs < 0 ||
    autoMove.durationMs > 60_000 ||
    (autoMove.mode === 'instant' && autoMove.durationMs !== 0) ||
    (autoMove.mode !== 'instant' && autoMove.durationMs <= 0)
  ) {
    issues.unshift({ code: 'auto-move-duration', index: -1 })
  }

  let durationUs = analysis.durationUs
  let cursor: MacroAction['point']
  if (!issues.some((issue) => issue.code.startsWith('auto-move')) && autoMove.enabled) {
    for (const action of document.actions) {
      if (
        action.kind === 'click' &&
        action.point &&
        (!cursor || pointDistancePixels(cursor, action.point, document.baseResolution) >= 5)
      ) {
        durationUs += autoMove.durationMs * 1000
      }
      if (
        action.point &&
        ['move', 'click', 'mouse-down', 'mouse-up', 'scroll', 'drag'].includes(action.kind)
      ) {
        cursor = action.point
      }
    }
  }
  return { ...analysis, issues, durationUs }
}

export function canonicalBrowserKey(value: string): string {
  const named: Record<string, string> = {
    Control: 'CTRL',
    Escape: 'ESC',
    ' ': 'SPACE',
    PageUp: 'PGUP',
    PageDown: 'PGDN',
    ArrowLeft: 'LEFT',
    ArrowUp: 'UP',
    ArrowRight: 'RIGHT',
    ArrowDown: 'DOWN',
  }
  if (named[value]) return named[value]
  if (/^[a-z0-9]$/i.test(value)) return value.toUpperCase()
  if (/^F([1-9]|1[0-9]|2[0-4])$/.test(value)) return value
  return [
    'Backspace',
    'Tab',
    'Enter',
    'Shift',
    'Alt',
    'CapsLock',
    'End',
    'Home',
    'Insert',
    'Delete',
    ',',
    '.',
  ].includes(value)
    ? value.toUpperCase()
    : ''
}

export function moveMacroAction(actions: MacroAction[], from: number, to: number): MacroAction[] {
  const next = actions.map(cloneMacroAction)
  if (from < 0 || from >= next.length || to < 0 || to >= next.length || from === to) return next
  const [action] = next.splice(from, 1)
  if (action) next.splice(to, 0, action)
  return next
}

export function projectMacroRows(actions: MacroAction[], simplified: boolean): MacroEditorRow[] {
  const rows: MacroEditorRow[] = []
  for (let index = 0; index < actions.length; index++) {
    const first = actions[index]
    if (!first) continue
    const wait = actions[index + 1]
    const last = actions[index + 2]
    if (
      simplified &&
      first.kind === 'key-down' &&
      wait?.kind === 'sleep' &&
      last?.kind === 'key-up' &&
      first.key === last.key
    ) {
      rows.push({
        id: `key-press:${first.id}:${last.id}`,
        kind: 'key-press',
        source: 'key-hold',
        startIndex: index,
        endIndex: index + 2,
        actionIds: [first.id, wait.id, last.id],
        key: first.key,
        durationUs: wait.durationUs,
      })
      index += 2
      continue
    }
    if (
      simplified &&
      first.kind === 'mouse-down' &&
      wait?.kind === 'sleep' &&
      last?.kind === 'mouse-up' &&
      first.button === last.button &&
      samePoint(first.point, last.point)
    ) {
      rows.push({
        id: `click:${first.id}:${last.id}`,
        kind: 'click',
        source: 'mouse-hold',
        startIndex: index,
        endIndex: index + 2,
        actionIds: [first.id, wait.id, last.id],
        button: first.button,
        point: first.point ? { ...first.point } : undefined,
        durationUs: wait.durationUs,
      })
      index += 2
      continue
    }
    rows.push({
      id: first.id,
      kind: first.kind,
      source: 'atomic',
      startIndex: index,
      endIndex: index,
      actionIds: [first.id],
      key: first.key,
      button: first.button,
      from: first.from ? { ...first.from } : undefined,
      point: first.point ? { ...first.point } : undefined,
      notches: first.notches,
      durationUs: first.durationUs,
      motion: first.motion,
    })
  }
  return rows
}

export function patchMacroRow(
  actions: MacroAction[],
  row: MacroEditorRow,
  patch: Partial<MacroEditorRowPatch>,
): MacroAction[] {
  return actions.map((action, index) => {
    const next = cloneMacroAction(action)
    if (index < row.startIndex || index > row.endIndex) return next
    if (row.source === 'key-hold') {
      if ((next.kind === 'key-down' || next.kind === 'key-up') && patch.key !== undefined)
        next.key = patch.key
      if (next.kind === 'sleep' && patch.durationUs !== undefined)
        next.durationUs = patch.durationUs
      return next
    }
    if (row.source === 'mouse-hold') {
      if (next.kind === 'mouse-down' || next.kind === 'mouse-up') {
        if (patch.button !== undefined) next.button = patch.button
        if (patch.point !== undefined) next.point = { ...patch.point }
      }
      if (next.kind === 'sleep' && patch.durationUs !== undefined)
        next.durationUs = patch.durationUs
      return next
    }
    return {
      ...next,
      ...patch,
      from: patch.from ? { ...patch.from } : next.from,
      point: patch.point ? { ...patch.point } : next.point,
    }
  })
}

export function insertMacroActions(
  actions: MacroAction[],
  afterIndex: number,
  additions: MacroAction[],
): MacroAction[] {
  const next = actions.map(cloneMacroAction)
  const insertion = Math.max(0, Math.min(next.length, afterIndex + 1))
  next.splice(insertion, 0, ...additions.map(cloneMacroAction))
  return next
}

export function duplicateMacroRows(
  actions: MacroAction[],
  rows: MacroEditorRow[],
  id: (sourceID: string, offset: number) => string,
): MacroAction[] {
  const sourceByID = new Map(actions.map((action) => [action.id, action]))
  const ordered = [...rows].sort((left, right) => left.startIndex - right.startIndex)
  const next = actions.map(cloneMacroAction)
  const insertion = ordered.length
    ? Math.max(...ordered.map((row) => row.endIndex)) + 1
    : next.length
  const copies: MacroAction[] = []
  let offset = 0
  for (const row of ordered) {
    for (const actionID of row.actionIds) {
      const source = sourceByID.get(actionID)
      if (source) copies.push({ ...cloneMacroAction(source), id: id(source.id, offset++) })
    }
  }
  next.splice(insertion, 0, ...copies)
  return next
}

export function moveMacroRows(
  actions: MacroAction[],
  rows: MacroEditorRow[],
  insertAt: number,
): MacroAction[] {
  const selected = new Set(rows.flatMap((row) => row.actionIds))
  if (!selected.size) return actions.map(cloneMacroAction)
  const boundary = Math.max(0, Math.min(actions.length, insertAt))
  const insertion = actions.slice(0, boundary).filter((action) => !selected.has(action.id)).length
  const moving = actions.filter((action) => selected.has(action.id)).map(cloneMacroAction)
  const remaining = actions.filter((action) => !selected.has(action.id)).map(cloneMacroAction)
  remaining.splice(insertion, 0, ...moving)
  return remaining
}

export function duplicateMacroAction(
  actions: MacroAction[],
  index: number,
  id: string,
): MacroAction[] {
  const next = actions.map(cloneMacroAction)
  const source = next[index]
  if (source) next.splice(index + 1, 0, { ...cloneMacroAction(source), id })
  return next
}

export function cloneMacroAction(action: MacroAction): MacroAction {
  return {
    ...action,
    from: action.from ? { ...action.from } : undefined,
    point: action.point ? { ...action.point } : undefined,
  }
}

function samePoint(left: MacroAction['point'], right: MacroAction['point']): boolean {
  const clickJitterTolerance = 0.0025
  return (
    left !== undefined &&
    right !== undefined &&
    left.unit === right.unit &&
    Math.abs(left.x - right.x) <= clickJitterTolerance &&
    Math.abs(left.y - right.y) <= clickJitterTolerance
  )
}

function pointDistancePixels(
  left: NonNullable<MacroAction['point']>,
  right: NonNullable<MacroAction['point']>,
  resolution: [number, number],
): number {
  return Math.hypot((left.x - right.x) * resolution[0], (left.y - right.y) * resolution[1])
}
