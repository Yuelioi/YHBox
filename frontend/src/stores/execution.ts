// 全局 worker 状态。容器/动作运行由唯一 worker 串行执行；这里订阅
// `execution:state` 事件给 AppStatusBar 渲染，并在错误结束时 toast 出来。

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { toastError } from '@/lib/invoke'

export interface WorkerStateEvent {
  running: boolean
  runId: number
  source: 'hotkey' | 'schedule' | 'manual' | ''
  targets?: { kind: string; id: string }[]
  targetIdx?: number
  error?: string
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

  Events.On('execution:state', (e: any) => {
    const d = e?.data ?? e
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
      const errMsg = typeof d.Error === 'string' ? d.Error : ''
      lastError.value = errMsg || null
      if (errMsg) toastError(errMsg, '运行失败')
    }
  })

  // 后端 200ms 累积一批 node-enter event 发 batch (省 IPC). 前端取 last 1 个作为
  // currentNode (覆盖, batch 内中间节点不渲染 — 高亮只看最新). 单条 node-enter 兼容保留.
  Events.On('container:node-enter-batch', (e: any) => {
    if (!running.value) return
    const d = e?.data ?? e
    const entries = d?.entries
    if (!Array.isArray(entries) || entries.length === 0) return
    const last = entries[entries.length - 1]
    currentNodeID.value = String(last?.nodeId ?? '')
    currentNodeKind.value = String(last?.nodeKind ?? '')
  })

  Events.On('container:node-enter', (e: any) => {
    if (!running.value) return
    const d = e?.data ?? e
    if (!d) return
    currentNodeID.value = String(d.nodeId ?? '')
    currentNodeKind.value = String(d.nodeKind ?? '')
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
  }
})
