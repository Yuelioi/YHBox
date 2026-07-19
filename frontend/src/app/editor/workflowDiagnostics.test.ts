import { describe, expect, it } from 'vitest'
import {
  diagnosticFieldLocation,
  groupDiagnostics,
  nodeDiagnosticSeverities,
  type WorkflowDiagnostic,
} from './workflowDiagnostics'

describe('workflow diagnostics', () => {
  it('groups diagnostics by severity without changing compiler order', () => {
    const diagnostics = [
      diagnostic('warning', 'W-1'),
      diagnostic('error', 'E-1'),
      diagnostic('warning', 'W-2'),
    ]
    expect(
      groupDiagnostics(diagnostics).map((group) => [
        group.severity,
        group.diagnostics.map((item) => item.code),
      ]),
    ).toEqual([
      ['error', ['E-1']],
      ['warning', ['W-1', 'W-2']],
    ])
  })

  it('shows a node-relative field or port location', () => {
    expect(
      diagnosticFieldLocation({
        ...diagnostic('error', 'MISSING_INPUT_BINDING'),
        fieldPath: ['graphs', '0', 'nodes', '2', 'bindings', 'target'],
      }),
    ).toBe('bindings › target')
  })

  it('projects the strongest diagnostic severity onto nodes in the active graph', () => {
    const severities = nodeDiagnosticSeverities(
      [
        diagnostic('warning', 'W-1', 'child', 'warned'),
        diagnostic('info', 'I-1', 'child', 'warned'),
        diagnostic('error', 'E-1', 'child', 'broken'),
        diagnostic('error', 'E-2', 'main', 'outside'),
      ],
      'child',
    )

    expect(severities).toEqual(
      new Map([
        ['warned', 'warning'],
        ['broken', 'error'],
      ]),
    )
  })
})

function diagnostic(
  severity: 'error' | 'warning' | 'info',
  code: string,
  graphId = '',
  nodeId = '',
): WorkflowDiagnostic {
  return {
    severity,
    code,
    graphPath: graphId ? [graphId] : [],
    nodeId,
  } as WorkflowDiagnostic
}
