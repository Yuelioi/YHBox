// 所有 wails3 对接走这一层 —— stores / views / events.ts 都通过 backend.xxx.yyy(...) 调用，
// 不直接 import bindings 或 @wailsio/runtime。理由：wails3 alpha API 漂移时只改这一个文件。

import { Events } from '@wailsio/runtime'
import * as SettingsService from '@bindings/yotta/internal/services/settingsservice.js'
import * as HotkeyService from '@bindings/yotta/internal/hotkey/hotkeyservice.js'
import * as ContainerService from '@bindings/yotta/internal/services/container/service.js'
import * as ScheduleService from '@bindings/yotta/internal/services/schedule/service.js'
import * as TemplateService from '@bindings/yotta/internal/services/template/service.js'
import * as CalibrationService from '@bindings/yotta/internal/services/calibration/service.js'
import * as ToolsService from '@bindings/yotta/internal/services/tools/service.js'
import * as AppInfoService from '@bindings/yotta/internal/services/appinfoservice.js'
import * as RecordingService from '@bindings/yotta/internal/services/recording/service.js'
import * as ClipService from '@bindings/yotta/internal/services/inputclip/service.js'
import * as LibraryService from '@bindings/yotta/internal/services/container/library/service.js'
import { invoke } from './invoke'
import * as E from '@/constants/events'

// 事件 payload 类型（跟 Go events.go 一一对应；wails3 bindings 也会产 .d.ts，
// 这里手写一份用于 store 引用更稳，避免 bindings 路径变化）
export interface LogLinesEvent {
  seq: number
  lines: string[]
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
  type: 'number' | 'bool' | 'string' | 'point' | 'any'
  default?: any
}
export interface GraphNode {
  id: string
  kind: string
  label?: string       // 用户可编辑的显示名 (UE/Houdini 标准, optional, 不影响逻辑)
  x: number
  y: number
  config?: Record<string, any>
  disabled?: boolean   // runtime 跳过该节点 — 走 kind-aware passthrough
  logEnabled?: boolean // 勾选 → 执行时吐通用 dump 日志到面板/文件
  createdAt?: string
}
export interface GraphEdge {
  from: string
  to: string
  // edge kind 由 (fromNode.kind, fromPin) 经 nodeRegistry.edgeKindOf 派生, 不存字段.
}

// Graph v2: 加 id + version
export interface Graph {
  id: string
  version: number
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

export interface RecordingContext {
  mouseCounts360: number
  resolution: [number, number]
  recordedAt: string
}

// Subgraph — 容器内的可执行函数
export interface Subgraph {
  id: string
  label: string
  description?: string
  graph: Graph
  entry: SubgraphMarker // B2: 子图入口 virtual marker
  outputPins: SubgraphOutputDecl[]
  tags?: string[]
  recordingContext?: RecordingContext
  createdAt: string
}

// SubgraphPackage library package: root + 嵌入 callee + 共用 asset key 列表.
export interface SubgraphPackage {
  root: Subgraph
  embedded: Record<string, Subgraph>
  templates: string[]
  clips: string[]
}

// ImportConflict 单条冲突信息.
export interface ImportConflict {
  kind: string
  key: string
}

// SubgraphRequiredGlobal B11: 子图需要的容器级 global var 声明.
export interface SubgraphRequiredGlobal {
  name: string
  type?: string
  default?: unknown
}

// ImportResult Import 操作结果.
// B11: missingGlobals 反映 import 的 sg union 需要但目标 container 未声明的 var, FE 据此弹 prompt.
export interface ImportResult {
  imported: { kind: string; key: string }[]
  conflicts: ImportConflict[]
  missingGlobals?: SubgraphRequiredGlobal[]
}

export interface Container {
  schemaVersion: number
  id: string
  name: string
  description?: string
  tags?: string[]
  hotkey?: string
  inputBackend?: string
  captureBackend?: string
  scaleTolerance?: number
  vars?: VarDecl[]
  graph: Graph
  // subgraphs json:"-" 后端不持久化到 container.json 但 runtime 注入; 前端通过 listSubgraphs 单独拿
  // 这里声明 optional 仅供 type 完整性
  subgraphs?: Subgraph[]
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

export interface TemplateMeta {
  name: string
  description?: string
  // 后端 [2]int Go fixed-size array → wails3 binding 生成 number[] (encoding/json 视作普通 slice).
  // 这里跟 binding 对齐用 number[]; 实际固定 2 元素, 消费者按 indexed access 用就行.
  recordedResolution: number[]
  sha256: string
  width: number
  height: number
  // 同理: Go [4]int → number[].
  region: number[]
  createdAt: string
  // v2: 仅库模板使用；容器内模板该字段为空。
  tags?: string[]
  // v2: 来源追踪（screenshot / library / imported / embedded）。
  origin?: { kind: string; sourceID?: string }
}

export const backend = {
  settings: {
    get: () => invoke(SettingsService.Get),
    update: (patch: object) => invoke(SettingsService.Update, JSON.stringify(patch)),
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
    run: (id: string) => invoke(ContainerService.Run, id),
    stopAll: () => invoke(ContainerService.StopAll),
    listSubgraphs: (id: string) => invoke(ContainerService.ListSubgraphs, id),
    getSubgraph: (cid: string, sgid: string) => invoke(ContainerService.GetSubgraph, cid, sgid),
    createSubgraph: (cid: string, label: string) => invoke(ContainerService.CreateSubgraph, cid, label),
    updateSubgraph: (cid: string, sgid: string, patchJSON: string) =>
      invoke(ContainerService.UpdateSubgraph, cid, sgid, patchJSON),
    // 裸版本: 同 updateSilent, 让 useEditorSave 子图循环只汇总成一条失败 toast。
    updateSubgraphSilent: (cid: string, sgid: string, patchJSON: string) =>
      ContainerService.UpdateSubgraph(cid, sgid, patchJSON),
    deleteSubgraph: (cid: string, sgid: string) =>
      invoke(ContainerService.DeleteSubgraph, cid, sgid),
    openEditorWindow: (id: string) => invoke(ContainerService.OpenEditorWindow, id),
    openInWindow: (id: string) => invoke(ContainerService.OpenInWindow, id),
    syncLocalMouseCalibration: (newCounts: number) =>
      invoke(ContainerService.SyncLocalMouseCalibration, newCounts),
    deleteMany: (ids: string[]) => invoke(ContainerService.DeleteMany, ids),
    // 「清空容器热键」: 去掉所有容器的热键绑定 (容器/蓝图保留)。返回清掉数量。
    clearAllHotkeys: () => invoke(ContainerService.ClearAllHotkeys) as Promise<number | undefined>,
    validate: (id: string) =>
      invoke(ContainerService.ValidateContainerByID, id) as Promise<ValidationError[]>,
  },
  library: {
    listSubgraphs: () => invoke(LibraryService.ListSubgraphs),
    getSubgraphPackage: (sgID: string) => invoke(LibraryService.GetSubgraphPackage, sgID),
    deleteSubgraphPackage: (sgID: string) => invoke(LibraryService.DeleteSubgraphPackage, sgID),
    importToContainer: (libSgID: string, containerID: string, strategy: string) =>
      invoke(LibraryService.ImportToContainer, libSgID, containerID, strategy),
    exportSubgraph: (containerID: string, sgID: string, overwrite: boolean) =>
      invoke(LibraryService.ExportSubgraph, containerID, sgID, overwrite),
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
  templates: {
    list: (containerID: string) => invoke(TemplateService.List, containerID),
    save: (
      containerID: string,
      key: string,
      dataURL: string,
      name: string,
      description: string,
      recordedResolution: [number, number],
      region: [number, number, number, number],
    ) => invoke(TemplateService.Save, containerID, key, dataURL, name, description, recordedResolution, region),
    delete_: (containerID: string, key: string) => invoke(TemplateService.Delete, containerID, key),
    capture: (containerID: string) => invoke(TemplateService.Capture, containerID),
    readPngDataURL: (containerID: string, key: string) => invoke(TemplateService.ReadPngDataURL, containerID, key),
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
    // Start 收 {filterMode, containerID}. containerID 必传 — 录完 Subgraph 落到该容器
    // subgraphs/. 返临时 recording ID (前端订阅事件流过滤用).
    start: (args: { filterMode: 'precise' | 'simple'; containerID: string }) =>
      invoke(RecordingService.Start, args as any),
    // Stop 返 {subgraphID, containerID, label, filterMode} — 录完产物 = 一个 Subgraph,
    // 前端拿 subgraphID 在 activeGraph 加 Subgraph 引用节点.
    stop: () => invoke(RecordingService.Stop),
    stopAsync: () => invoke(RecordingService.StopAsync),
    // Pause/Resume 切除间隔: 暂停期不录, 时间戳扣除该段 → 回放无空档. HUD 按钮 / 暂停热键触发.
    pause: () => invoke(RecordingService.Pause),
    resume: () => invoke(RecordingService.Resume),
    // ValidateTarget 录制前预检: 找不到 WindowTarget 窗口返 error (倒计时前调, 不用等录完才报错);
    // 成功则把游戏窗口拉到前台. 失败抛出供前端 toast + 中止倒计时.
    validateTarget: (containerID: string) => invoke(RecordingService.ValidateTarget, containerID),
    // GetState 返回后端权威录制状态 {phase, containerID, filterMode, tempID, startedAtMs}.
    // 前端 recordStore reconcile 对账用 — 取代旧的 isRecording (bool 不够, desync 无法自愈).
    getState: () => invoke(RecordingService.GetState),
  },
  // 容器级 ClipService (main.go 只 RegisterService(clipSvc); libClipSvc 不注册).
  // 暴露 list/get/save/update/delete + Resolve (runtime 用, 前端基本不直接调).
  clipsContainer: {
    list: () => invoke(ClipService.List),
    get: (id: string) => invoke(ClipService.Get, id),
    save: (clip: unknown) => invoke(ClipService.Save, clip as any),
    update: (id: string, label: string, description: string, tags: string[]) =>
      invoke(ClipService.Update, id, label, description, tags),
    delete_: (id: string) => invoke(ClipService.Delete, id),
    resolve: (id: string) => invoke(ClipService.Resolve, id),
  },
  tools: {
    mousePos: (containerID: string, nodeID = '') => invoke(ToolsService.MousePos, containerID, nodeID),
    pixelAt: (containerID: string, nodeID = '') => invoke(ToolsService.PixelAt, containerID, nodeID),
    openMouseHUD: (containerID: string) => invoke(ToolsService.OpenMouseHUD, containerID),
    openRecordingHUD: () => invoke(ToolsService.OpenRecordingHUD),
    closeRecordingHUD: () => invoke(ToolsService.CloseRecordingHUD),
    openCalibratorHUD: (id: string) => invoke(ToolsService.OpenCalibratorHUD, id),
    closeCalibratorHUD: () => invoke(ToolsService.CloseCalibratorHUD),
    openScreenPicker: (
      mode: 'point' | 'rect' | 'template_save' | 'color',
      id: string,
      containerID = '',
      nodeID = '',
      colorSpace = '',
    ) => invoke(ToolsService.OpenScreenPicker, mode, id, containerID, nodeID, colorSpace),
    extractColorRange: (samples: { R: number; G: number; B: number }[], colorSpace: string) =>
      invoke(ToolsService.ExtractColorRange, samples, colorSpace),
    closePicker: (id: string) => invoke(ToolsService.ClosePicker, id),
    // WindowTarget capture: 注册全局 hotkey (默认 F9 = 0x78), 用户在游戏窗口按下后
    // 走 'windowtarget:captured' event 回填. 取代旧同步 captureForegroundWindow
    // — 用户在游戏前台时根本点不到 Yotta 按钮.
    // 捕获键来源 = 后端读热键中心 tools.window-capture 绑定 (可在「快捷键」页 rebind)，不再 FE 传死。
    startWindowTargetCapture: () => invoke(ToolsService.StartWindowTargetCapture),
    cancelWindowTargetCapture: (id: string) =>
      invoke(ToolsService.CancelWindowTargetCapture, id),
    openLauncher: () => invoke(ToolsService.OpenLauncher),
    toggleLauncher: () => invoke(ToolsService.ToggleLauncher),
    hideLauncher: () => invoke(ToolsService.HideLauncher),
    setLauncherAlwaysOnTop: (on: boolean) => invoke(ToolsService.SetLauncherAlwaysOnTop, on),
    setLauncherSize: (width: number, height: number) => invoke(ToolsService.SetLauncherSize, width, height),
  },
  events: {
    // 共享事件
    onLogLines: (cb: (e: LogLinesEvent) => void) =>
      Events.On(E.EVENT_LOG_LINES, (e: any) => cb(e.data)),
    onHotkeyChanged: (cb: () => void) => Events.On('hotkey:changed', () => cb()),
  },
}
