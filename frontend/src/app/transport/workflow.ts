import { Dialogs, Events } from '@wailsio/runtime'
import * as WorkflowService from '@bindings/github.com/yottaapp/yotta/internal/services/workflow/service.js'
import type {
  CompileView,
  BundleExportResult,
  BundleInfoView,
  DeleteSourcePreview,
  DeleteSourceRequest,
  DeleteSourceResult,
  PatchView,
  RunView,
  SourcePage,
  SourceQuery,
  SourceView,
  StartRunView,
} from '@bindings/github.com/yottaapp/yotta/internal/services/workflow/models.js'
import type {
  DebugBreakpoint,
  DebugSnapshot,
} from '@bindings/github.com/yottaapp/yotta/internal/workflow/compiler/models.js'
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

export interface DebugChangedEvent {
  runId: string
  snapshot: DebugSnapshot
}

export interface WorkflowTransport {
  listSources(): Promise<SourceView[]>
  querySources(query: SourceQuery): Promise<SourcePage>
  previewDeleteSources(workflowIds: string[]): Promise<DeleteSourcePreview[]>
  deleteSources(requests: DeleteSourceRequest[]): Promise<DeleteSourceResult[]>
  createSource(name: string): Promise<SourceView>
  chooseSourceBundle(): Promise<string>
  chooseSourceBundleDestination(filename: string): Promise<string>
  chooseSourceBundleDirectory(): Promise<string>
  inspectSourceBundle(path: string): Promise<BundleInfoView>
  importSourceBundle(path: string): Promise<SourceView>
  replaceSourceFromBundle(
    path: string,
    workflowId: string,
    revision: number,
    sourceHash: string,
  ): Promise<SourceView>
  exportSourceBundle(workflowId: string, destination: string): Promise<BundleExportResult>
  exportSourceBundles(workflowIds: string[], directory: string): Promise<BundleExportResult[]>
  getSource(workflowId: string): Promise<SourceView>
  applyPatch(
    workflowId: string,
    baseRevision: number,
    commands: WorkflowPatchCommand[],
  ): Promise<PatchView>
  compileSource(workflowId: string): Promise<CompileView>
  startRun(workflowId: string): Promise<StartRunView>
  startDebugRun(workflowId: string, breakpoints: DebugBreakpoint[]): Promise<StartRunView>
  getDebugSnapshot(runId: string): Promise<DebugSnapshot>
  controlDebugRun(runId: string, action: 'continue' | 'pause' | 'step'): Promise<DebugSnapshot>
  setDebugBreakpoints(runId: string, breakpoints: DebugBreakpoint[]): Promise<DebugSnapshot>
  cancelRun(runId: string): Promise<RunView>
  cancelAllRuns(): Promise<void>
  getRunTimeline(runId: string): Promise<RunView>
  getAuthoringProjection(): Promise<string>
}

export const workflowTransport: WorkflowTransport = {
  listSources: () => WorkflowService.ListSources(),
  querySources: (query) => WorkflowService.QuerySources(query),
  previewDeleteSources: (workflowIds) => WorkflowService.PreviewDeleteSources(workflowIds),
  deleteSources: (requests) => WorkflowService.DeleteSources(requests),
  createSource: (name) => WorkflowService.CreateSource(name),
  chooseSourceBundle: () =>
    Dialogs.OpenFile({
      Title: 'Import Workflow Source',
      AllowsMultipleSelection: false,
      CanChooseFiles: true,
      CanChooseDirectories: false,
      Filters: [{ DisplayName: 'Yotta Workflow Source', Pattern: '*.yotta-workflow' }],
    }),
  chooseSourceBundleDestination: (filename) =>
    Dialogs.SaveFile({
      Title: 'Export Workflow Source',
      Filename: filename,
      CanChooseFiles: true,
      CanChooseDirectories: false,
      Filters: [{ DisplayName: 'Yotta Workflow Source', Pattern: '*.yotta-workflow' }],
    }),
  chooseSourceBundleDirectory: () =>
    Dialogs.OpenFile({
      Title: 'Export Workflow Sources',
      AllowsMultipleSelection: false,
      CanChooseFiles: false,
      CanChooseDirectories: true,
      CanCreateDirectories: true,
    }),
  inspectSourceBundle: (path) => WorkflowService.InspectSourceBundle(path),
  importSourceBundle: (path) => WorkflowService.ImportSourceBundle(path),
  replaceSourceFromBundle: (path, workflowId, revision, sourceHash) =>
    WorkflowService.ReplaceSourceFromBundle(path, workflowId, revision, sourceHash),
  exportSourceBundle: (workflowId, destination) =>
    WorkflowService.ExportSourceBundle(workflowId, destination),
  exportSourceBundles: (workflowIds, directory) =>
    WorkflowService.ExportSourceBundles(workflowIds, directory),
  getSource: (workflowId) => WorkflowService.GetSource(workflowId),
  applyPatch: (workflowId, baseRevision, commands) =>
    WorkflowService.ApplyPatch(
      workflowId,
      baseRevision,
      commands as Parameters<typeof WorkflowService.ApplyPatch>[2],
    ),
  compileSource: (workflowId) => WorkflowService.CompileSource(workflowId),
  startRun: (workflowId) => WorkflowService.StartRun(workflowId),
  startDebugRun: (workflowId, breakpoints) =>
    WorkflowService.StartDebugRun(workflowId, breakpoints),
  getDebugSnapshot: (runId) => WorkflowService.GetDebugSnapshot(runId),
  controlDebugRun: (runId, action) => WorkflowService.ControlDebugRun(runId, action),
  setDebugBreakpoints: (runId, breakpoints) =>
    WorkflowService.SetDebugBreakpoints(runId, breakpoints),
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

export function onDebugChanged(listener: (event: DebugChangedEvent) => void): () => void {
  return Events.On('debug:changed', (event: { data?: unknown }) => {
    const payload =
      Array.isArray(event.data) && event.data.length === 1 ? event.data[0] : event.data
    if (!isDebugChangedEvent(payload)) return
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

function isDebugChangedEvent(value: unknown): value is DebugChangedEvent {
  if (typeof value !== 'object' || value === null) return false
  const candidate = value as Record<string, unknown>
  if (
    typeof candidate.runId !== 'string' ||
    typeof candidate.snapshot !== 'object' ||
    candidate.snapshot === null
  )
    return false
  const snapshot = candidate.snapshot as Record<string, unknown>
  return typeof snapshot.status === 'string' && typeof snapshot.generation === 'number'
}

export type {
  BundleExportResult,
  BundleInfoView,
  CompileView,
  DeleteSourcePreview,
  DeleteSourceRequest,
  DeleteSourceResult,
  PatchView,
  RunView,
  SourcePage,
  SourceQuery,
  SourceView,
  StartRunView,
  WorkflowJSONValue,
  WorkflowPatchCommand,
  DebugBreakpoint,
  DebugSnapshot,
}
