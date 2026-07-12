// 子图 → 等价脚本纯函数转换器。
// Spec 驱动零 kind 特判, 结构性特例仅: 出入口 marker / Subgraph·CollapsedNode(→Subgraph()
// 调用, 出口 decl ID→Name) / GetParam(→params.get)。任何不支持结构整体拒转 — 攒全清单
// 返回, 不出半对的代码。
import type { NodeKindSpec } from '@/components/containers/nodeRegistry'
import type { GraphEdge, GraphNode } from '@/lib/backend'

export interface UnsupportedItem {
  nodeID: string
  kind: string
  reason: string // i18n key: subgraphScript.reason.*
}

// 转换器只需子图这几个结构字段; 用结构类型让前端 SubgraphSummary(editorStore 里的完整
// 子图载体)与后端 Subgraph 都能直接喂进来, 不必关心 createdAt 等无关字段。
export interface SubgraphLike {
  id: string
  label: string
  graph: { nodes: GraphNode[]; edges: GraphEdge[] }
  entry: { nodeID: string }
  outputPins: { id: string; name: string; nodeID?: string }[]
}

export interface ConvertCtx {
  specFor: (kind: string) => NodeKindSpec | undefined
  bindable: ReadonlySet<string>
  subgraphsById: ReadonlyMap<string, SubgraphLike>
}

export type ConvertResult =
  | { ok: true; code: string }
  | { ok: false; unsupported: UnsupportedItem[] }

const SUBGRAPH_KINDS = new Set(['Subgraph', 'CollapsedNode'])
const STRUCTURAL_KINDS = new Set(['Loop', 'Break', 'Continue'])
const R = (k: string) => `subgraphScript.reason.${k}`

const STANDARD_EXIT_CONSTANTS = new Map<string, string>([
  ['Body', 'Body'],
  ['Changed', 'Changed'],
  ['default', 'Default'],
  ['Done', 'Done'],
  ['Fail', 'Fail'],
  ['False', 'False'],
  ['Found', 'Found'],
  ['Gone', 'Gone'],
  ['NotFound', 'NotFound'],
  ['Out', 'Out'],
  ['Stable', 'Stable'],
  ['Timeout', 'Timeout'],
  ['True', 'True'],
])

interface Edge {
  fromNode: string
  fromPin: string
  toNode: string
  toPin: string
}

function splitRef(ref: string): [string, string] {
  const i = ref.indexOf('.')
  return i < 0 ? [ref, ''] : [ref.slice(0, i), ref.slice(i + 1)]
}

function subgraphIDOf(n: GraphNode): string {
  const cfg = n.config ?? {}
  const lit = (cfg.literal ?? {}) as Record<string, unknown>
  return String(cfg.SubgraphID ?? lit.SubgraphID ?? '')
}

function literalOf(n: GraphNode): Record<string, unknown> {
  return ((n.config ?? {}).literal ?? {}) as Record<string, unknown>
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function replaceIdentifier(src: string, name: string, value: string): string {
  const re = new RegExp(`(^|[^A-Za-z0-9_$])(${escapeRegExp(name)})(?=[^A-Za-z0-9_$]|$)`, 'g')
  return src.replace(re, (_m, prefix) => `${prefix}${value}`)
}

export function subgraphToScript(sg: SubgraphLike, ctx: ConvertCtx): ConvertResult {
  const problems: UnsupportedItem[] = []
  const bad = (n: GraphNode, key: string) => {
    if (!problems.some((p) => p.nodeID === n.id && p.reason === R(key)))
      problems.push({ nodeID: n.id, kind: n.kind, reason: R(key) })
  }

  const nodeById = new Map<string, GraphNode>()
  for (const n of sg.graph.nodes) nodeById.set(n.id, n)
  const exitNameByMarker = new Map<string, string>()
  for (const p of sg.outputPins) if (p.nodeID) exitNameByMarker.set(p.nodeID, p.name)
  const entryID = sg.entry.nodeID

  // ---- 边分类: data 边 iff from-pin ∈ 源 kind 的 dataOut (同后端 IsDataOutPin);
  // Subgraph/CollapsedNode 出口 pin 是 callee decl ID, spec 上看不到, 特判成 exec。
  const execEdges: Edge[] = []
  const dataEdges: Edge[] = []
  for (const e of sg.graph.edges as GraphEdge[]) {
    const [fromNode, fromPin] = splitRef(e.from)
    const [toNode, toPin] = splitRef(e.to)
    const edge: Edge = { fromNode, fromPin, toNode, toPin }
    const src = nodeById.get(fromNode)
    if (fromNode === entryID || !src) {
      execEdges.push(edge) // entry marker, 或源是出口 marker 之类(不该有出边, 容错按 exec)
      continue
    }
    if (SUBGRAPH_KINDS.has(src.kind)) {
      const callee = ctx.subgraphsById.get(subgraphIDOf(src))
      const isExit = fromPin === 'Fail' || !!callee?.outputPins.some((p) => p.id === fromPin)
      ;(isExit ? execEdges : dataEdges).push(edge)
      continue
    }
    const spec = ctx.specFor(src.kind)
    ;(spec && fromPin in spec.dataOut ? dataEdges : execEdges).push(edge)
  }

  // exec 出边按 (节点, 出口 pin) 索引; data 入边按 (节点, 入 pin) 索引。
  const execOutByNode = new Map<string, Map<string, string[]>>()
  for (const e of execEdges) {
    let pins = execOutByNode.get(e.fromNode)
    if (!pins) execOutByNode.set(e.fromNode, (pins = new Map()))
    const targets = pins.get(e.fromPin) ?? []
    targets.push(e.toNode)
    pins.set(e.fromPin, targets)
  }
  const dataInByNode = new Map<string, Map<string, { src: string; pin: string }>>()
  const dataConsumers = new Map<string, number>() // 源节点 → 被引用次数
  for (const e of dataEdges) {
    let pins = dataInByNode.get(e.toNode)
    if (!pins) dataInByNode.set(e.toNode, (pins = new Map()))
    pins.set(e.toPin, { src: e.fromNode, pin: e.fromPin })
    dataConsumers.set(e.fromNode, (dataConsumers.get(e.fromNode) ?? 0) + 1)
  }

  // ---- 全图校验 (攒全, 不 fail-fast) ----
  for (const n of sg.graph.nodes) {
    const spec = ctx.specFor(n.kind)
    if (n.disabled) bad(n, 'disabled')
    if (SUBGRAPH_KINDS.has(n.kind)) {
      if (!ctx.subgraphsById.get(subgraphIDOf(n))) bad(n, 'callee_missing')
    } else if (STRUCTURAL_KINDS.has(n.kind)) {
      // Script has native loops and control transfer; these are rendered as JS syntax.
    } else if (!spec || !ctx.bindable.has(n.kind)) {
      if (!spec?.isVisualOnly) bad(n, 'not_bindable')
      continue
    }
    if (spec?.dynamicInputs && n.kind !== 'Expr') bad(n, 'dynamic_inputs')
    if (
      spec?.isPureData &&
      (dataConsumers.get(n.id) ?? 0) > 0 &&
      Object.keys(spec.dataOut).length > 1
    )
      bad(n, 'multi_out_pure')
    // Fail(error 语义)出口接线 → 图里是错误路由, 脚本里是异常, v1 不自动生成 try/catch。
    const wiredPins = execOutByNode.get(n.id)
    if (wiredPins) {
      const errorPins = SUBGRAPH_KINDS.has(n.kind) ? ['Fail'] : (spec?.errorOut ?? [])
      for (const [pin, targets] of wiredPins) {
        if (errorPins.includes(pin)) bad(n, 'fail_edge')
        if (targets.length > 1) bad(n, 'fan_out')
      }
    }
  }
  // exec 成环: 从 entry 沿 exec 边 DFS, 灰节点重入 = 环。
  {
    const state = new Map<string, 1 | 2>() // 1=在栈上 2=完成
    const visit = (id: string): void => {
      if (state.get(id) === 2) return
      if (state.get(id) === 1) {
        const n = nodeById.get(id)
        if (n) bad(n, 'cycle')
        return
      }
      state.set(id, 1)
      for (const targets of execOutByNode.get(id)?.values() ?? []) for (const t of targets) visit(t)
      state.set(id, 2)
    }
    visit(entryID)
  }
  if (problems.length > 0) return { ok: false, unsupported: problems }

  // ---- 生成 ----
  const lines: string[] = []
  let varSeq = 0
  const resultVar = new Map<string, string>() // exec 节点 → const 名

  const renderValue = (v: unknown): string => JSON.stringify(v)
  const exitExpr = (name: string, opts?: { subgraph?: boolean }): string => {
    if (opts?.subgraph) return renderValue(name)
    const constant = STANDARD_EXIT_CONSTANTS.get(name)
    return constant ? `Exit.${constant}` : renderValue(name)
  }

  // 纯函数节点求值表达式 (递归解析它自己的 data 入边)。
  const pureExpr = (n: GraphNode, ancestors: Set<string>, indent: string): string => {
    const lit = literalOf(n)
    if (n.kind === 'GetParam') return `params.get(${renderValue(String(lit.ParamName ?? ''))})`
    if (n.kind === 'Expr') return exprSource(n, ancestors, indent)
    return `${n.kind}({${argEntries(n, ancestors, indent)}})`
  }

  const exprSource = (n: GraphNode, ancestors: Set<string>, indent: string): string => {
    const lit = literalOf(n)
    let src = String(lit.Expression ?? lit.Expr ?? n.config?.Expression ?? n.config?.Expr ?? '')
    const inputs = (n.config?.Inputs ?? []) as Array<{ Name?: unknown }>
    const wired = dataInByNode.get(n.id)
    for (const d of inputs) {
      const name = String(d.Name ?? '')
      if (!name) continue
      const w = wired?.get(name)
      if (!w) continue
      src = replaceIdentifier(src, name, dataRef(n, w, ancestors, indent))
    }
    return `(${src})`
  }

  // data 入 pin → 表达式 (来源: 纯函数内联/提升 const、exec 祖先的 rN.字段)。
  const dataRef = (
    consumer: GraphNode,
    src: { src: string; pin: string },
    ancestors: Set<string>,
    indent: string,
  ): string => {
    const srcNode = nodeById.get(src.src)
    if (!srcNode) {
      bad(consumer, 'cross_branch_data')
      return 'undefined'
    }
    const spec = ctx.specFor(srcNode.kind)
    if (spec?.isPureData) {
      return pureExpr(srcNode, ancestors, indent)
    }
    const rv = resultVar.get(srcNode.id)
    if (!rv || !ancestors.has(srcNode.id)) {
      bad(consumer, 'cross_branch_data')
      return 'undefined'
    }
    return `${rv}.${src.pin}`
  }

  // 调用参数对象内容: spec.dataIn 顺序, data 边 > literal, 与默认值相等省略。
  const argEntries = (n: GraphNode, ancestors: Set<string>, indent: string): string => {
    const spec = ctx.specFor(n.kind)
    const lit = (n.config?.literal ?? {}) as Record<string, unknown>
    const defaults = (spec?.defaults?.literal ?? {}) as Record<string, unknown>
    const wired = dataInByNode.get(n.id)
    const parts: string[] = []
    for (const pin of Object.keys(spec?.dataIn ?? {})) {
      const w = wired?.get(pin)
      if (w) {
        parts.push(`${pin}: ${dataRef(n, w, ancestors, indent)}`)
        continue
      }
      const v = lit[pin]
      if (v === undefined || v === null) continue
      if (JSON.stringify(v) === JSON.stringify(defaults[pin])) continue
      parts.push(`${pin}: ${renderValue(v)}`)
    }
    return parts.length > 0 ? ` ${parts.join(', ')} ` : ''
  }

  // Subgraph/CollapsedNode → Subgraph({SubgraphID, ...字面入参}); Params JSON 字面量平铺。
  const subgraphCallExpr = (n: GraphNode, ancestors: Set<string>, indent: string): string => {
    const id = subgraphIDOf(n)
    const wired = dataInByNode.get(n.id)
    const parts = [`SubgraphID: ${renderValue(id)}`]
    const lit = (n.config?.literal ?? {}) as Record<string, unknown>
    const params = lit.Params
    if (wired && wired.size > 0) {
      // callee 声明入参的 data-in pin(动态), 名字即参数名。
      for (const [pin, src] of wired) {
        if (pin === 'In') continue
        parts.push(`${pin}: ${dataRef(n, src, ancestors, indent)}`)
      }
    }
    if (params && typeof params === 'object') {
      for (const [k, v] of Object.entries(params as Record<string, unknown>)) {
        parts.push(`${k}: ${renderValue(v)}`)
      }
    }
    return `Subgraph({ ${parts.join(', ')} })`
  }

  // 从某节点起沿 exec 链生成语句; 多出口 → if/else if 递归。
  const emitChain = (
    startID: string | undefined,
    indent: string,
    ancestors: Set<string>,
    opts: { marker: 'return' | 'stop' } = { marker: 'return' },
  ): void => {
    let curID = startID
    while (curID) {
      const exitName = exitNameByMarker.get(curID)
      if (exitName !== undefined) {
        if (opts.marker === 'return') lines.push(`${indent}return ${renderValue(exitName)}`)
        return
      }
      const n = nodeById.get(curID)
      if (!n) return
      if (n.kind === 'Break') {
        lines.push(`${indent}break`)
        return
      }
      if (n.kind === 'Continue') {
        lines.push(`${indent}continue`)
        return
      }
      if (n.kind === 'Loop') {
        const lit = literalOf(n)
        const mode = String(lit.Mode ?? n.config?.Mode ?? 'count')
        const count = lit.Count ?? n.config?.Count ?? 10
        if (mode === 'forever') {
          lines.push(`${indent}while (true) {`)
        } else {
          const idx = `i${++varSeq}`
          lines.push(`${indent}for (let ${idx} = 0; ${idx} < ${renderValue(count)}; ${idx}++) {`)
        }
        const bodyTarget = execOutByNode.get(n.id)?.get('Body')?.[0]
        const nextAncestors = new Set(ancestors)
        nextAncestors.add(n.id)
        emitChain(bodyTarget, `${indent}  `, nextAncestors, { marker: 'stop' })
        lines.push(`${indent}}`)
        curID = execOutByNode.get(n.id)?.get('Done')?.[0]
        ancestors.add(n.id)
        continue
      }
      const isSub = SUBGRAPH_KINDS.has(n.kind)
      const spec = ctx.specFor(n.kind)
      const callee = isSub ? ctx.subgraphsById.get(subgraphIDOf(n)) : undefined

      // 可能出口集 (error 出口已在校验期保证未接线, 不参与分支)。
      const exits: { pin: string; name: string }[] = isSub
        ? (callee?.outputPins ?? []).map((p) => ({ pin: p.id, name: p.name }))
        : (spec?.execOutFn?.(n.config) ?? spec?.execOut ?? [])
            .filter((p) => !(spec?.errorOut ?? []).includes(p))
            .map((p) => ({ pin: p, name: p }))
      const wiredPins = execOutByNode.get(n.id)
      const connected = exits.filter((x) => wiredPins?.has(x.pin))

      const consumed = (dataConsumers.get(n.id) ?? 0) > 0
      const branching = exits.length > 1
      const callExpr = isSub
        ? subgraphCallExpr(n, ancestors, indent)
        : `${n.kind}({${argEntries(n, ancestors, indent)}})`
      const comment = n.label && n.label !== n.kind ? ` // ${n.label}` : ''

      if (branching || consumed) {
        const rv = `r${++varSeq}`
        resultVar.set(n.id, rv)
        lines.push(`${indent}const ${rv} = ${callExpr}${comment}`)
      } else {
        lines.push(`${indent}${callExpr}${comment}`)
      }
      const nextAncestors = new Set(ancestors)
      nextAncestors.add(n.id)

      if (!branching) {
        const targets = connected[0] ? wiredPins!.get(connected[0].pin)! : []
        curID = targets[0]
        ancestors.add(n.id)
        continue
      }
      const rv = resultVar.get(n.id)!
      connected.forEach((x, i) => {
        const head = i === 0 ? `${indent}if` : `${indent}} else if`
        lines.push(`${head} (${rv}.exit === ${exitExpr(x.name, { subgraph: isSub })}) {`)
        emitChain(wiredPins!.get(x.pin)![0], `${indent}  `, new Set(nextAncestors), opts)
      })
      if (connected.length > 0) lines.push(`${indent}}`)
      return
    }
  }

  const entryTargets = execOutByNode.get(entryID)?.values().next().value as string[] | undefined
  emitChain(entryTargets?.[0], '', new Set())

  if (problems.length > 0) return { ok: false, unsupported: problems }
  return { ok: true, code: lines.join('\n') }
}
