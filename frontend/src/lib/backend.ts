// 所有 wails3 对接走这一层 —— stores / views / events.ts 都通过 backend.xxx.yyy(...) 调用，
// 不直接 import bindings 或 @wailsio/runtime。理由：wails3 alpha API 漂移时只改这一个文件。

import { Events } from '@wailsio/runtime'
import * as FishService from '@bindings/yhbox/internal/bots/fishservice.js'
import * as CookService from '@bindings/yhbox/internal/bots/cookservice.js'
import * as PianoService from '@bindings/yhbox/internal/bots/pianoservice.js'
import * as BattleService from '@bindings/yhbox/internal/bots/battleservice.js'
import * as RhythmService from '@bindings/yhbox/internal/bots/rhythmservice.js'
import * as SettingsService from '@bindings/yhbox/internal/services/settingsservice.js'
import * as GameService from '@bindings/yhbox/internal/services/gameservice.js'
import * as ActionsService from '@bindings/yhbox/internal/services/actions/service.js'
import * as HotkeyService from '@bindings/yhbox/internal/hotkey/hotkeyservice.js'
import * as ContainerService from '@bindings/yhbox/internal/services/container/service.js'
import * as ScheduleService from '@bindings/yhbox/internal/services/schedule/service.js'
import * as TemplateService from '@bindings/yhbox/internal/services/template/service.js'
import * as CalibrationService from '@bindings/yhbox/internal/services/calibration/service.js'
import * as ToolsService from '@bindings/yhbox/internal/services/tools/service.js'
import * as AppInfoService from '@bindings/yhbox/internal/services/appinfoservice.js'
import { invoke } from './invoke'
import * as E from '@/constants/events'

// 事件 payload 类型（跟 Go events.go 一一对应；wails3 bindings 也会产 .d.ts，
// 这里手写一份用于 store 引用更稳，避免 bindings 路径变化）
export interface BotStateEvent {
  state: 'idle' | 'running' | 'paused'
}
export interface FishStatsEvent {
  commonCount: number
  purpleCount: number
  goldenCount: number
  startedAt: string
}
export interface PianoProgressEvent {
  curMs: number
  totalMs: number
}
export interface BattleStateEvent {
  enabled: boolean
  mods: string
}
export interface GameStatusEvent {
  ok: boolean
  hwnd: number
  title: string
  w: number
  h: number
}
export interface LogLinesEvent {
  seq: number
  lines: string[]
}
export interface ActionStateEvent {
  seq: number
  timestamp: string
  actionId: string
  runId: number
  status: 'running' | 'step' | 'idle' | 'error'
  error?: string
  // status='step' 时填
  stepIndex?: number
  stepTotal?: number
  stepId?: string
  loopIter?: number
}
export interface RecorderStateEvent {
  status: 'started' | 'stopped' | 'error'
  tempActionId?: string
  action?: any
  error?: string
}

// HotkeyEntry 跟 Go services.HotkeyEntry 对齐。Normalized 字段后端故意不导出 — 前端不依赖
// canonicalization 规则。冲突 / reserved / 验证错误通过 error message 前缀 [conflict] /
// [reserved] / [invalid] 区分。
export interface HotkeyEntry {
  key: string
  source: 'system' | 'action'
  label: string
  hotkeyStr: string
  status: 'active' | 'unbound' | 'failed'
  lastError: string
  readonlyReason: string
}

// ---- 新数据层类型（spec 2026-05-15 容器架构）----

export interface VarDecl {
  name: string
  type: 'number' | 'bool' | 'string' | 'point'
  default?: any
}
export interface GraphNode {
  id: string
  kind: string
  x: number
  y: number
  config?: Record<string, any>
}
export interface GraphEdge {
  from: string
  to: string
}

export interface Container {
  schemaVersion: number
  id: string
  name: string
  description?: string
  category?: string
  tags?: string[]
  hotkey?: string
  runMode?: 'foreground' | 'background'
  vars?: VarDecl[]
  graph: { nodes: GraphNode[]; edges: GraphEdge[] }
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
  recordedResolution: [number, number]
  sha256: string
  width: number
  height: number
  region: [number, number, number, number]
  createdAt: string
}

export const backend = {
  fish: {
    start: () => invoke(FishService.Start),
    pause: () => invoke(FishService.Pause),
    resume: () => invoke(FishService.Resume),
    stop: () => invoke(FishService.Stop),
    setAutoSell: (v: boolean) => invoke(FishService.SetAutoSell, v),
    setCaptureGolden: (v: boolean) => invoke(FishService.SetCaptureGolden, v),
    getStats: () => invoke(FishService.GetStats),
  },
  cook: {
    start: () => invoke(CookService.Start),
    pause: () => invoke(CookService.Pause),
    resume: () => invoke(CookService.Resume),
    stop: () => invoke(CookService.Stop),
    setIntervalMs: (ms: number) => invoke(CookService.SetIntervalMs, ms),
  },
  piano: {
    start: () => invoke(PianoService.Start),
    pause: () => invoke(PianoService.Pause),
    resume: () => invoke(PianoService.Resume),
    stop: () => invoke(PianoService.Stop),
    seek: (ms: number) => invoke(PianoService.SeekMs, ms),
    setMode: (m: number) => invoke(PianoService.SetMode, m),
    setTrackPick: (p: number) => invoke(PianoService.SetTrackPick, p),
    setAutoTranspose: (v: boolean) => invoke(PianoService.SetAutoTranspose, v),
    setMelodyOnly: (v: boolean) => invoke(PianoService.SetMelodyOnly, v),
    setOctaveOffset: (v: number) => invoke(PianoService.SetOctaveOffset, v),
    getLibrary: () => invoke(PianoService.GetLibrary),
    loadBuiltin: (asset: string) => invoke(PianoService.LoadBuiltin, asset),
    loadFile: (path: string) => invoke(PianoService.LoadFile, path),
  },
  battle: {
    enable: () => invoke(BattleService.Enable),
    disable: () => invoke(BattleService.Disable),
    setMods: (label: string) => invoke(BattleService.SetMods, label),
  },
  rhythm: {
    start: () => invoke(RhythmService.Start),
    pause: () => invoke(RhythmService.Pause),
    resume: () => invoke(RhythmService.Resume),
    stop: () => invoke(RhythmService.Stop),
    // setDebugFlags 暂不暴露 — 后端 RPC 仍存在，但前端没 UI 触发；要用直接走开发者控制台
  },
  settings: {
    get: () => invoke(SettingsService.Get),
    update: (patch: object) => invoke(SettingsService.Update, JSON.stringify(patch)),
  },
  game: {
    detect: () => invoke(GameService.Detect),
  },
  actions: {
    list: () => invoke(ActionsService.List),
    create: (name: string) => invoke(ActionsService.Create, name),
    update: (id: string, patchJSON: string) => invoke(ActionsService.Update, id, patchJSON),
    delete_: (id: string) => invoke(ActionsService.Delete, id),
    runOnce: (id: string) => invoke(ActionsService.RunOnce, id),
    stopRunning: () => invoke(ActionsService.StopRunning),
    openEditorWindow: (id: string) => invoke(ActionsService.OpenEditorWindow, id),
    startRecording: () => invoke(ActionsService.StartRecording),
    stopRecording: () => invoke(ActionsService.StopRecording),
    cancelRecording: () => invoke(ActionsService.CancelRecording),
    isRecording: () => invoke(ActionsService.IsRecording),
  },
  containers: {
    list: () => invoke(ContainerService.List),
    get: (id: string) => invoke(ContainerService.Get, id),
    create: (name: string) => invoke(ContainerService.Create, name),
    update: (id: string, patchJSON: string) => invoke(ContainerService.Update, id, patchJSON),
    delete_: (id: string) => invoke(ContainerService.Delete, id),
    run: (id: string) => invoke(ContainerService.Run, id),
    stopAll: () => invoke(ContainerService.StopAll),
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
    list: () => invoke(TemplateService.List),
    get: (key: string) => invoke(TemplateService.Get, key),
    save: (
      key: string,
      dataURL: string,
      name: string,
      description: string,
      recordedResolution: [number, number],
      region: [number, number, number, number],
    ) => invoke(TemplateService.Save, key, dataURL, name, description, recordedResolution, region),
    delete_: (key: string) => invoke(TemplateService.Delete, key),
    capture: () => invoke(TemplateService.CaptureScreenshot),
    readPngDataURL: (key: string) => invoke(TemplateService.ReadPngDataURL, key),
    updateMeta: (key: string, name: string, description: string) =>
      invoke(TemplateService.UpdateMeta, key, name, description),
  },
  hotkeys: {
    list: () => invoke(HotkeyService.List),
    update: (key: string, hotkeyStr: string) => invoke(HotkeyService.Update, key, hotkeyStr),
    // pause/resume：HotkeyCaptureInput 进入捕获模式时 pause，离开时 resume。
    // 否则 Win32 RegisterHotKey 在 OS 层拦截已注册组合，webview 收不到 keystroke。
    pause: () => invoke(HotkeyService.Pause),
    resume: () => invoke(HotkeyService.Resume),
  },
  calibration: {
    start: () => invoke(CalibrationService.Start),
    stop: () => invoke(CalibrationService.Stop),
    status: () => invoke(CalibrationService.Status),
  },
  appInfo: {
    info: () => invoke(AppInfoService.Info),
  },
  tools: {
    mousePos: () => invoke(ToolsService.MousePos),
    pixelAt: () => invoke(ToolsService.PixelAt),
    openMouseHUD: () => invoke(ToolsService.OpenMouseHUD),
    openScreenPicker: (mode: 'point' | 'rect' | 'template_save', id: string) =>
      invoke(ToolsService.OpenScreenPicker, mode, id),
    closePicker: (id: string) => invoke(ToolsService.ClosePicker, id),
  },
  events: {
    // 长跑 bot 的 state 事件：通用订阅（替代之前 4 个 onFishState/onCookState/...）
    onBotState: (kind: string, cb: (e: BotStateEvent) => void) =>
      Events.On(`${kind}:state`, (e: any) => cb(e.data)),
    // Bot-specific 事件（形态各异，单独保留）
    onFishStats: (cb: (e: FishStatsEvent) => void) =>
      Events.On(E.EVENT_FISH_STATS, (e: any) => cb(e.data)),
    onPianoProgress: (cb: (e: PianoProgressEvent) => void) =>
      Events.On(E.EVENT_PIANO_PROGRESS, (e: any) => cb(e.data)),
    onBattleState: (cb: (e: BattleStateEvent) => void) =>
      Events.On(E.EVENT_BATTLE_STATE, (e: any) => cb(e.data)),
    // 共享事件
    onGameStatus: (cb: (e: GameStatusEvent) => void) =>
      Events.On(E.EVENT_GAME_STATUS, (e: any) => cb(e.data)),
    onLogLines: (cb: (e: LogLinesEvent) => void) =>
      Events.On(E.EVENT_LOG_LINES, (e: any) => cb(e.data)),
    // actions 事件：state（run 状态推送）/ recorder-state（录制状态推送）/ recorder-event（录制期单次 raw 事件，给 dialog 计数）/ recorder-toggle（Ctrl+Shift+R 全局热键触发，前端走跟 UI 按钮同一路径）/ changed（任何 mutation 后广播，触发各窗口 reload list）
    onActionState: (cb: (e: ActionStateEvent) => void) =>
      Events.On('action:state', (e: any) => cb(e.data)),
    onRecorderState: (cb: (e: RecorderStateEvent) => void) =>
      Events.On('action:recorder-state', (e: any) => cb(e.data)),
    onRecorderEvent: (cb: () => void) => Events.On('action:recorder-event', () => cb()),
    onRecorderToggle: (cb: () => void) => Events.On('action:recorder-toggle', () => cb()),
    onActionChanged: (cb: () => void) => Events.On('action:changed', () => cb()),
    onHotkeyChanged: (cb: () => void) => Events.On('hotkey:changed', () => cb()),
  },
}
