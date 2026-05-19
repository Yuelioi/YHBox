// frontend/src/components/containers/nodeRegistry/specs/specs.test.ts
// B2 parity/consistency tests. Side-effect-import the barrel to populate the registry.
import { describe, it, expect } from 'vitest'
import { allKinds, getSpec } from '../registry'
import './index' // side-effect: registers all 72 kinds

const EXPECTED_KINDS = [
  // control (11)
  'Start',
  'Stop',
  'Sleep',
  'Loop',
  'If',
  'Switch',
  'Parallel',
  'Race',
  'Break',
  'Continue',
  'Cron',
  // variables (5)
  'SetVar',
  'IncVar',
  'GetVar',
  'GetSys',
  'GetParam',
  // expr (1)
  'Expr',
  // purefunc (22)
  'Add',
  'Sub',
  'Mul',
  'Div',
  'Mod',
  'Neg',
  'Lt',
  'LtEq',
  'Gt',
  'GtEq',
  'Eq',
  'NotEq',
  'And',
  'Or',
  'Not',
  'Concat',
  'Contains',
  'Length',
  'ToString',
  'ToNumber',
  'ToBool',
  'Select',
  // detect (8)
  'WaitTemplate',
  'CheckTemplate',
  'ClickTemplate',
  'DetectColor',
  'DetectColorHSV',
  'ROIColorScan',
  'Screenshot',
  'ColorBarTrack',
  // input (10)
  'ClickAt',
  'KeyPress',
  'MouseMoveRel',
  'Scroll',
  'KeyHoldStart',
  'KeyHoldStop',
  'MouseHoldStart',
  'MouseHoldStop',
  'OnEvent',
  'PlayClip',
  // system (15)
  'BringGameForeground',
  'MouseCalibration',
  'WindowTarget',
  'CommentBox',
  'Try',
  'Throw',
  'StopwatchStart',
  'StopwatchStop',
  'StopwatchRead',
  'Log',
  'Toast',
  'Subgraph',
  'SubgraphInput',
  'SubgraphOutput',
  'CollapsedNode',
]

describe('nodeRegistry specs', () => {
  it('registers all expected kinds (72)', () => {
    const registered = new Set(allKinds())
    for (const k of EXPECTED_KINDS) {
      expect(registered.has(k), `kind ${k} missing`).toBe(true)
    }
    expect(registered.size, 'unexpected extra kinds in registry').toBe(EXPECTED_KINDS.length)
  })

  it('every kind has all required fields', () => {
    for (const k of allKinds()) {
      const s = getSpec(k)!
      expect(s.kind, `${k}.kind`).toBe(k)
      expect(s.group, `${k}.group`).toBeTruthy()
      expect(s.labelZh, `${k}.labelZh`).toBeTruthy()
      expect(s.visual.icon, `${k}.visual.icon`).toBeTruthy()
    }
  })

  it('pure-data kinds have no exec pins', () => {
    for (const k of allKinds()) {
      const s = getSpec(k)!
      if (!s.isPureData) continue
      expect(s.execIn.length, `${k}.execIn`).toBe(0)
      expect(s.execOut.length, `${k}.execOut`).toBe(0)
      expect(s.execOutFn, `${k}.execOutFn`).toBeUndefined()
    }
  })

  it('visual-only kinds have no pins at all', () => {
    for (const k of allKinds()) {
      const s = getSpec(k)!
      if (!s.isVisualOnly) continue
      expect(s.execIn.length).toBe(0)
      expect(s.execOut.length).toBe(0)
      expect(Object.keys(s.dataIn).length).toBe(0)
      expect(Object.keys(s.dataOut).length).toBe(0)
    }
  })

  // Pin the contract bidirectionally: exactly these 3 kinds opt out of the
  // palette (created via dedicated UI flows — see system.ts comments).
  it('excludeFromPalette set on subgraph-machinery kinds', () => {
    const expected = new Set(['SubgraphInput', 'SubgraphOutput', 'CollapsedNode'])
    for (const k of allKinds()) {
      const s = getSpec(k)!
      expect(!!s.excludeFromPalette, `${k}.excludeFromPalette`).toBe(expected.has(k))
    }
  })
})
