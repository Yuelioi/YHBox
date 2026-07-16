import { Events } from '@wailsio/runtime'
import * as WorkflowService from '@bindings/github.com/yottaapp/yotta/internal/services/workflow/service.js'
import type {
  CompileView,
  PatchView,
  RunView,
  SourceView,
  StartRunView,
} from '@bindings/github.com/yottaapp/yotta/internal/services/workflow/models.js'
import type {
  Command as WorkflowPatchCommand,
  JSONValue as WorkflowJSONValue,
} from '../../../../contracts/workflow/3.1/authoring-patch'

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
  applyPatch(
    workflowId: string,
    baseRevision: number,
    commands: WorkflowPatchCommand[],
  ): Promise<PatchView>
  compileSource(workflowId: string): Promise<CompileView>
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
  applyPatch: (workflowId, baseRevision, commands) =>
    WorkflowService.ApplyPatch(
      workflowId,
      baseRevision,
      commands as Parameters<typeof WorkflowService.ApplyPatch>[2],
    ),
  compileSource: (workflowId) => WorkflowService.CompileSource(workflowId),
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

export type {
  CompileView,
  PatchView,
  RunView,
  SourceView,
  StartRunView,
  WorkflowJSONValue,
  WorkflowPatchCommand,
}
