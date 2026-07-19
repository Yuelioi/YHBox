import { Dialogs, Events } from '@wailsio/runtime'
import * as WorkflowService from '@bindings/github.com/yottaapp/yotta/internal/services/workflow/service.js'
import type {
  BatchUpdateSourceMetadataRequest,
  BatchUpdateSourceMetadataResult,
  CompileView,
  CreateSourceRequest,
  BundleExportResult,
  BundleInfoView,
  DeleteSourcePreview,
  DeleteSourceRequest,
  DeleteSourceResult,
  PatchView,
  RunView,
  SourcePage,
  SourceQuery,
  SourceRecoveryView,
  SourceView,
  StartRunView,
  UpdateSourceMetadataRequest,
} from '@bindings/github.com/yottaapp/yotta/internal/services/workflow/models.js'
import type {
  DebugBreakpoint,
  DebugSnapshot,
} from '@bindings/github.com/yottaapp/yotta/internal/workflow/compiler/models.js'
import type {
  Command as WorkflowPatchCommand,
  JSONValue as WorkflowJSONValue,
} from '../../../../contracts/workflow/3.1/authoring-patch'
import { callRPC, invoke } from '@/lib/invoke'

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
  listSourceRecoveries(): Promise<SourceRecoveryView[]>
  repairSourceRecovery(recoveryId: string, sourceJson: string): Promise<SourceView>
  deleteSourceRecovery(recoveryId: string): Promise<void>
  previewDeleteSources(workflowIds: string[]): Promise<DeleteSourcePreview[]>
  deleteSources(requests: DeleteSourceRequest[]): Promise<DeleteSourceResult[]>
  createSource(name: string): Promise<SourceView>
  createSourceWithMetadata(request: CreateSourceRequest): Promise<SourceView>
  updateSourceMetadata(
    workflowId: string,
    baseRevision: number,
    request: UpdateSourceMetadataRequest,
  ): Promise<SourceView>
  batchUpdateSourceMetadata(
    requests: BatchUpdateSourceMetadataRequest[],
  ): Promise<BatchUpdateSourceMetadataResult[]>
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
  getRunTimelinePage(runId: string, page: number, pageSize: number): Promise<RunView>
  getAuthoringProjection(): Promise<string>
}

export const workflowTransport: WorkflowTransport = {
  listSources: () => invoke(WorkflowService.ListSources),
  querySources: (query) => invoke(WorkflowService.QuerySources, query),
  listSourceRecoveries: () => invoke(WorkflowService.ListSourceRecoveries),
  repairSourceRecovery: (recoveryId, sourceJson) =>
    invoke(WorkflowService.RepairSourceRecovery, recoveryId, sourceJson),
  deleteSourceRecovery: (recoveryId) => invoke(WorkflowService.DeleteSourceRecovery, recoveryId),
  previewDeleteSources: (workflowIds) => invoke(WorkflowService.PreviewDeleteSources, workflowIds),
  deleteSources: (requests) => invoke(WorkflowService.DeleteSources, requests),
  createSource: (name) => invoke(WorkflowService.CreateSource, name),
  createSourceWithMetadata: (request) => invoke(WorkflowService.CreateSourceWithMetadata, request),
  updateSourceMetadata: (workflowId, baseRevision, request) =>
    invoke(WorkflowService.UpdateSourceMetadata, workflowId, baseRevision, request),
  batchUpdateSourceMetadata: (requests) =>
    invoke(WorkflowService.BatchUpdateSourceMetadata, requests),
  chooseSourceBundle: () =>
    callRPC('workflow.chooseSourceBundle', () =>
      Dialogs.OpenFile({
        Title: 'Import Workflow Source',
        AllowsMultipleSelection: false,
        CanChooseFiles: true,
        CanChooseDirectories: false,
        Filters: [{ DisplayName: 'Yotta Workflow Source', Pattern: '*.yotta-workflow' }],
      }),
    ),
  chooseSourceBundleDestination: (filename) =>
    callRPC('workflow.chooseSourceBundleDestination', () =>
      Dialogs.SaveFile({
        Title: 'Export Workflow Source',
        Filename: filename,
        CanChooseFiles: true,
        CanChooseDirectories: false,
        Filters: [{ DisplayName: 'Yotta Workflow Source', Pattern: '*.yotta-workflow' }],
      }),
    ),
  chooseSourceBundleDirectory: () =>
    callRPC('workflow.chooseSourceBundleDirectory', () =>
      Dialogs.OpenFile({
        Title: 'Export Workflow Sources',
        AllowsMultipleSelection: false,
        CanChooseFiles: false,
        CanChooseDirectories: true,
        CanCreateDirectories: true,
      }),
    ),
  inspectSourceBundle: (path) => invoke(WorkflowService.InspectSourceBundle, path),
  importSourceBundle: (path) => invoke(WorkflowService.ImportSourceBundle, path),
  replaceSourceFromBundle: (path, workflowId, revision, sourceHash) =>
    invoke(WorkflowService.ReplaceSourceFromBundle, path, workflowId, revision, sourceHash),
  exportSourceBundle: (workflowId, destination) =>
    invoke(WorkflowService.ExportSourceBundle, workflowId, destination),
  exportSourceBundles: (workflowIds, directory) =>
    invoke(WorkflowService.ExportSourceBundles, workflowIds, directory),
  getSource: (workflowId) => invoke(WorkflowService.GetSource, workflowId),
  applyPatch: (workflowId, baseRevision, commands) =>
    invoke(
      WorkflowService.ApplyPatch,
      workflowId,
      baseRevision,
      commands as Parameters<typeof WorkflowService.ApplyPatch>[2],
    ),
  compileSource: (workflowId) => invoke(WorkflowService.CompileSource, workflowId),
  startRun: (workflowId) => invoke(WorkflowService.StartRun, workflowId),
  startDebugRun: (workflowId, breakpoints) =>
    invoke(WorkflowService.StartDebugRun, workflowId, breakpoints),
  getDebugSnapshot: (runId) => invoke(WorkflowService.GetDebugSnapshot, runId),
  controlDebugRun: (runId, action) => invoke(WorkflowService.ControlDebugRun, runId, action),
  setDebugBreakpoints: (runId, breakpoints) =>
    invoke(WorkflowService.SetDebugBreakpoints, runId, breakpoints),
  cancelRun: (runId) => invoke(WorkflowService.CancelRun, runId),
  cancelAllRuns: () => invoke(WorkflowService.CancelAllRuns),
  getRunTimeline: (runId) => invoke(WorkflowService.GetRunTimeline, runId),
  getRunTimelinePage: (runId, page, pageSize) =>
    invoke(WorkflowService.GetRunTimelinePage, runId, page, pageSize),
  getAuthoringProjection: () => invoke(WorkflowService.GetAuthoringProjection),
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
  BatchUpdateSourceMetadataRequest,
  BatchUpdateSourceMetadataResult,
  BundleExportResult,
  BundleInfoView,
  CompileView,
  CreateSourceRequest,
  DeleteSourcePreview,
  DeleteSourceRequest,
  DeleteSourceResult,
  PatchView,
  RunView,
  SourcePage,
  SourceQuery,
  SourceRecoveryView,
  SourceView,
  StartRunView,
  UpdateSourceMetadataRequest,
  WorkflowJSONValue,
  WorkflowPatchCommand,
  DebugBreakpoint,
  DebugSnapshot,
}
