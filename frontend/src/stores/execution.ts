// 全局 worker 状态。容器/动作运行由唯一 worker 串行执行；这里订阅
// `execution:state` 事件给 AppStatusBar 渲染，并在错误结束时 toast 出来。

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { toastError, errorMessage } from '@/lib/invoke'
import { i18n } from '@/i18n'

export interface WorkerStateEvent {
  running: boolean
  runId: number
  source: 'hotkey' | 'schedule' | 'manual' | ''
  targets?: { kind: string; id: string }[]
  targetIdx?: number
  error?: string
}

export type DebugStatus =
  | ''
  | 'paused'
  | 'stepping'
  | 'running'
  | 'pause_requested'
  | 'finished'
  | 'failed'
  | 'stopped'

export interface DebugTokenSummary {
  nodeId: string
  nodeKind: string
  inPin: string
  graphPath?: string[]
  loopDepth?: number
  execDataKeys?: string[]
}

export interface DebugRunError {
  message?: string
  code?: string
  params?: Record<string, unknown>
  errors?: unknown[]
}

export interface DebugWarning {
  code: string
  message: string
  nodeId?: string
  params?: Record<string, unknown>
}

export interface DebugSessionState {
  sessionId: string
  containerId: string
  status: DebugStatus
  mode: 'entry' | 'from_node' | ''
  startNodeId?: string
  currentNodeId?: string
  currentNodeKind?: string
  runningNodeId?: string
  runningNodeKind?: string
  lastNodeId?: string
  lastNodeKind?: string
  lastExit?: string
  queue?: DebugTokenSummary[]
  error?: DebugRunError | null
  warnings?: DebugWarning[]
}

function eventPayload(e: any): any {
  if (Array.isArray(e?.data)) return e.data[0]
  return e?.data ?? e
}

export const useExecutionStore = defineStore('execution', () => {
  const running = ref(false)
  const runId = ref(0)
  const source = ref<WorkerStateEvent['source']>('')
  const targets = ref<{ kind: string; id: string }[]>([])
  const targetIdx = ref(0)
  // 当前 container.name（前端用 containers store 映射 target id）
  const currentTargetID = ref('')
  // 当前正在执行的节点 id + kind（runner 每次进新节点 emit container:node-enter）
  const currentNodeID = ref('')
  const currentNodeKind = ref('')
  // 最近一次结束 run 的错误（null = 上次成功 / 还没结束过）
  const lastError = ref<string | null>(null)

  const debugSessionID = ref('')
  const debugContainerID = ref('')
  const debugStatus = ref<DebugStatus>('')
  const debugMode = ref<'entry' | 'from_node' | ''>('')
  const debugStartNodeID = ref('')
  const debugCurrentNodeID = ref('')
  const debugCurrentNodeKind = ref('')
  const debugRunningNodeID = ref('')
  const debugRunningNodeKind = ref('')
  const debugLastNodeID = ref('')
  const debugLastNodeKind = ref('')
  const debugLastExit = ref('')
  const debugFailedNodeID = ref('')
  const debugQueue = ref<DebugTokenSummary[]>([])
  const debugWarnings = ref<DebugWarning[]>([])
  const debugError = ref<DebugRunError | null>(null)

  const debugTerminal = computed(() =>
    ['finished', 'failed', 'stopped'].includes(debugStatus.value),
  )
  const debugActive = computed(() => !!debugSessionID.value && !debugTerminal.value)
  const debugBusy = computed(() =>
    ['stepping', 'running', 'pause_requested'].includes(debugStatus.value),
  )
  const debugCanStep = computed(() => debugStatus.value === 'paused')
  const debugCanContinue = computed(() => debugStatus.value === 'paused')
  const debugCanPause = computed(() => debugStatus.value === 'running')
  const debugNextNodeID = computed(() => debugQueue.value[0]?.nodeId ?? '')
  const debugNextNodeKind = computed(() => debugQueue.value[0]?.nodeKind ?? '')

  function clearDebugState() {
    debugSessionID.value = ''
    debugContainerID.value = ''
    debugStatus.value = ''
    debugMode.value = ''
    debugStartNodeID.value = ''
    debugCurrentNodeID.value = ''
    debugCurrentNodeKind.value = ''
    debugRunningNodeID.value = ''
    debugRunningNodeKind.value = ''
    debugLastNodeID.value = ''
    debugLastNodeKind.value = ''
    debugLastExit.value = ''
    debugFailedNodeID.value = ''
    debugQueue.value = []
    debugWarnings.value = []
    debugError.value = null
  }

  function applyDebugState(state: DebugSessionState | any) {
    if (!state) return
    const status = String(state.status ?? '') as DebugStatus
    const terminal = ['finished', 'failed', 'stopped'].includes(status)

    debugSessionID.value = String(state.sessionId ?? '')
    debugContainerID.value = String(state.containerId ?? '')
    debugStatus.value = status
    debugMode.value = (state.mode ?? '') as 'entry' | 'from_node' | ''
    debugStartNodeID.value = String(state.startNodeId ?? '')
    debugQueue.value = Array.isArray(state.queue) ? state.queue : []
    debugWarnings.value = Array.isArray(state.warnings) ? state.warnings : []
    debugError.value = state.error ?? null

    debugLastNodeID.value = String(state.lastNodeId ?? debugLastNodeID.value ?? '')
    debugLastNodeKind.value = String(state.lastNodeKind ?? debugLastNodeKind.value ?? '')
    debugLastExit.value = String(state.lastExit ?? debugLastExit.value ?? '')

    const currentID = String(state.currentNodeId ?? '')
    const currentKind = String(state.currentNodeKind ?? '')
    const runningID = String(state.runningNodeId ?? '')
    const runningKind = String(state.runningNodeKind ?? '')

    if (status === 'failed') {
      debugFailedNodeID.value = currentID || runningID || debugLastNodeID.value
      debugCurrentNodeID.value = currentID
      debugCurrentNodeKind.value = currentKind
      debugRunningNodeID.value = ''
      debugRunningNodeKind.value = ''
      return
    }

    if (status === 'paused') {
      debugCurrentNodeID.value = currentID
      debugCurrentNodeKind.value = currentKind
      debugRunningNodeID.value = ''
      debugRunningNodeKind.value = ''
      return
    }

    if (terminal) {
      debugCurrentNodeID.value = ''
      debugCurrentNodeKind.value = ''
      debugRunningNodeID.value = ''
      debugRunningNodeKind.value = ''
      return
    }

    debugFailedNodeID.value = ''
    debugCurrentNodeID.value = currentID
    debugCurrentNodeKind.value = currentKind
    debugRunningNodeID.value = runningID || currentID
    debugRunningNodeKind.value = runningKind || currentKind
  }

  Events.On('execution:state', (e: any) => {
    const d = eventPayload(e)
    if (!d) return
    running.value = !!d.Running
    runId.value = Number(d.RunID ?? 0)
    source.value = (d.Source ?? '') as WorkerStateEvent['source']
    targets.value = Array.isArray(d.Targets) ? d.Targets : []
    targetIdx.value = Number(d.TargetIdx ?? 0)
    currentTargetID.value = targets.value[targetIdx.value]?.id ?? ''
    // worker 转为 idle 时清 node 状态；若带 Error → toast
    if (!running.value) {
      currentNodeID.value = ''
      currentNodeKind.value = ''
      const errMsg = d.Error ? errorMessage(d.Error) : ''
      lastError.value = errMsg || null
      if (errMsg) toastError(errMsg, i18n.global.t('execution.run_failed'))
    }
  })

  // 后端 200ms 累积一批 node-enter event 发 batch (省 IPC). 前端取 last 1 个作为
  // currentNode (覆盖, batch 内中间节点不渲染 — 高亮只看最新). 单条 node-enter 兼容保留.
  Events.On('container:node-enter-batch', (e: any) => {
    if (!running.value && !debugActive.value) return
    const d = eventPayload(e)
    const entries = d?.entries
    if (!Array.isArray(entries) || entries.length === 0) return
    const last = entries[entries.length - 1]
    const nodeID = String(last?.nodeId ?? '')
    const nodeKind = String(last?.nodeKind ?? '')
    if (running.value) {
      currentNodeID.value = nodeID
      currentNodeKind.value = nodeKind
    }
    if (debugActive.value) {
      debugRunningNodeID.value = nodeID
      debugRunningNodeKind.value = nodeKind
    }
  })

  Events.On('container:node-enter', (e: any) => {
    if (!running.value && !debugActive.value) return
    const d = eventPayload(e)
    if (!d) return
    const nodeID = String(d.nodeId ?? '')
    const nodeKind = String(d.nodeKind ?? '')
    if (running.value) {
      currentNodeID.value = nodeID
      currentNodeKind.value = nodeKind
    }
    if (debugActive.value) {
      debugRunningNodeID.value = nodeID
      debugRunningNodeKind.value = nodeKind
    }
  })

  Events.On('debug:state', (e: any) => {
    applyDebugState(eventPayload(e))
  })

  return {
    running,
    runId,
    source,
    targets,
    targetIdx,
    currentTargetID,
    currentNodeID,
    currentNodeKind,
    lastError,
    debugSessionID,
    debugContainerID,
    debugStatus,
    debugMode,
    debugStartNodeID,
    debugCurrentNodeID,
    debugCurrentNodeKind,
    debugRunningNodeID,
    debugRunningNodeKind,
    debugLastNodeID,
    debugLastNodeKind,
    debugLastExit,
    debugFailedNodeID,
    debugQueue,
    debugWarnings,
    debugError,
    debugTerminal,
    debugActive,
    debugBusy,
    debugCanStep,
    debugCanContinue,
    debugCanPause,
    debugNextNodeID,
    debugNextNodeKind,
    applyDebugState,
    clearDebugState,
  }
})
