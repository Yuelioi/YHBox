import { defineStore } from 'pinia'
import { ref } from 'vue'
import { parseLine, type LogLine } from '@/lib/logFormat'

const RING_CAP = 500

interface NodeEnterEntry {
  nodeId: string
  nodeKind: string
  count: number
}

interface ContainerLogPayload {
  level: string
  message: string
}

interface NodeDumpEntry {
  nodeId: string
  nodeKind: string
  lineKey: string
  line: string
  count: number
  final: boolean
}

export interface ActionTraceEntry {
  containerId?: string
  action: string
  source?: {
    containerId?: string
    nodeId?: string
    nodeKind?: string
    inPin?: string
    ContainerID?: string
    NodeID?: string
    NodeKind?: string
    InPin?: string
  }
  target?: {
    id?: string
    ID?: string
  }
  backend?: string
  request?: unknown
  result?: unknown
  status?: string
  error?: string
  coordinateSteps?: unknown[]
  startedAt?: string
  endedAt?: string
  durationMs?: number
}

function pushBounded(lines: LogLine[], line: LogLine) {
  lines.push(line)
  if (lines.length > RING_CAP) {
    lines.splice(0, lines.length - RING_CAP)
  }
}

function nowIso() {
  return new Date().toISOString()
}

function pushBoundedTrace(traces: ActionTraceEntry[], trace: ActionTraceEntry) {
  traces.push(trace)
  if (traces.length > RING_CAP) {
    traces.splice(0, traces.length - RING_CAP)
  }
}

function traceSource(trace: ActionTraceEntry) {
  const source = trace.source ?? {}
  const nodeKind = source.nodeKind ?? source.NodeKind ?? '?'
  const nodeId = source.nodeId ?? source.NodeID ?? '?'
  const inPin = source.inPin ?? source.InPin ?? ''
  return `${nodeKind}(${nodeId})${inPin ? '.' + inPin : ''}`
}

function traceTarget(trace: ActionTraceEntry) {
  return trace.target?.id ?? trace.target?.ID ?? ''
}

function formatActionTrace(trace: ActionTraceEntry) {
  const status = trace.status || 'unknown'
  const duration = Number.isFinite(trace.durationMs) ? ` ${trace.durationMs}ms` : ''
  const target = traceTarget(trace)
  const targetPart = target ? ` @ ${target}` : ''
  const backend = trace.backend ? ` via ${trace.backend}` : ''
  const error = trace.error ? `: ${trace.error}` : ''
  return `${traceSource(trace)} -> ${trace.action} ${status}${duration}${targetPart}${backend}${error}`
}

export const useLogStore = defineStore('log', () => {
  const lines = ref<LogLine[]>([])
  const actionTraces = ref<ActionTraceEntry[]>([])
  const lastSeq = ref(0)
  const dropDetected = ref(false)

  function appendSystem(seq: number, raw: string[]) {
    if (lastSeq.value !== 0 && seq !== lastSeq.value + 1) {
      dropDetected.value = true
      pushBounded(lines.value, {
        time: nowIso(),
        level: 'warn',
        message: `[log:lines] sequence gap: expected ${lastSeq.value + 1}, got ${seq}`,
        source: 'SYS',
      })
    }
    lastSeq.value = seq
    for (const r of raw) {
      pushBounded(lines.value, parseLine(r))
    }
  }

  function appendContainerLog(p: ContainerLogPayload) {
    pushBounded(lines.value, {
      time: nowIso(),
      level: p.level || 'info',
      message: p.message,
      source: 'CTR',
    })
  }

  function appendNodeEnter(entries: NodeEnterEntry[]) {
    for (const e of entries) {
      const suffix = e.count > 1 ? ` × ${e.count}` : ''
      pushBounded(lines.value, {
        time: nowIso(),
        level: 'node',
        message: `→ ${e.nodeKind} (${e.nodeId})${suffix}`,
        source: 'CTR',
      })
    }
  }

  function appendNodeDump(entries: NodeDumpEntry[]) {
    for (const e of entries) {
      const idx = lines.value.findIndex(
        (l) => l.nodeId === e.nodeId && l.lineKey === e.lineKey && !l.frozen,
      )
      if (idx >= 0) {
        const row = lines.value[idx]
        row.count = e.count
        row.message = e.line
        if (e.final) row.frozen = true
        continue
      }
      pushBounded(lines.value, {
        time: nowIso(),
        level: 'dump',
        message: e.line,
        source: 'CTR',
        nodeId: e.nodeId,
        lineKey: e.lineKey,
        count: e.count,
        frozen: e.final,
      })
    }
  }

  function appendActionTrace(trace: ActionTraceEntry) {
    pushBoundedTrace(actionTraces.value, trace)
    pushBounded(lines.value, {
      time: nowIso(),
      level: 'action',
      message: formatActionTrace(trace),
      source: 'CTR',
    })
  }

  function clear() {
    lines.value = []
    actionTraces.value = []
    lastSeq.value = 0
    dropDetected.value = false
  }

  return {
    lines,
    actionTraces,
    lastSeq,
    dropDetected,
    appendSystem,
    appendContainerLog,
    appendNodeEnter,
    appendNodeDump,
    appendActionTrace,
    clear,
  }
})
