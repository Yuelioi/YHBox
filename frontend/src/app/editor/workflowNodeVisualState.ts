import type { NodeRunStatus } from './runTrace'
import type { DiagnosticSeverity } from './workflowDiagnostics'

export interface WorkflowNodeVisualStateInput {
  selected?: boolean
  runStatus?: NodeRunStatus
  debugCurrent?: boolean
  diagnosticSeverity?: DiagnosticSeverity
}

export interface WorkflowNodeVisualState {
  surfaceClasses: string
  executionTone: 'primary' | 'success' | 'error' | 'warning' | null
  executionStripeClasses: string
  debugCurrent: boolean
  diagnosticTone: DiagnosticSeverity | null
}

export function workflowNodeVisualState(
  input: WorkflowNodeVisualStateInput,
): WorkflowNodeVisualState {
  const executionTone = runTone(input.runStatus)
  return {
    surfaceClasses: [
      'border-default',
      input.selected &&
        'ring-2 ring-primary/80 ring-offset-2 ring-offset-default shadow-primary/10',
    ]
      .filter(Boolean)
      .join(' '),
    executionTone,
    executionStripeClasses: runStripe(input.runStatus),
    debugCurrent: Boolean(input.debugCurrent),
    diagnosticTone: input.diagnosticSeverity ?? null,
  }
}

function runStripe(status?: NodeRunStatus): string {
  if (status === 'succeeded') return 'bg-success'
  if (status === 'failed') return 'bg-error'
  if (status === 'cancelled' || status === 'routed') return 'bg-warning'
  if (status === 'running') return 'bg-primary'
  return ''
}

function runTone(status?: NodeRunStatus): WorkflowNodeVisualState['executionTone'] {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'error'
  if (status === 'cancelled' || status === 'routed') return 'warning'
  if (status === 'running') return 'primary'
  return null
}
