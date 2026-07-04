// yt 脚本控制台 —— 纯执行器 (不碰 Vue / Pinia / 草稿)。
// Vue 层把节点组装成 NodeModel[] 喂进来, 跑用户 JS, 产出"待应用变更 / 被拒 / 日志"。
// 应用(写草稿+子图+撤销) 与 UI 都在 Vue 层, 不在这里 —— 故本文件可纯单测。

// 前端 pin 类型 (来自 nodeRegistry, 后端 Number/Integer/Duration 都塌成 'number')。
export type PinType = 'number' | 'bool' | 'string' | 'point' | 'any' | 'list' | 'file'

export interface NodeModel {
  id: string
  kind: string
  label: string
  sgID: string | null // null = 主图; 否则所在子图 id
  literal: Record<string, unknown> // 当前 config.literal
  defaults: Record<string, unknown> // KIND_DEFAULTS[kind] (get 回退)
  specPins: Record<string, PinType> // PIN_SPECS[kind].dataIn (has + 归一类型)
}

export interface Change {
  nodeId: string
  sgID: string | null
  pin: string
  value: unknown
}
export interface Reject {
  nodeId: string
  kind: string
  pin: string
  reason: string
}
export interface RunResult {
  applied: Change[]
  rejected: Reject[]
  logs: string[]
  error?: string
  nodeCount: number // 不同节点数
  pinCount: number // 不同 (节点,pin) 对数
}

export interface RunContext {
  nodes: NodeModel[]
  selectedIds: string[]
  container: { id: string; name: string }
}

// 把值按 pin 类型归一; 返回 { ok, value } 或 { ok:false, reason }。
function coerce(type: PinType, v: unknown): { ok: true; value: unknown } | { ok: false; reason: string } {
  switch (type) {
    case 'number': {
      const n = Number(v)
      if (!Number.isFinite(n)) return { ok: false, reason: `${JSON.stringify(v)} is not a finite number` }
      return { ok: true, value: n }
    }
    case 'bool': {
      if (typeof v === 'boolean') return { ok: true, value: v }
      if (v === 0 || v === 1) return { ok: true, value: v === 1 }
      if (v === 'true') return { ok: true, value: true }
      if (v === 'false') return { ok: true, value: false }
      return { ok: false, reason: `${JSON.stringify(v)} is not a boolean` }
    }
    case 'string':
      return { ok: true, value: String(v) }
    default: {
      // point / list / file / any: 原样写, 但拒非 JSON-可序列化 (函数 / 循环引用等)。
      try {
        if (JSON.stringify(v) === undefined) return { ok: false, reason: 'not JSON-serializable' }
      } catch {
        return { ok: false, reason: 'not JSON-serializable (circular?)' }
      }
      return { ok: true, value: v }
    }
  }
}

export function runConsoleScript(code: string, ctx: RunContext): RunResult {
  // overlay: nodeId → (pin → 归一后的值)。后写覆盖。
  const overlay = new Map<string, Map<string, unknown>>()
  const rejected: Reject[] = []
  const logs: string[] = []

  const handleById = new Map<string, Readonly<unknown>>()
  for (const m of ctx.nodes) {
    const handle = {
      get id() {
        return m.id
      },
      get kind() {
        return m.kind
      },
      get label() {
        return m.label
      },
      get sgID() {
        return m.sgID
      },
      has(pin: string): boolean {
        return Object.prototype.hasOwnProperty.call(m.specPins, pin)
      },
      get(pin: string): unknown {
        const ov = overlay.get(m.id)
        if (ov && ov.has(pin)) return ov.get(pin)
        if (Object.prototype.hasOwnProperty.call(m.literal, pin)) return m.literal[pin]
        if (Object.prototype.hasOwnProperty.call(m.defaults, pin)) return m.defaults[pin]
        return undefined
      },
      set(pin: string, value: unknown): void {
        const type = m.specPins[pin]
        if (type === undefined) {
          rejected.push({ nodeId: m.id, kind: m.kind, pin, reason: `node (${m.kind}) has no input pin "${pin}"` })
          return
        }
        const c = coerce(type, value)
        if (!c.ok) {
          rejected.push({ nodeId: m.id, kind: m.kind, pin, reason: c.reason })
          return
        }
        let ov = overlay.get(m.id)
        if (!ov) {
          ov = new Map()
          overlay.set(m.id, ov)
        }
        ov.set(pin, c.value)
      },
    }
    Object.freeze(handle)
    handleById.set(m.id, handle)
  }

  const nodes = Object.freeze(ctx.nodes.map((m) => handleById.get(m.id)))
  const selectedSet = new Set(ctx.selectedIds)
  const selected = Object.freeze(ctx.nodes.filter((m) => selectedSet.has(m.id)).map((m) => handleById.get(m.id)))
  const yt = Object.freeze({
    nodes,
    selected,
    container: Object.freeze({ id: ctx.container.id, name: ctx.container.name }),
    log: (...args: unknown[]) => {
      logs.push(args.map((a) => String(a)).join(' '))
    },
  })

  let error: string | undefined
  try {
    // strict mode: 少踩 JS 老坑 + 让对冻结对象的写直接抛错 (而非静默)。
    const fn = new Function('yt', '"use strict";\n' + code)
    fn(yt)
  } catch (e) {
    error = e instanceof Error ? (e.stack ?? e.message) : String(e)
  }

  // 抛异常 → 零变更 (丢弃 overlay)。否则按 model 顺序展平 overlay。
  if (error) {
    return { applied: [], rejected, logs, error, nodeCount: 0, pinCount: 0 }
  }
  const applied: Change[] = []
  let nodeCount = 0
  for (const m of ctx.nodes) {
    const ov = overlay.get(m.id)
    if (!ov || ov.size === 0) continue
    nodeCount++
    for (const [pin, value] of ov) applied.push({ nodeId: m.id, sgID: m.sgID, pin, value })
  }
  return { applied, rejected, logs, nodeCount, pinCount: applied.length }
}
