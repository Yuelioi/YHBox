// 所有 wails3 对接走这一层 —— stores / views / events.ts 都通过 backend.xxx.yyy(...) 调用，
// 不直接 import bindings 或 @wailsio/runtime。理由：wails3 alpha API 漂移时只改这一个文件。

import { Dialogs, Events } from '@wailsio/runtime'
import * as SettingsService from '@bindings/github.com/yottaapp/yotta/internal/services/settingsservice.js'
import * as HotkeyService from '@bindings/github.com/yottaapp/yotta/internal/hotkey/hotkeyservice.js'
import * as ScheduleService from '@bindings/github.com/yottaapp/yotta/internal/services/schedule/service.js'
import * as AssetService from '@bindings/github.com/yottaapp/yotta/internal/services/asset/service.js'
import * as CalibrationService from '@bindings/github.com/yottaapp/yotta/internal/services/calibration/service.js'
import * as ToolsService from '@bindings/github.com/yottaapp/yotta/internal/services/tools/service.js'
import * as AppInfoService from '@bindings/github.com/yottaapp/yotta/internal/services/appinfoservice.js'
import * as RecordingService from '@bindings/github.com/yottaapp/yotta/internal/services/recording/service.js'
import * as ClipService from '@bindings/github.com/yottaapp/yotta/internal/services/inputclip/service.js'
import * as MacroService from '@bindings/github.com/yottaapp/yotta/internal/services/macro/service.js'
import * as AIService from '@bindings/github.com/yottaapp/yotta/internal/services/aiservice.js'
import * as NetworkService from '@bindings/github.com/yottaapp/yotta/internal/services/networkservice.js'
import * as ApplicationService from '@bindings/github.com/yottaapp/yotta/internal/services/applicationservice.js'
import * as AutomationService from '@bindings/github.com/yottaapp/yotta/internal/services/automationservice.js'
import { AIModelSettings as AIModelSettingsBinding } from '@bindings/github.com/yottaapp/yotta/internal/services/models.js'
import {
  EvalReportArtifact as EvalReportArtifactBinding,
  EvaluationStatus as EvaluationStatusBinding,
  ProfileCapabilities as ProfileCapabilitiesBinding,
  ProviderKind as ProviderKindBinding,
  TokenPricing as TokenPricingBinding,
} from '@bindings/github.com/yottaapp/yotta/internal/ai/models.js'
import type { Schedule as ScheduleModel } from '@bindings/github.com/yottaapp/yotta/internal/services/schedule/models.js'
import { BlobRef as BlobRefBinding } from '@bindings/github.com/yottaapp/yotta/internal/blob/models.js'
import { callRPC, invoke } from './invoke'
import * as E from '@/constants/events'

// 事件 payload 类型（跟 Go events.go 一一对应；wails3 bindings 也会产 .d.ts，
// 这里手写一份用于 store 引用更稳，避免 bindings 路径变化）
export interface BackendLogEntry {
  time: string
  level: string
  source: 'SYS' | 'WF'
  tag?: string
  message: string
  fields?: unknown
  graphId?: string
  nodeId?: string
  invocationId?: string
  attempt?: number
}
export interface LogBatchEvent {
  seq: number
  entries: BackendLogEntry[]
  dropped?: number
}
// HotkeyEntry 跟 Go services.HotkeyEntry 对齐。Normalized 字段后端故意不导出 — 前端不依赖
// canonicalization 规则。冲突 / reserved / 验证错误通过 error message 前缀 [conflict] /
// [reserved] / [invalid] 区分。
//
// label 是 i18n key string (FE 走 t(entry.label, entry.labelParams)). labelParams
// 装 vue-i18n named interpolation (工作流名 / 计划名等动态), backend Register 时填.
export interface HotkeyEntry {
  key: string
  source: 'system' | 'action' | 'schedule' | 'editor' | 'recording'
  label: string
  labelParams?: Record<string, string>
  hotkeyStr: string
  status: 'active' | 'unbound' | 'failed'
  lastError: string
  readonlyReason: string
  mechanism?: 'os-global' | 'editor-inapp' | 'll-hook'
}

export type Schedule = ScheduleModel

// AssetSummary 全局资产列表项 — 对应后端 asset.AssetSummary.
// 键 = guid (稳定 UUID), 不再是 namespace.name key.
export interface AssetSummary {
  guid: string
  kind: 'template' | 'macro' | 'clip'
  name: string
  description?: string
  category?: string
  tags?: string[]
  variantCount: number
  variants: Array<{ resolution: [number, number]; blob: BlobRef }>
  blob?: BlobRef
  thumbnail?: BlobRef
  createdAt?: string
}

// AssetRecord 全局资产完整记录 — 对应后端 asset.AssetRecord.
export interface AssetRecord {
  guid: string
  kind: string
  name: string
  description?: string
  category?: string
  tags?: string[]
  origin: { kind: string; sourceID?: string }
  variants?: Array<{ resolution: number[]; bbox: number[]; blob: BlobRef }>
  blob?: BlobRef
  createdAt: string
}

export interface AssetQuery {
  search: string
  kind: string
  category: string
  tags: string[]
  sort: string
  page: number
  pageSize: number
  thumbnailBudget: number
  recentGUIDs: string[]
}

export interface AssetPage {
  items: AssetSummary[]
  total: number
  page: number
  pageSize: number
  revision: number
  categories: Array<{ value: string; count: number }>
  tags: Array<{ value: string; count: number }>
}

export interface AssetBinding {
  found: boolean
  guid: string
  kind: 'template' | 'macro' | 'clip' | ''
  name: string
  resolution: [number, number]
  blob: BlobRef
  matchCount: number
}

export interface AssetBatchResult {
  guid: string
  updated?: boolean
  deleted?: boolean
  error?: string
}

export interface AssetCleanupPreview {
  token: string
  candidateCount: number
  candidateBytes: number
  liveCount: number
  objectCount: number
}

export interface BlobRef {
  mediaType: string
  digest: string
  size: number
}

export type MacroActionKind =
  | 'key-down'
  | 'key-up'
  | 'mouse-down'
  | 'mouse-up'
  | 'click'
  | 'scroll'
  | 'sleep'

export interface MacroAction {
  id: string
  kind: MacroActionKind
  key?: string
  button?: 'left' | 'middle' | 'right'
  point?: { x: number; y: number; unit: 'ratio' }
  notches?: number
  durationUs?: number
}

export interface MacroDocument {
  schemaVersion: number
  baseResolution: [number, number]
  actions: MacroAction[]
}

export interface MacroAsset {
  id: string
  label: string
  description?: string
  category?: string
  tags?: string[]
  createdAt: string
  document: MacroDocument
  blob: BlobRef
}

export interface InputEvent {
  tUs: number
  seq: number
  type: number
  a: number
  b: number
  c: number
}

export interface InputEventPage {
  items: InputEvent[]
  total: number
  offset: number
  limit: number
}

export interface InputClipSummary {
  id: string
  label: string
  description?: string
  category?: string
  tags?: string[]
  durationUs: number
  createdAt: string
  meta: {
    recordingMode: 'simple' | 'precise'
    mouseMode: string
    baseResolution: [number, number]
    mouseCounts360: number
    stopHotkeyVK: number
  }
  eventCount: number
  blob: BlobRef
  tracks: Array<{
    kind: 'keyboard' | 'mouse-buttons' | 'absolute-motion' | 'relative-motion' | 'scroll'
    count: number
    firstUs: number
    lastUs: number
  }>
}

export type MacroSaveInput = Omit<MacroAsset, 'id' | 'createdAt' | 'blob'> & {
  id?: string
  createdAt?: string
  blob?: BlobRef
}

export interface BlobPreview {
  mediaType: string
  base64: string
  width: number
  height: number
}

export type AIProviderKind = 'openai-responses' | 'anthropic-messages'

export interface AIProfileCapabilities {
  structuredOutput: boolean
  toolCalling: boolean
  parallelTools: boolean
  background: boolean
  zeroRetention: boolean
}

export interface AIProfilePricing {
  inputMicrounitsPerMillion: number
  cacheReadMicrounitsPerMillion: number
  outputMicrounitsPerMillion: number
}

export interface AIEvaluationReport {
  digest?: string
  report?: unknown
}

// AIModelProfile carries installation metadata only. Stored API keys never
// cross the backend seam after they enter the OS credential manager.
export interface AIModelProfile {
  slot: string
  label: string
  provider: AIProviderKind
  endpoint: string
  allowLocalHttp: boolean
  model: string
  maxOutputTokens: number
  capabilities: AIProfileCapabilities
  pricing: AIProfilePricing
  evaluation: 'unverified' | 'approved' | 'rejected'
  evaluationSuite?: string
  evaluationReport?: AIEvaluationReport
  workflowConsent?: string
}

export interface AIProfileTestResult {
  ok: boolean
  provider: AIProviderKind | ''
  requestedModel: string
  resolvedModel: string
  finish: string
  failureClass?: string
  error?: string
}

export interface AIWorkflowChange {
  index: number
  kind: string
  target: string
  sensitive: boolean
}

export interface AIWorkflowCapabilityChange {
  capabilityId: string
  operations: string[]
  targetSlot: string
  credentialSlot?: string
}

export interface AIWorkflowTraceEvent {
  sequence: number
  kind: string
  occurredAt: string
  providerRequestId?: string
  facts: Record<string, string>
}

export interface AIWorkflowReview {
  reviewId: string
  status: 'proposed' | 'accepted' | 'rejected' | 'stale'
  workflowId: string
  baseRevision: number
  newRevision: number
  baseHash: string
  candidateHash: string
  input: { trustClass: string; digest: string; bytes: number }
  profileSubject: string
  promptManifest: string
  toolSet: string
  summary: string
  changes: AIWorkflowChange[]
  diagnostics: Array<{
    code: string
    severity: string
    graphId?: string
    nodeId?: string
    path?: string
    message?: string
  }>
  permissions: { added: AIWorkflowCapabilityChange[]; removed: AIWorkflowCapabilityChange[] }
  risks: string[]
  usage: {
    inputTokens: number
    outputTokens: number
    costMicrounits: number
    wallTimeMillis: number
    iterations: number
    toolCalls: number
    maxParallelism: number
  }
  trace: AIWorkflowTraceEvent[]
}

export interface HTTPOriginProfile {
  slot: string
  label: string
  origin: string
  allowPrivateNetwork: boolean
  responseByteLimit: number
  timeoutMilliseconds: number
  workflowConsent?: string
}

export interface InstalledApplicationProfile {
  slot: string
  label: string
  executable: string
  executableDigest: string
  arguments: string[]
  workflowConsent?: string
}

export interface DesktopAutomationTargetProfile {
  applicationSlot: string
  windowTitle: string
  windowTitleMatch: 'exact' | 'regex'
  windowSelection: 'unique' | 'topmost'
  windowClass: string
  inputBackend: 'sendinput' | 'postmessage'
  captureBackend: 'gdi' | 'wgc'
  mouseCounts360: number
  resolveTimeoutMilliseconds: number
}

export interface AndroidAutomationTargetProfile {
  adbSerial: string
  adbProduct: string
  adbModel: string
  adbDevice: string
  androidPackage: string
  resolveTimeoutMilliseconds: number
}

export interface BrowserAutomationTargetProfile {
  browserEndpoint: string
  browserTargetId: string
  browserWebSocketUrl: string
  browserTitle: string
  browserUrl: string
  resolveTimeoutMilliseconds: number
}

export interface InstalledAutomationTargetProfile {
  slot: string
  label: string
  targetKind: string
  adapterKind: string
  profileVersion: string
  profile:
    | DesktopAutomationTargetProfile
    | AndroidAutomationTargetProfile
    | BrowserAutomationTargetProfile
    | Record<string, unknown>
  workflowConsent?: string
}

export interface AndroidDeviceDescriptor {
  serial: string
  state: string
  product: string
  model: string
  device: string
  transportId: string
}

export interface AndroidAppDescriptor {
  package: string
  label: string
  foreground: boolean
}

export interface BrowserTargetDescriptor {
  id: string
  type: string
  title: string
  url: string
  webSocketDebuggerUrl: string
}

export interface AutomationTargetHealth {
  ok: boolean
  code: string
  message: string
}

export interface AutomationCapabilityDescriptor {
  capabilityId: string
  resourceKind: string
  operations: string[]
}

export interface AutomationProfileFieldDescriptor {
  id: string
  kind: string
  required: boolean
  options?: string[]
}

export interface AutomationTargetTypeDescriptor {
  targetKind: string
  adapterKind: string
  profileKind: string
  profileVersion: string
  hostAvailable: boolean
  capabilities: AutomationCapabilityDescriptor[]
  resourceKinds: string[]
  operations: string[]
  fields: AutomationProfileFieldDescriptor[]
  inputBackends: string[]
  captureBackends: string[]
  applicationIdentityKinds: string[]
}

export interface ExecutableInspection {
  executable: string
  digest: string
  size: number
}

function toAIModelSettingsBinding(profile: AIModelProfile): AIModelSettingsBinding {
  return new AIModelSettingsBinding({
    ...profile,
    provider: profile.provider as ProviderKindBinding,
    evaluation: profile.evaluation as EvaluationStatusBinding,
    capabilities: new ProfileCapabilitiesBinding(profile.capabilities),
    pricing: new TokenPricingBinding(profile.pricing),
    evaluationReport: profile.evaluationReport
      ? new EvalReportArtifactBinding(profile.evaluationReport)
      : undefined,
  })
}

export const backend = {
  settings: {
    get: () => invoke(SettingsService.Get),
    update: (patch: object) => invoke(SettingsService.Update, JSON.stringify(patch)),
  },
  ai: {
    testProfile: (profile: AIModelProfile) =>
      invoke(AIService.TestProfile, {
        profile: toAIModelSettingsBinding(profile),
      }) as Promise<AIProfileTestResult>,
    secretStatus: (slots: string[]) =>
      invoke(AIService.SecretStatus, slots) as Promise<Record<string, boolean>>,
    setAPIKey: (slot: string, apiKey: string) => invoke(AIService.SetAPIKey, slot, apiKey),
    deleteAPIKey: (slot: string) => invoke(AIService.DeleteAPIKey, slot),
    applyEvaluation: (slot: string, evidence: AIEvaluationReport) =>
      invoke(AIService.ApplyEvaluation, slot, new EvalReportArtifactBinding(evidence)),
    revokeEvaluation: (slot: string) => invoke(AIService.RevokeEvaluation, slot),
    grantWorkflowUse: (slot: string) => invoke(AIService.GrantWorkflowUse, slot) as Promise<string>,
    revokeWorkflowUse: (slot: string) => invoke(AIService.RevokeWorkflowUse, slot),
    proposeWorkflow: (
      slot: string,
      workflowId: string,
      baseRevision: number,
      instruction: string,
    ) =>
      invoke(
        AIService.ProposeWorkflow,
        slot,
        workflowId,
        baseRevision,
        instruction,
      ) as Promise<AIWorkflowReview>,
    acceptWorkflowProposal: (reviewId: string) =>
      invoke(AIService.AcceptWorkflowProposal, reviewId) as Promise<AIWorkflowReview>,
    rejectWorkflowProposal: (reviewId: string) =>
      invoke(AIService.RejectWorkflowProposal, reviewId) as Promise<AIWorkflowReview>,
    getWorkflowProposal: (reviewId: string) =>
      invoke(AIService.GetWorkflowProposal, reviewId) as Promise<AIWorkflowReview>,
  },
  network: {
    grantHTTPWorkflowConsent: (slot: string) =>
      invoke(NetworkService.GrantHTTPWorkflowConsent, slot) as Promise<string>,
    revokeHTTPWorkflowConsent: (slot: string) =>
      invoke(NetworkService.RevokeHTTPWorkflowConsent, slot),
  },
  applications: {
    pickExecutable: (title: string) =>
      callRPC('applications.pickExecutable', () =>
        Dialogs.OpenFile({
          Title: title,
          AllowsMultipleSelection: false,
          Filters: [{ DisplayName: 'Windows Application', Pattern: '*.exe' }],
        }),
      ) as Promise<string>,
    inspectExecutable: (path: string) =>
      invoke(ApplicationService.InspectExecutable, path) as Promise<ExecutableInspection>,
    grantWorkflowConsent: (slot: string) =>
      invoke(ApplicationService.GrantWorkflowConsent, slot) as Promise<string>,
    revokeWorkflowConsent: (slot: string) => invoke(ApplicationService.RevokeWorkflowConsent, slot),
  },
  automation: {
    listTargetTypes: () => invoke(AutomationService.ListTargetTypes),
    listADBDevices: () =>
      invoke(AutomationService.ListADBDevices) as Promise<AndroidDeviceDescriptor[]>,
    listAndroidApps: (serial: string) =>
      invoke(AutomationService.ListAndroidApps, serial) as Promise<AndroidAppDescriptor[]>,
    listBrowserTargets: (endpoint: string) =>
      invoke(AutomationService.ListBrowserTargets, endpoint) as Promise<BrowserTargetDescriptor[]>,
    checkTargetHealth: (slot: string) =>
      invoke(AutomationService.CheckTargetHealth, slot) as Promise<AutomationTargetHealth>,
    grantWorkflowConsent: (slot: string) =>
      invoke(AutomationService.GrantWorkflowConsent, slot) as Promise<string>,
    revokeWorkflowConsent: (slot: string) => invoke(AutomationService.RevokeWorkflowConsent, slot),
    grantAllWorkflowConsents: () => invoke(AutomationService.GrantAllWorkflowConsents),
    revokeAllWorkflowConsents: () => invoke(AutomationService.RevokeAllWorkflowConsents),
  },
  schedules: {
    list: () => invoke(ScheduleService.List),
    get: (id: string) => invoke(ScheduleService.Get, id),
    create: (name: string) => invoke(ScheduleService.Create, name),
    save: (sc: Schedule) => invoke(ScheduleService.Save, sc),
    update: (id: string, patchJSON: string) => invoke(ScheduleService.Update, id, patchJSON),
    delete_: (id: string) => invoke(ScheduleService.Delete, id),
  },
  assets: {
    // List 全局资产列表 (template + macro + clip), 无工作流级存储分支.
    list: () => invoke(AssetService.List),
    query: (query: AssetQuery) =>
      invoke(AssetService.QueryAssets, query) as unknown as Promise<AssetPage>,
    batchUpdateMeta: (requests: Array<{ guid: string; category: string; tags: string[] }>) =>
      invoke(AssetService.BatchUpdateMeta, requests) as Promise<AssetBatchResult[]>,
    batchDelete: (guids: string[]) =>
      invoke(AssetService.BatchDelete, guids) as Promise<AssetBatchResult[]>,
    previewCleanup: () => invoke(AssetService.PreviewCleanup) as Promise<AssetCleanupPreview>,
    commitCleanup: (token: string) => invoke(AssetService.CommitCleanup, token),
    // SaveTemplateCapture 截图存为新模板资产, 返 GUID. tags 截图时可选设标签.
    saveTemplateCapture: (
      dataURL: string,
      name: string,
      category: string,
      tags: string[],
      recRes: [number, number],
      region: [number, number, number, number],
    ) => invoke(AssetService.SaveTemplateCapture, dataURL, name, category, tags, recRes, region),
    // AddTemplateVariant 给已有资产加/换分辨率变体.
    addTemplateVariant: (
      guid: string,
      dataURL: string,
      recRes: [number, number],
      region: [number, number, number, number],
    ) => invoke(AssetService.AddTemplateVariant, guid, dataURL, recRes, region),
    // Get 单条资产完整记录 (含 variants[] — 详情页看分辨率档/元信息).
    get: (guid: string) => invoke(AssetService.Get, guid) as Promise<AssetRecord>,
    resolveBinding: (ref: BlobRef) =>
      invoke(AssetService.ResolveBinding, new BlobRefBinding(ref)) as Promise<AssetBinding>,
    previewBlob: (ref: BlobRef) =>
      invoke(AssetService.PreviewBlob, new BlobRefBinding(ref)) as Promise<BlobPreview>,
    delete_: (guid: string) => invoke(AssetService.Delete, guid),
    // UpdateMeta 改显示名 + 标签 (记录级元数据).
    updateMeta: (
      guid: string,
      name: string,
      description: string,
      category: string,
      tags: string[],
    ) => invoke(AssetService.UpdateMeta, guid, name, description, category, tags),
    // Capture one exact installed automation target for local authoring.
    capture: (targetSlot: string) => invoke(AssetService.Capture, targetSlot),
    // CurrentResolution 当前容器 Windows 窗口客户区分辨率 [宽,高]; 窗口没开/无容器上下文 → 静默返 undefined.
    // 不走 invoke: 浏览态窗口没开属正常, 不该弹 error toast.
    currentResolution: async (targetSlot: string): Promise<[number, number] | undefined> => {
      try {
        const r = await callRPC('assets.currentResolution', () =>
          AssetService.CurrentResolution(targetSlot),
        )
        return Array.isArray(r) && r.length === 2 ? [r[0], r[1]] : undefined
      } catch {
        return undefined
      }
    },
    // PickVariant 给定分辨率, 返推荐绑定的档位在 variants[] 里的下标 + 是否精确命中. 自动调用, 失败静默.
    pickVariant: async (
      guid: string,
      w: number,
      h: number,
    ): Promise<{ index: number; exact: boolean } | undefined> => {
      try {
        const r = await callRPC('assets.pickVariant', () => AssetService.PickVariant(guid, w, h))
        return { index: r.index, exact: r.exact }
      } catch {
        return undefined
      }
    },
    // RemoveVariant 删指定分辨率的单个变体档；失败抛 typed RPCError，由调用场景决定反馈。
    removeVariant: (guid: string, w: number, h: number) =>
      invoke(AssetService.RemoveVariant, guid, w, h),
  },
  hotkeys: {
    list: () => invoke(HotkeyService.List),
    update: (key: string, hotkeyStr: string) => invoke(HotkeyService.Update, key, hotkeyStr),
    // pause/resume：HotkeyCaptureInput 进入捕获模式时 pause，离开时 resume。
    // 否则 Win32 RegisterHotKey 在 OS 层拦截已注册组合，webview 收不到 keystroke。
    pause: () => invoke(HotkeyService.Pause),
    resume: () => invoke(HotkeyService.Resume),
    // useEditorHotkeys onActivated 时调 — 注册 webview in-app key 进 registry
    // (只挂可见性 + 冲突检查, 不占 OS RegisterHotKey).
    registerEditor: (key: string, label: string, hotkeyStr: string, readonlyReason: string) =>
      invoke(HotkeyService.RegisterEditor, key, label, hotkeyStr, readonlyReason),
    // useEditorHotkeys onDeactivated 时调 — 从 registry 摘 editor key.
    unregister: (key: string) => invoke(HotkeyService.Unregister, key),
    // 「重置默认」: 把内置热键 (强停/校准/录制停止/录制暂停) 恢复出厂默认。容器热键不动。
    resetSystemDefaults: () => invoke(HotkeyService.ResetSystemDefaults),
  },
  calibration: {
    start: () => invoke(CalibrationService.Start),
    stop: () => invoke(CalibrationService.Stop),
    status: () => invoke(CalibrationService.Status),
    startHotkeyWatch: () => invoke(CalibrationService.StartHotkeyWatch),
    stopHotkeyWatch: () => invoke(CalibrationService.StopHotkeyWatch),
  },
  appInfo: {
    info: () => invoke(AppInfoService.Info),
  },
  recording: {
    start: (args: { targetSlot: string; mode: 'simple' | 'precise' }) =>
      invoke(RecordingService.Start, args as any),
    stop: () => invoke(RecordingService.Stop),
    stopAsync: () => invoke(RecordingService.StopAsync),
    cancel: () => invoke(RecordingService.Cancel),
    finalize: (args: {
      pendingID: string
      label: string
      description: string
      category: string
      tags: string[]
      trimStartUs?: number
      trimEndUs?: number
      actions?: Array<{
        id: string
        kind: 'key-down' | 'key-up' | 'mouse-down' | 'mouse-up' | 'click' | 'scroll' | 'sleep'
        key?: string
        button?: 'left' | 'middle' | 'right'
        point?: { x: number; y: number; unit: 'ratio' }
        notches?: number
        durationUs?: number
      }>
    }) => invoke(RecordingService.Finalize, args as any),
    discard: (pendingID: string) => invoke(RecordingService.Discard, pendingID),
    pause: () => invoke(RecordingService.Pause),
    resume: () => invoke(RecordingService.Resume),
    validateTarget: (targetSlot: string) => invoke(RecordingService.ValidateTarget, targetSlot),
    getState: () => invoke(RecordingService.GetState),
    pendingEvents: (pendingID: string, offset: number, limit: number) =>
      invoke(RecordingService.PendingEvents, pendingID, offset, limit) as Promise<InputEventPage>,
  },
  // 全局 ClipService (main.go RegisterService(clipSvc); 资产全局化后无 lib/容器两套存储).
  // Exposes authoring metadata and the nominal content BlobRef; runtime does not call this RPC.
  clips: {
    list: () => invoke(ClipService.List),
    get: (id: string) => invoke(ClipService.Get, id),
    summary: (id: string) => invoke(ClipService.Summary, id) as Promise<InputClipSummary>,
    events: (id: string, offset: number, limit: number) =>
      invoke(ClipService.Events, id, offset, limit) as Promise<InputEventPage>,
    save: (clip: unknown) => invoke(ClipService.Save, clip as any),
    update: (id: string, label: string, description: string, category: string, tags: string[]) =>
      invoke(ClipService.Update, id, label, description, category, tags),
    delete_: (id: string) => invoke(ClipService.Delete, id),
  },
  macros: {
    get: (id: string) => invoke(MacroService.Get, id) as Promise<MacroAsset | null>,
    save: (value: MacroSaveInput) =>
      invoke(
        MacroService.Save,
        value as unknown as Parameters<typeof MacroService.Save>[0],
      ) as Promise<MacroAsset>,
    analyze: (document: MacroDocument) =>
      invoke(
        MacroService.Analyze,
        document as unknown as Parameters<typeof MacroService.Analyze>[0],
      ),
    delete_: (id: string) => invoke(MacroService.Delete, id),
  },
  tools: {
    mousePos: (targetSlot: string) => invoke(ToolsService.MousePos, targetSlot),
    pixelAt: (targetSlot: string) => invoke(ToolsService.PixelAt, targetSlot),
    openMouseHUD: (targetSlot: string) => invoke(ToolsService.OpenMouseHUD, targetSlot),
    openRecordingHUD: () => invoke(ToolsService.OpenRecordingHUD),
    closeRecordingHUD: () => invoke(ToolsService.CloseRecordingHUD),
    openCalibratorHUD: (id: string) => invoke(ToolsService.OpenCalibratorHUD, id),
    closeCalibratorHUD: () => invoke(ToolsService.CloseCalibratorHUD),
    openScreenPicker: (
      mode: 'point' | 'rect' | 'template_save' | 'template_recapture' | 'color',
      id: string,
      targetSlot = '',
      colorSpace = '',
      guid = '',
    ) => invoke(ToolsService.OpenScreenPicker, mode, id, targetSlot, colorSpace, guid),
    extractColorRange: (samples: { R: number; G: number; B: number }[], colorSpace: string) =>
      invoke(ToolsService.ExtractColorRange, samples, colorSpace),
    closePicker: (id: string) => invoke(ToolsService.ClosePicker, id),
    // Win32WindowTarget capture: 临时安装全局键盘钩子 (默认 F9 = 0x78), 用户在游戏窗口按下后
    // 走 'win32windowtarget:captured' event 回填. 取代旧同步 captureForegroundWindow
    // — 用户在游戏前台时根本点不到 Yotta 按钮.
    // 捕获键来源 = 后端读热键中心 tools.window-capture 绑定 (可在「快捷键」页 rebind)，不再 FE 传死。
    startWin32WindowTargetCapture: () => invoke(ToolsService.StartWin32WindowTargetCapture),
    cancelWin32WindowTargetCapture: (id: string) =>
      invoke(ToolsService.CancelWin32WindowTargetCapture, id),
    openLauncher: () => invoke(ToolsService.OpenLauncher),
    openLauncherSettings: () => invoke(ToolsService.OpenLauncherSettings),
    toggleLauncher: () => invoke(ToolsService.ToggleLauncher),
    hideLauncher: () => invoke(ToolsService.HideLauncher),
    setLauncherAlwaysOnTop: (on: boolean) => invoke(ToolsService.SetLauncherAlwaysOnTop, on),
    setLauncherSize: (width: number, height: number) =>
      invoke(ToolsService.SetLauncherSize, width, height),
  },
  events: {
    // 共享事件
    onLogBatch: (cb: (e: LogBatchEvent) => void) =>
      Events.On(E.EVENT_LOG_BATCH, (e: any) => cb(e?.data?.[0] ?? e?.data ?? e)),
    onHotkeyChanged: (cb: () => void) => Events.On('hotkey:changed', () => cb()),
    onSettingsChanged: (cb: () => void) => Events.On('settings:changed', () => cb()),
    onMainNavigate: (cb: (target: { path: string; section?: string }) => void) =>
      Events.On('main:navigate', (event: { data?: unknown }) => {
        const payload = Array.isArray(event.data) ? event.data[0] : event.data
        if (typeof payload !== 'object' || payload === null) return
        const target = payload as Record<string, unknown>
        if (typeof target.path !== 'string') return
        cb({
          path: target.path,
          ...(typeof target.section === 'string' ? { section: target.section } : {}),
        })
      }),
  },
}
