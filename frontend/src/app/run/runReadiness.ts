import { i18n } from '@/i18n'
import type { WorkflowStartRunView } from '@/app/transport/workflow'

export interface RunReadinessLike {
  state: string
  code?: string
  slot?: string
  nodeId?: string
  requirementId?: string
}

export type RunStartOutcome =
  | { state: 'started'; runId: string }
  | {
      state:
        | 'workflow-invalid'
        | 'target-required'
        | 'credential-required'
        | 'environment-unavailable'
        | 'not-started'
        | 'failed'
      code?: string
      slot?: string
      nodeId?: string
      requirementId?: string
    }

export function runStartOutcome(started: WorkflowStartRunView): RunStartOutcome {
  if (started.run) return { state: 'started', runId: started.run.runId }
  const readiness = started.readiness
  if (readiness && readiness.state !== 'started' && readiness.state !== 'failed') {
    return readinessOutcome(readiness)
  }
  if (readiness?.state === 'failed') {
    return readinessOutcome(readiness)
  }
  const diagnostic = started.diagnostics.find((item) => item.severity === 'error')
  if (diagnostic) {
    return { state: 'workflow-invalid', code: diagnostic.code, nodeId: diagnostic.nodeId }
  }
  return { state: 'not-started' }
}

export function readinessOutcome(
  readiness?: RunReadinessLike | null,
): Exclude<RunStartOutcome, { state: 'started' }> {
  if (!readiness) return { state: 'not-started' }
  return {
    state: normalizeReadinessState(readiness.state),
    code: readiness.code,
    slot: readiness.slot,
    nodeId: readiness.nodeId,
    requirementId: readiness.requirementId,
  }
}

export function runReadinessMessage(
  outcome: Exclude<RunStartOutcome, { state: 'started' }>,
): string {
  const t = i18n.global.t
  const te = i18n.global.te
  if (
    outcome.requirementId === 'model' &&
    (outcome.code === 'admission.target_unavailable' ||
      outcome.code === 'admission.target_ambiguous')
  ) {
    return t('workflow.run_readiness.ai_model_unavailable', { slot: outcome.slot ?? '' })
  }
  if (
    outcome.requirementId === 'model' &&
    (outcome.code === 'admission.credential_unavailable' ||
      outcome.code === 'admission.credential_ambiguous')
  ) {
    return t('workflow.run_readiness.ai_credential_unavailable', { slot: outcome.slot ?? '' })
  }
  if (outcome.code && te(`error.${outcome.code}`)) {
    return t(`error.${outcome.code}`, {
      slot: outcome.slot ?? '',
      nodeId: outcome.nodeId ?? '',
    })
  }
  return t(`workflow.run_readiness.${outcome.state}`)
}

function normalizeReadinessState(
  state: string,
): Exclude<RunStartOutcome, { state: 'started' }>['state'] {
  switch (state) {
    case 'workflow-invalid':
    case 'target-required':
    case 'credential-required':
    case 'environment-unavailable':
    case 'failed':
      return state
    default:
      return 'not-started'
  }
}
