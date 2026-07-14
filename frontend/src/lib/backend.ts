// 所有 wails3 对接走这一层 —— stores / views / events.ts 都通过 backend.xxx.yyy(...) 调用，
// 不直接 import bindings 或 @wailsio/runtime。理由：wails3 alpha API 漂移时只改这一个文件。

import { Dialogs, Events } from '@wailsio/runtime'
import * as SettingsService from '@bindings/github.com/yottaapp/yotta/internal/services/settingsservice.js'
import * as HotkeyService from '@bindings/github.com/yottaapp/yotta/internal/hotkey/hotkeyservice.js'
import * as ContainerService from '@bindings/github.com/yottaapp/yotta/internal/services/container/service.js'
import * as ScheduleService from '@bindings/github.com/yottaapp/yotta/internal/services/schedule/service.js'
import * as AssetService from '@bindings/github.com/yottaapp/yotta/internal/services/asset/service.js'
import * as CalibrationService from '@bindings/github.com/yottaapp/yotta/internal/services/calibration/service.js'
import * as ToolsService from '@bindings/github.com/yottaapp/yotta/internal/services/tools/service.js'
import * as AppInfoService from '@bindings/github.com/yottaapp/yotta/internal/services/appinfoservice.js'
import * as RecordingService from '@bindings/github.com/yottaapp/yotta/internal/services/recording/service.js'
import * as ClipService from '@bindings/github.com/yottaapp/yotta/internal/services/inputclip/service.js'
import * as SubgraphService from '@bindings/github.com/yottaapp/yotta/internal/services/container/subgraphservice.js'
import * as CodeSnippetService from '@bindings/github.com/yottaapp/yotta/internal/services/codesnippet/service.js'
import * as AIService from '@bindings/github.com/yottaapp/yotta/internal/services/aiservice.js'
import { invoke } from './invoke'
import * as E from '@/constants/events'

// 事件 payload 类型（跟 Go events.go 一一对应；wails3 bindings 也会产 .d.ts，
// 这里手写一份用于 store 引用更稳，避免 bindings 路径变化）
export interface BackendLogEntry {
  time: string
  level: string
  source: 'SYS' | 'CTR'
  kind: 'system' | 'log' | 'node' | 'dump' | 'action'
  tag?: string
  message: string
  fields?: unknown
  nodeId?: string
  lineKey?: string
  count?: number
  final?: boolean
  trace?: unknown
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
// 装 vue-i18n named interpolation (容器名 / 计划名等动态), backend Register 时填.
export interface HotkeyEntry {
  key: string
  source: 'system' | 'action' | 'container' | 'schedule' | 'editor' | 'recording'
  label: string
  labelParams?: Record<string, string>
  hotkeyStr: string
  status: 'active' | 'unbound' | 'failed'
  lastError: string
  readonlyReason: string
  mechanism?: 'os-global' | 'editor-inapp' | 'll-hook'
}

// ---- 容器架构数据层类型 ----

export interface VarDecl {
  name: string
  type: 'number' | 'bool' | 'string' | 'point' | 'list' | 'any'
  default?: any
}
export interface GraphNode {
  id: string
  kind: string
  label?: string // 用户可编辑的显示名 (UE/Houdini 标准, optional, 不影响逻辑)
  x: number
  y: number
  config?: Record<string, any>
  disabled?: boolean // runtime 跳过该节点 — 走 kind-aware passthrough
  logEnabled?: boolean // 勾选 → 执行时吐通用 dump 日志到面板/文件
  createdAt?: string
}
export interface GraphEdge {
  from: string
  to: string
  // edge kind 由 (fromNode.kind, fromPin) 经 nodeRegistry.edgeKindOf 派生, 不存字段.
}

// Graph: id + schemaVersion
export interface Graph {
  id: string
  schemaVersion: number
  nodes: GraphNode[]
  edges: GraphEdge[]
}

// SubgraphOutputDecl — 父图边引用稳定 ID, UI 显示 Name (允许 rename name 不破坏 edge).
// B2: nodeID/x/y 是子图内 virtual 出口节点 metadata, 编辑器渲染为虚拟节点.
export interface SubgraphOutputDecl {
  id: string
  name: string
  nodeID?: string
  x?: number
  y?: number
}

// SubgraphMarker — B2 Subgraph 入口 virtual 节点位置 + ID. Edges 引用 NodeID.
export interface SubgraphMarker {
  nodeID: string
  x?: number
  y?: number
}

// ValidationError 后端 validator 结构化错误 (validator.go ValidationError 镜像).
// B5: Message 字段已删, FE 全走 t(`error.<code>`, params).
export interface ValidationError {
  severity: 'error' | 'warning'
  code: string
  graphPath: string[]
  nodeId?: string
  params?: Record<string, unknown>
}

export interface DebugStartOptions {
  startNodeId?: string
  graphPath?: string[]
}

export interface DebugSessionState {
  sessionId: string
  containerId: string
  status: string
  mode: string
  startNodeId?: string
  currentNodeId?: string
  currentNodeKind?: string
  runningNodeId?: string
  runningNodeKind?: string
  lastNodeId?: string
  lastNodeKind?: string
  lastExit?: string
  lastOutput?: Record<string, unknown>
  vars?: Record<string, unknown>
  queue?: Array<{
    nodeId: string
    nodeKind: string
    inPin: string
    graphPath?: string[]
    loopDepth?: number
    execDataKeys?: string[]
  }>
  error?: {
    message?: string
    code?: string
    params?: Record<string, unknown>
    errors?: ValidationError[]
  } | null
  warnings?: Array<{
    code: string
    message: string
    nodeId?: string
    params?: Record<string, unknown>
  }>
}

export interface RecordingContext {
  mouseCounts360: number
  resolution: [number, number]
  recordedAt: string
}

// Subgraph — 全局子图池里的可执行函数 (2026-06-12 全局化: 容器只引用不复制)
export interface Subgraph {
  id: string
  rev: number // 单调版本号 — 保存/删除乐观锁基准 (仅单实例并发控制)
  label: string
  description?: string
  graph: Graph
  entry: SubgraphMarker // B2: 子图入口 virtual marker
  outputPins: SubgraphOutputDecl[]
  tags?: string[]
  category?: string
  // 引用的容器级 var 名字 (保存时后端派生; Type/Default 由消费方按目标容器即时现算)
  requiredGlobals?: string[]
  recordingContext?: RecordingContext
  isAnonymous?: boolean // CollapsedNode 后备体 — 不进库浏览/候选列表
  createdAt: string
}

// SubgraphReferrer 子图引用位置 — 删除前警告 + 库页"被 N 个容器使用"(按 containerID 去重).
// 引用方在池子图内时 containerID 为空 (子图可被多容器使用, 无单一归属).
export interface SubgraphReferrer {
  containerID: string
  subgraphID?: string
  nodeID: string
  nodeKind: string
}

export interface Container {
  schemaVersion: number
  id: string
  name: string
  description?: string
  tags?: string[]
  packageId?: string
  packageName?: string
  version?: string
  category?: string
  keywords?: string[]
  author?: { name?: string; email?: string; url?: string } | string
  hotkey?: string
  inputBackend?: string
  captureBackend?: string
  scaleTolerance?: number
  vars?: VarDecl[]
  graph: Graph
  // 子图已全局化 — 容器无 subgraphs 字段, 全池走 backend.subgraphs.list()
  status?: string
  incompatibleReason?: string
  createdAt: string
  updatedAt: string
}

export interface Schedule {
  schemaVersion: number
  id: string
  name: string
  enabled: boolean
  targets: { kind: 'container'; id: string }[]
  trigger: {
    kind: 'cron' | 'hotkey' | 'once' | 'manual'
    subKind?: 'daily' | 'interval'
    at?: string
    everyMinutes?: number
    hotkey?: string
  }
  timeoutMinutes: number
  onError: 'stop' | 'continue'
  lastFiredAt?: string
  lastStatus?: string
  createdAt: string
  updatedAt: string
}

// AssetSummary 全局资产列表项 — 对应后端 asset.AssetSummary.
// 键 = guid (稳定 UUID), 不再是 namespace.name key.
export interface AssetSummary {
  guid: string
  kind: string // "template" | "clip"
  name: string
  description?: string
  category?: string
  tags?: string[]
  variantCount: number
  firstBlobSha?: string // 首变体 blob sha (FE 用 ReadBlobDataURL 拉缩略图)
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
  variants?: Array<{ resolution: number[]; bbox: number[]; blob: string }>
  blob?: string
  createdAt: string
}

// Referrer 引用位置 — Delete 返回, FE 据此弹"被 N 处引用"确认.
export interface AssetReferrer {
  containerID: string
  subgraphID?: string
  nodeID: string
  nodeKind: string
}

// AIConnection 跟 Go services.AIConnection 对齐（连接 = credential，model 由 AI 节点选）。
export interface AIConnection {
  id: string
  label: string
  protocol: 'openai' | 'anthropic'
  baseURL: string
  apiKey: string
}

// AITestResult 跟 Go services.TestResult 对齐。
export interface AITestResult {
  ok: boolean
  models: string[]
  error: string
  kind: string
}

export const backend = {
  settings: {
    get: () => invoke(SettingsService.Get),
    update: (patch: object) => invoke(SettingsService.Update, JSON.stringify(patch)),
  },
  ai: {
    testConnection: (connection: AIConnection, testModel: string) =>
      invoke(AIService.TestConnection, { connection, testModel }) as Promise<
        AITestResult | undefined
      >,
  },
  containers: {
    list: () => invoke(ContainerService.List),
    get: (id: string) => invoke(ContainerService.Get, id),
    // 从磁盘重读单个容器 (MCP / 外部改盘后, 编辑器「重载」按钮用)。返回最新容器, 同时刷新后端 byID 缓存。
    reload: (id: string) => invoke(ContainerService.Reload, id),
    create: (name: string) => invoke(ContainerService.Create, name),
    update: (id: string, patchJSON: string) => invoke(ContainerService.Update, id, patchJSON),
    // 裸版本: 不走 invoke 自动 toast, 抛错给调用方自己 catch 定制错误提示
    // (useEditorSave 合并进「主图保存失败」单条, 不叠两条 toast)。
    updateSilent: (id: string, patchJSON: string) => ContainerService.Update(id, patchJSON),
    delete_: (id: string) => invoke(ContainerService.Delete, id),
    pickExportPath: (filename: string, title: string, buttonText: string) =>
      Dialogs.SaveFile({
        Filename: filename,
        Title: title,
        ButtonText: buttonText,
        CanCreateDirectories: true,
        Filters: [{ DisplayName: 'Yotta Container', Pattern: '*.yotta-container.zip' }],
      }),
    exportPackage: (id: string, destPath: string) =>
      invoke(ContainerService.ExportPackage, id, destPath) as Promise<boolean | undefined>,
    run: (id: string) => invoke(ContainerService.Run, id),
    stopAll: () => invoke(ContainerService.StopAll),
    debugStart: (id: string, options: DebugStartOptions = {}) =>
      invoke(ContainerService.DebugStart, id, options as any) as Promise<
        DebugSessionState | undefined
      >,
    debugStep: (sessionID: string) =>
      invoke(ContainerService.DebugStep, sessionID) as Promise<DebugSessionState | undefined>,
    debugContinue: (sessionID: string) =>
      invoke(ContainerService.DebugContinue, sessionID) as Promise<DebugSessionState | undefined>,
    debugPause: (sessionID: string) =>
      invoke(ContainerService.DebugPause, sessionID) as Promise<DebugSessionState | undefined>,
    debugStop: (sessionID: string) =>
      invoke(ContainerService.DebugStop, sessionID) as Promise<DebugSessionState | undefined>,
    debugState: (sessionID: string) =>
      invoke(ContainerService.DebugState, sessionID) as Promise<DebugSessionState | undefined>,
    syncLocalMouseCalibration: (newCounts: number) =>
      invoke(ContainerService.SyncLocalMouseCalibration, newCounts),
    deleteMany: (ids: string[]) => ContainerService.DeleteMany(ids),
    // 「清空容器热键」: 去掉所有容器的热键绑定 (容器/蓝图保留)。返回清掉数量。
    clearAllHotkeys: () => invoke(ContainerService.ClearAllHotkeys) as Promise<number | undefined>,
    validate: (id: string) =>
      invoke(ContainerService.ValidateContainerByID, id) as Promise<ValidationError[]>,
  },
  // 全局子图池 (2026-06-12 全局化): 无 containerID; 保存/删除带基准 rev 乐观锁.
  subgraphs: {
    list: () => invoke(SubgraphService.List) as Promise<Subgraph[] | undefined>,
    get: (id: string) => invoke(SubgraphService.Get, id) as Promise<Subgraph | undefined>,
    create: (label: string) =>
      invoke(SubgraphService.Create, label) as Promise<Subgraph | undefined>,
    update: (id: string, patchJSON: string, baseRev: number) =>
      invoke(SubgraphService.Update, id, patchJSON, baseRev),
    // 裸版本: 不走 invoke 自动 toast, useEditorSave 子图循环汇总失败 + 乐观锁拒绝走重载对话框。
    updateSilent: (id: string, patchJSON: string, baseRev: number) =>
      SubgraphService.Update(id, patchJSON, baseRev),
    delete_: (id: string, baseRev: number) => invoke(SubgraphService.Delete, id, baseRev),
    // 复制为新子图 (fork, ≈Blender Make Local): 新 ID / rev=1 / 一律具名.
    duplicate: (id: string) =>
      invoke(SubgraphService.Duplicate, id) as Promise<Subgraph | undefined>,
    // 删除前警告 + 库页引用计数 ("被 N 个容器使用" = referrers 按 containerID 去重).
    referrers: (id: string) =>
      invoke(SubgraphService.Referrers, id) as Promise<SubgraphReferrer[] | undefined>,
    previewCleanup: () => invoke(SubgraphService.PreviewCleanup),
    cleanupUnused: (ids: string[]) => invoke(SubgraphService.CleanupUnused, { ids }),
  },
  // 编辑器用户代码片段: <dataDir>/snippets.json 整存整取 (量小改动低频, 前端持全量列表).
  codeSnippets: {
    list: () => invoke(CodeSnippetService.List),
    saveAll: (
      list: {
        id: string
        lang: string
        prefix: string
        name: string
        description?: string
        body: string
      }[],
    ) => invoke(CodeSnippetService.SaveAll, list as any),
  },
  schedules: {
    list: () => invoke(ScheduleService.List),
    get: (id: string) => invoke(ScheduleService.Get, id),
    create: (name: string) => invoke(ScheduleService.Create, name),
    // Schedule 类型 cast：wails 生成的 Trigger 把 optional 字段当成 required-undefined，
    // 跟我们手写的 optional Schedule 类型有微小不兼容。运行期 JSON 形态一致。
    save: (sc: Schedule) => invoke(ScheduleService.Save, sc as any),
    update: (id: string, patchJSON: string) => invoke(ScheduleService.Update, id, patchJSON),
    delete_: (id: string) => invoke(ScheduleService.Delete, id),
  },
  assets: {
    // List 全局资产列表 (template + clip), 无 containerID.
    list: () => invoke(AssetService.List),
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
    get: (guid: string) => invoke(AssetService.Get, guid) as Promise<AssetRecord | undefined>,
    delete_: (guid: string) => invoke(AssetService.Delete, guid),
    // Referrers 只扫不删 — 删除前拿引用列表, FE 据此弹"被 N 处引用"确认.
    referrers: (guid: string) =>
      invoke(AssetService.Referrers, guid) as Promise<AssetReferrer[] | undefined>,
    previewCleanup: () => invoke(AssetService.PreviewCleanup),
    cleanupUnused: (ids: string[]) => invoke(AssetService.CleanupUnused, { ids }),
    // UpdateMeta 改显示名 + 标签 (记录级元数据).
    updateMeta: (
      guid: string,
      name: string,
      description: string,
      category: string,
      tags: string[],
    ) => invoke(AssetService.UpdateMeta, guid, name, description, category, tags),
    // Capture 截当前容器的 Windows 窗口帧 (保留 containerID — 现阶段资产截帧仍需 Win32 窗口上下文).
    capture: (containerID: string, nodeID = '') =>
      invoke(AssetService.Capture, containerID, nodeID),
    // ReadBlobDataURL 按 blob sha 拿 data URL (缩略图).
    readBlobDataURL: (sha: string) => invoke(AssetService.ReadBlobDataURL, sha),
    gcBlobs: () => invoke(AssetService.GCBlobs),
    // CurrentResolution 当前容器 Windows 窗口客户区分辨率 [宽,高]; 窗口没开/无容器上下文 → 静默返 undefined.
    // 不走 invoke: 浏览态窗口没开属正常, 不该弹 error toast.
    currentResolution: async (containerID: string): Promise<[number, number] | undefined> => {
      try {
        const r = await AssetService.CurrentResolution(containerID)
        return Array.isArray(r) && r.length === 2 ? [r[0], r[1]] : undefined
      } catch {
        return undefined
      }
    },
    // PickVariant 给定分辨率, 返运行时真会用的那档在 variants[] 里的下标 + 是否精确命中. 自动调用, 失败静默.
    pickVariant: async (
      guid: string,
      w: number,
      h: number,
    ): Promise<{ index: number; exact: boolean } | undefined> => {
      try {
        const r = await AssetService.PickVariant(guid, w, h)
        return { index: r.index, exact: r.exact }
      } catch {
        return undefined
      }
    },
    // RemoveVariant 删指定分辨率的单个变体档. 返目标 GUID (成功) / undefined (失败已 toast).
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
    start: (args: { filterMode: 'precise' | 'simple'; containerID: string }) =>
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
    }) => invoke(RecordingService.Finalize, args as any),
    discard: (pendingID: string) => invoke(RecordingService.Discard, pendingID),
    previewCleanup: () => invoke(RecordingService.PreviewCleanup),
    cleanupUnused: (ids: string[]) => invoke(RecordingService.CleanupUnused, { ids } as any),
    pause: () => invoke(RecordingService.Pause),
    resume: () => invoke(RecordingService.Resume),
    validateTarget: (containerID: string) => invoke(RecordingService.ValidateTarget, containerID),
    getState: () => invoke(RecordingService.GetState),
  },
  // 全局 ClipService (main.go RegisterService(clipSvc); 资产全局化后无 lib/容器两套存储).
  // 暴露 list/get/save/update/delete + Resolve (runtime 用, 前端基本不直接调).
  clipsContainer: {
    list: () => invoke(ClipService.List),
    get: (id: string) => invoke(ClipService.Get, id),
    save: (clip: unknown) => invoke(ClipService.Save, clip as any),
    update: (id: string, label: string, description: string, category: string, tags: string[]) =>
      invoke(ClipService.Update, id, label, description, category, tags),
    delete_: (id: string) => invoke(ClipService.Delete, id),
    resolve: (id: string) => invoke(ClipService.Resolve, id),
  },
  tools: {
    mousePos: (containerID: string, nodeID = '') =>
      invoke(ToolsService.MousePos, containerID, nodeID),
    pixelAt: (containerID: string, nodeID = '') =>
      invoke(ToolsService.PixelAt, containerID, nodeID),
    openMouseHUD: (containerID: string) => invoke(ToolsService.OpenMouseHUD, containerID),
    openRecordingHUD: () => invoke(ToolsService.OpenRecordingHUD),
    closeRecordingHUD: () => invoke(ToolsService.CloseRecordingHUD),
    openCalibratorHUD: (id: string) => invoke(ToolsService.OpenCalibratorHUD, id),
    closeCalibratorHUD: () => invoke(ToolsService.CloseCalibratorHUD),
    openScreenPicker: (
      mode: 'point' | 'rect' | 'template_save' | 'template_recapture' | 'color',
      id: string,
      containerID = '',
      nodeID = '',
      colorSpace = '',
      guid = '',
    ) => invoke(ToolsService.OpenScreenPicker, mode, id, containerID, nodeID, colorSpace, guid),
    extractColorRange: (samples: { R: number; G: number; B: number }[], colorSpace: string) =>
      invoke(ToolsService.ExtractColorRange, samples, colorSpace),
    closePicker: (id: string) => invoke(ToolsService.ClosePicker, id),
    // Win32WindowTarget capture: 注册全局 hotkey (默认 F9 = 0x78), 用户在游戏窗口按下后
    // 走 'win32windowtarget:captured' event 回填. 取代旧同步 captureForegroundWindow
    // — 用户在游戏前台时根本点不到 Yotta 按钮.
    // 捕获键来源 = 后端读热键中心 tools.window-capture 绑定 (可在「快捷键」页 rebind)，不再 FE 传死。
    startWin32WindowTargetCapture: () => invoke(ToolsService.StartWin32WindowTargetCapture),
    cancelWin32WindowTargetCapture: (id: string) =>
      invoke(ToolsService.CancelWin32WindowTargetCapture, id),
    openLauncher: () => invoke(ToolsService.OpenLauncher),
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
  },
}
