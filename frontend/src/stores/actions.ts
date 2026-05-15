// actions store —— 录制/管理/执行用户自定义动作。
// 跟长跑型 bot 不一样：actions 一次跑完即结束，state 走 running/idle/error
// 而不是 idle/running/paused。所以不进 bot-registry。
//
// 事件订阅在 lib/events.ts 里集中 wire，这里只暴露 setter。
//
// Plan 1 数据层（spec 2026-05-15 容器架构）：Action 退化为纯录制片段，元数据 / 循环 /
// 热键 / 模板检测全部搬到 Container 层。

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { backend, type ActionStateEvent, type RecorderStateEvent } from '@/lib/backend'

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export type StepKind =
  | 'click_left'
  | 'click_middle'
  | 'click_right'
  | 'key'
  | 'key_down'
  | 'key_up'
  | 'sleep'
  | 'mouse_move'
  | 'mouse_drag'
  | 'scroll'
  | 'mouse_move_rel'

export interface Step {
  id: string
  kind: StepKind
  xRatio?: number
  yRatio?: number
  durationMs?: number
  vk?: string
  // mouse_drag
  x2Ratio?: number
  y2Ratio?: number
  mouseButton?: 'left' | 'middle' | 'right'
  // scroll
  scrollDelta?: number
  // mouse_move_rel
  dx?: number
  dy?: number
}

export interface VariableDecl {
  name: string
  type: 'number' | 'bool' | 'string' | 'point'
  default?: any
}

export interface Action {
  schemaVersion?: number // = 2
  id: string
  name: string
  description?: string
  recordingContext?: {
    resolution?: [number, number]
    dpiScale?: number
  }
  params?: VariableDecl[]
  steps: Step[]
  createdAt: string
  updatedAt: string
}

export const useActionsStore = defineStore('actions', () => {
  const list = ref<Action[]>([])
  const runningID = ref<string | null>(null)
  // 当前 running action 的 step 进度，给卡片进度条 / 编辑器高亮 / 浮窗 mini-status 用。
  // 由后端 action:state status='step' 事件驱动。
  const runningStepIndex = ref(0)
  const runningStepTotal = ref(0)
  const runningStepID = ref<string>('')
  const runningLoopIter = ref(0)
  const recording = ref(false)
  // 录制 dialog 实时显示计数；由 action:recorder-event 累计。
  const recordingStepCount = ref(0)
  // 倒计时阶段：用户点"录制"后倒数 3-2-1，到 0 才真调 backend.StartRecording。
  // null = 不在倒计时；> 0 = 还剩秒数。dialog 用这个 ref 渲染大字。
  const countdown = ref<number | null>(null)

  async function reload() {
    const r = await backend.actions.list()
    if (r !== undefined) list.value = r as unknown as Action[]
  }

  async function create(name: string) {
    const a = await backend.actions.create(name)
    await reload()
    return a as unknown as Action | undefined
  }

  async function update(id: string, patch: object) {
    const r = await backend.actions.update(id, JSON.stringify(patch))
    if (r === undefined) return false
    await reload()
    return true
  }

  async function remove(id: string) {
    const r = await backend.actions.delete_(id)
    if (r === undefined) return false
    list.value = list.value.filter((a) => a.id !== id)
    return true
  }

  async function run(id: string) {
    return backend.actions.runOnce(id)
  }
  async function stop() {
    return backend.actions.stopRunning()
  }

  // startRecording 走 3 秒倒计时再真启录。期间用户可点 dialog 上的取消中止。
  // 倒计时跑完 backend 启录失败 → reset 状态返 error 让 caller toast。
  // 已经在倒计时 / 已经在录 → 静默忽略（幂等）。
  // 期间用户按 Ctrl+Shift+R 触发 backend 启录 → onRecorderEvent 把 countdown 清零，
  // 循环检测到提前结束（recording=true 或 countdown=null）。
  async function startRecording() {
    if (recording.value || countdown.value !== null) return
    recordingStepCount.value = 0
    countdown.value = 3
    for (let i = 3; i > 0; i--) {
      countdown.value = i
      await sleep(1000)
      // 取消 / 热键已抢先启录 → 退出，不再调 backend
      if (countdown.value === null || recording.value) return
    }
    countdown.value = null
    await backend.actions.startRecording()
  }

  // cancelCountdown 用户在倒计时阶段取消。
  function cancelCountdown() {
    countdown.value = null
  }

  async function stopRecording() {
    const r = await backend.actions.stopRecording()
    await reload()
    return r as unknown as Action | undefined
  }
  async function cancelRecording() {
    // 倒计时阶段就是 frontend-only，没真启录，直接清状态
    if (countdown.value !== null) {
      countdown.value = null
      return
    }
    return backend.actions.cancelRecording()
  }

  // toggleRecording UI 按钮 + Ctrl+Shift+R 全局热键 都走这条。统一三态切换：
  //   - 录制中 → 停录（落库）
  //   - 倒计时中 → 取消倒计时
  //   - 闲置 → 启动倒计时 → 启录
  // 走前端一条路径保证两种触发方式 UX 完全一致（都显示倒计时 + dialog）。
  async function toggleRecording() {
    if (recording.value) {
      await stopRecording()
      return
    }
    if (countdown.value !== null) {
      cancelCountdown()
      return
    }
    await startRecording()
  }

  // 由 lib/events.ts 转发后端事件
  function onStateEvent(ev: ActionStateEvent) {
    switch (ev.status) {
      case 'running':
        runningID.value = ev.actionId
        runningStepIndex.value = 0
        runningStepTotal.value = 0
        runningStepID.value = ''
        runningLoopIter.value = 0
        break
      case 'step':
        runningStepIndex.value = ev.stepIndex ?? 0
        runningStepTotal.value = ev.stepTotal ?? 0
        runningStepID.value = ev.stepId ?? ''
        runningLoopIter.value = ev.loopIter ?? 0
        break
      case 'idle':
      case 'error':
        runningID.value = null
        runningStepIndex.value = 0
        runningStepTotal.value = 0
        runningStepID.value = ''
        runningLoopIter.value = 0
        break
    }
  }
  function onRecorderEvent(ev: RecorderStateEvent) {
    if (ev.status === 'started') {
      recording.value = true
      recordingStepCount.value = 0
      // 如果是热键抢在倒计时之前启录，把倒计时清掉避免后续 backend 再调一次
      countdown.value = null
    } else {
      // stopped / error
      recording.value = false
      // 后端通过 Ctrl+Shift+R 触发 StopRecording 时只 emit 事件不调前端 store —— 这里要主动 reload，
      // 否则全局热键停录后列表不会出现新 action。dialog 按钮路径走 stopRecording() 已经 reload 过，
      // 再 reload 一次幂等无害。
      if (ev.status === 'stopped') {
        void reload()
      }
    }
  }
  function incRecorderEventCount() {
    recordingStepCount.value++
  }

  return {
    list,
    runningID,
    runningStepIndex,
    runningStepTotal,
    runningStepID,
    runningLoopIter,
    recording,
    recordingStepCount,
    countdown,
    reload,
    create,
    update,
    remove,
    run,
    stop,
    startRecording,
    stopRecording,
    cancelRecording,
    cancelCountdown,
    toggleRecording,
    onStateEvent,
    onRecorderEvent,
    incRecorderEventCount,
  }
})
