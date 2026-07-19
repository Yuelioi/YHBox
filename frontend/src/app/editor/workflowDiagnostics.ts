import type { CompileView } from '@/app/transport/workflow'

export type WorkflowDiagnostic = CompileView['diagnostics'][number]
export type DiagnosticSeverity = 'error' | 'warning' | 'info'

export interface DiagnosticGroup {
  severity: DiagnosticSeverity
  diagnostics: WorkflowDiagnostic[]
}

const severities: DiagnosticSeverity[] = ['error', 'warning', 'info']

export function groupDiagnostics(diagnostics: readonly WorkflowDiagnostic[]): DiagnosticGroup[] {
  return severities.flatMap((severity) => {
    const matching = diagnostics.filter((diagnostic) => diagnostic.severity === severity)
    return matching.length ? [{ severity, diagnostics: matching }] : []
  })
}

export function diagnosticFieldLocation(diagnostic: WorkflowDiagnostic): string {
  if (!diagnostic.fieldPath?.length) return ''
  const nodeIndex = diagnostic.fieldPath.lastIndexOf('nodes')
  const relative = nodeIndex >= 0 ? diagnostic.fieldPath.slice(nodeIndex + 2) : diagnostic.fieldPath
  return relative.join(' › ')
}

export function nodeDiagnosticSeverities(
  diagnostics: readonly WorkflowDiagnostic[],
  graphId: string,
): Map<string, DiagnosticSeverity> {
  const result = new Map<string, DiagnosticSeverity>()
  for (const diagnostic of diagnostics) {
    if (!diagnostic.nodeId || diagnostic.graphPath?.at(-1) !== graphId) continue
    const current = result.get(diagnostic.nodeId)
    if (!current || severityRank(diagnostic.severity) < severityRank(current)) {
      result.set(diagnostic.nodeId, diagnostic.severity as DiagnosticSeverity)
    }
  }
  return result
}

function severityRank(severity: string): number {
  if (severity === 'error') return 0
  if (severity === 'warning') return 1
  return 2
}
