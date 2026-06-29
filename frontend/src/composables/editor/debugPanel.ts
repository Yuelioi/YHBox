import type { DebugSessionState, DebugTokenSummary, DebugWarning, DebugRunError } from '@/stores/execution'

export type DebugPanelTone = 'neutral' | 'primary' | 'warning' | 'error' | 'success'

export interface DebugPanelSummary {
  visible: boolean
  active: boolean
  status: string
  statusKey: string
  tone: DebugPanelTone
  focusNodeID: string
  focusNodeKind: string
  focusRoleKey: string
  lastNodeID: string
  lastNodeKind: string
  lastExit: string
  queue: DebugTokenSummary[]
  queueCount: number
  warnings: DebugWarning[]
  error: DebugRunError | null
}

const terminal = new Set(['finished', 'failed', 'stopped'])

export function summarizeDebugSession(state: Partial<DebugSessionState> | null | undefined): DebugPanelSummary {
  const status = String(state?.status ?? '')
  const active = !!state?.sessionId && !terminal.has(status)
  const queue = Array.isArray(state?.queue) ? state.queue : []
  const error = state?.error ?? null
  const warnings = Array.isArray(state?.warnings) ? state.warnings : []

  const runningNodeID = String(state?.runningNodeId ?? '')
  const runningNodeKind = String(state?.runningNodeKind ?? '')
  const currentNodeID = String(state?.currentNodeId ?? '')
  const currentNodeKind = String(state?.currentNodeKind ?? '')
  const lastNodeID = String(state?.lastNodeId ?? '')
  const lastNodeKind = String(state?.lastNodeKind ?? '')

  let focusNodeID = ''
  let focusNodeKind = ''
  let focusRoleKey = 'editor.debug_panel.focus_none'
  if (runningNodeID) {
    focusNodeID = runningNodeID
    focusNodeKind = runningNodeKind
    focusRoleKey = 'editor.debug_panel.focus_running'
  } else if (currentNodeID) {
    focusNodeID = currentNodeID
    focusNodeKind = currentNodeKind
    focusRoleKey = 'editor.debug_panel.focus_next'
  } else if (status === 'failed' && lastNodeID) {
    focusNodeID = lastNodeID
    focusNodeKind = lastNodeKind
    focusRoleKey = 'editor.debug_panel.focus_failed'
  }

  return {
    visible: active || status === 'failed' || status === 'finished',
    active,
    status,
    statusKey: `editor.debug_panel.status.${status || 'idle'}`,
    tone: toneForStatus(status),
    focusNodeID,
    focusNodeKind,
    focusRoleKey,
    lastNodeID,
    lastNodeKind,
    lastExit: String(state?.lastExit ?? ''),
    queue,
    queueCount: queue.length,
    warnings,
    error,
  }
}

function toneForStatus(status: string): DebugPanelTone {
  switch (status) {
    case 'paused':
      return 'primary'
    case 'stepping':
    case 'running':
    case 'pause_requested':
      return 'warning'
    case 'finished':
      return 'success'
    case 'failed':
      return 'error'
    default:
      return 'neutral'
  }
}
