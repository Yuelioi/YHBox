import { Events } from '@wailsio/runtime'
import * as WorkflowService from '@bindings/github.com/yottaapp/yotta/internal/services/workflow31/service.js'
import type {
  CompileView,
  RunView,
  SourceView,
  StartRunView,
} from '@bindings/github.com/yottaapp/yotta/internal/services/workflow31/models.js'

export interface RunChangedEvent {
  runId: string
  status: string
  generation: number
  recordDigest: string
  failed?: boolean
}

export interface WorkflowTransport {
  listSources(): Promise<SourceView[]>
  createSource(name: string): Promise<SourceView>
  getSource(workflowId: string): Promise<SourceView>
  saveSource(sourceJson: string, baseRevision: number): Promise<SourceView>
  compileDraft(sourceJson: string): Promise<CompileView>
  startRun(workflowId: string): Promise<StartRunView>
  cancelRun(runId: string): Promise<RunView>
  cancelAllRuns(): Promise<void>
  getRunTimeline(runId: string): Promise<RunView>
  getAuthoringProjection(): Promise<string>
}

export const workflowTransport: WorkflowTransport = {
  listSources: () => WorkflowService.ListSources(),
  createSource: (name) => WorkflowService.CreateSource(name),
  getSource: (workflowId) => WorkflowService.GetSource(workflowId),
  saveSource: (sourceJson, baseRevision) => WorkflowService.SaveSource(sourceJson, baseRevision),
  compileDraft: (sourceJson) => WorkflowService.CompileDraft(sourceJson),
  startRun: (workflowId) => WorkflowService.StartRun(workflowId),
  cancelRun: (runId) => WorkflowService.CancelRun(runId),
  cancelAllRuns: () => WorkflowService.CancelAllRuns(),
  getRunTimeline: (runId) => WorkflowService.GetRunTimeline(runId),
  getAuthoringProjection: () => WorkflowService.GetAuthoringProjection(),
}

export function onRunChanged(listener: (event: RunChangedEvent) => void): () => void {
  return Events.On('run:changed', (event: { data?: unknown }) => {
    const payload =
      Array.isArray(event.data) && event.data.length === 1 ? event.data[0] : event.data
    if (!isRunChangedEvent(payload)) return
    listener(payload)
  })
}

function isRunChangedEvent(value: unknown): value is RunChangedEvent {
  if (typeof value !== 'object' || value === null) return false
  const candidate = value as Record<string, unknown>
  return (
    typeof candidate.runId === 'string' &&
    typeof candidate.status === 'string' &&
    typeof candidate.generation === 'number' &&
    typeof candidate.recordDigest === 'string'
  )
}

export type { CompileView, RunView, SourceView, StartRunView }
