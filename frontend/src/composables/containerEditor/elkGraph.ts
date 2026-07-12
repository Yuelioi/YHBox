// elkGraph.ts —— 自动布局纯函数集(可单测，无 vue-flow / 全局 registry 依赖)。
import type { GraphNode, GraphEdge } from '@/lib/backend'
import type { ElkNode } from './elkConfig'
import { SUBGRAPH_ENTRY_DEFAULT, SUBGRAPH_OUTPUT_DEFAULT } from './constants'

export interface Size {
  width: number
  height: number
}

const ROW_H = 26 // 每个 pin 行约高
// 节点尺寸估算 —— 仅当 vue-flow 实测 dimensions 拿不到时的兜底(刚开容器、节点未 mount)。
// 高度按 pinCount 估算; pinCount 由 buildElkGraph 从 registry(execOutFn/dataInDynamicFn)
// 正确派生后传入 —— 不在这里自己解析 cfg.cases/N/Inputs: 各 kind 存法不一
// (Parallel 读 literal.N、Expr 读顶层 Inputs ...)，自己解析必然 drift。基线 90≈2 行。
export function estimateNodeSize(kind: string, cfg: Record<string, any> = {}, pinCount = 0): Size {
  if (kind === 'CommentBox')
    return { width: Number(cfg.width) || 320, height: Number(cfg.height) || 160 }
  const extra = Math.max(0, pinCount - 2) * ROW_H
  return { width: 220, height: 90 + extra }
}

// ── 子图 virtual marker (入口/出口) ──────────────────────────────────────────
// 入口/出口 marker 不在 graph.nodes 里 (存 sg.entry / sg.outputPins, 各带自己的 x/y),
// 但 edges 按 `<nodeID>.<pin>` 引用它们。自动布局必须把它们当节点一起算 — 否则 marker
// 不被重排、连 marker 的边还会被 buildElkGraph 的 layoutIds 过滤丢掉 (body 丢约束 → 排乱)。
// 这里把它们合成 pseudo GraphNode 喂给 buildElkGraph; 布局后用 writeMarkerPositions 写回 metadata。
export interface MarkerLike {
  nodeID?: string
  x?: number
  y?: number
}

// 默认坐标走 SUBGRAPH_ENTRY_DEFAULT / SUBGRAPH_OUTPUT_DEFAULT (constants.ts), 跟 syncFlowFromDraft 一致。
export function subgraphMarkerNodes(
  entry: MarkerLike | null | undefined,
  outputPins: MarkerLike[] | undefined,
): GraphNode[] {
  const out: GraphNode[] = []
  if (entry?.nodeID) {
    out.push({
      id: entry.nodeID,
      kind: 'SubgraphInput',
      x: entry.x ?? SUBGRAPH_ENTRY_DEFAULT.x,
      y: entry.y ?? SUBGRAPH_ENTRY_DEFAULT.y,
      config: {},
    })
  }
  for (const p of outputPins ?? []) {
    if (!p.nodeID) continue
    out.push({
      id: p.nodeID,
      kind: 'SubgraphOutput',
      x: p.x ?? SUBGRAPH_OUTPUT_DEFAULT.x,
      y: p.y ?? SUBGRAPH_OUTPUT_DEFAULT.y,
      config: {},
    })
  }
  return out
}

// 把布局后的新坐标写回 sg.entry / sg.outputPins (按 nodeID 匹配)。直接 mutate (调用方负责
// touchSubgraph 标脏 + syncFlowFromDraft 重渲染 — 跟 onNodesChange 的 marker 拖动写回同路径)。
export function writeMarkerPositions(
  sg: { entry?: MarkerLike; outputPins?: MarkerLike[] },
  posById: Record<string, Pos>,
): boolean {
  let changed = false
  if (sg.entry?.nodeID && posById[sg.entry.nodeID]) {
    sg.entry.x = posById[sg.entry.nodeID].x
    sg.entry.y = posById[sg.entry.nodeID].y
    changed = true
  }
  for (const p of sg.outputPins ?? []) {
    if (p.nodeID && posById[p.nodeID]) {
      p.x = posById[p.nodeID].x
      p.y = posById[p.nodeID].y
      changed = true
    }
  }
  return changed
}

// ── buildElkGraph ──────────────────────────────────────────────────────────
// exec/data 的 priority 占位键 __priority；useElkLayout 组装时换成真实 ELK key。
const EXEC_PRIORITY = '100'
const DATA_PRIORITY = '1'

// 首个 '.' 处切分，支持含点 pin 名（如 "case.0"）。
const srcId = (s: string) => s.slice(0, s.indexOf('.'))
const srcPin = (s: string) => s.slice(s.indexOf('.') + 1)

interface MiniSpec {
  execIn: string[]
  execOut: string[]
  execOutFn?: (cfg: any) => string[]
  dataIn: Record<string, string>
  dataInDynamicFn?: (cfg: any) => Record<string, string>
  dataOut: Record<string, string>
}

export interface BuildOpts {
  getSpec: (kind: string) => MiniSpec | undefined
  getDims: (id: string, kind: string) => Size | null
  direction: 'RIGHT' | 'DOWN'
}

function inputPins(spec: MiniSpec, cfg: any): string[] {
  const dyn = spec.dataInDynamicFn ? Object.keys(spec.dataInDynamicFn(cfg)) : []
  return [...spec.execIn, ...Object.keys(spec.dataIn), ...dyn]
}

function outputPins(spec: MiniSpec, cfg: any): string[] {
  const execOut = spec.execOutFn ? spec.execOutFn(cfg) : spec.execOut
  return [...execOut, ...Object.keys(spec.dataOut)]
}

export function buildElkGraph(nodes: GraphNode[], edges: GraphEdge[], opts: BuildOpts): ElkNode {
  // 参与规则：节点有 ≥1 条边才进 ELK；无边(游离/注释)排除，交给 placeDetached 安置。
  const connected = new Set<string>()
  for (const e of edges) {
    connected.add(srcId(e.from))
    connected.add(srcId(e.to))
  }
  const layoutNodes = nodes.filter((n) => connected.has(n.id))
  const layoutIds = new Set(layoutNodes.map((n) => n.id))

  const children: ElkNode[] = layoutNodes.map((n) => {
    const spec = opts.getSpec(n.kind)
    const cfg = n.config ?? {}
    // pin 列表用 registry 正确派生(含动态 pin)，算一次给尺寸估算。
    // 注意：pin 不再当 ELK 端口、边也不连端口（见下方 elkEdges）。原来给每个 pin 建固定顺序
    // 端口 + 边连端口，ELK 拉直「端口↔端口」边时，端口在多口节点里高低不一，它只能整体上下
    // 挪节点去对齐 → 链式图被摊成往右上爬的楼梯（实测 FIXED_ORDER/SIDE/FREE 都歪；边连节点
    // 中心才排平）。我们又不渲染 ELK 的端口/边路由（VueFlow 自己画），去端口零渲染损失。
    const inP = spec ? inputPins(spec, cfg) : []
    const outP = spec ? outputPins(spec, cfg) : []
    const size =
      opts.getDims(n.id, n.kind) ?? estimateNodeSize(n.kind, cfg, inP.length + outP.length)
    return { id: n.id, width: size.width, height: size.height } as unknown as ElkNode
  })

  const kindOf = (id: string) => layoutNodes.find((n) => n.id === id)?.kind
  // exec 边：from 节点的 spec.dataOut 里没有该 pin → 视为 exec 边，priority 高。
  const isData = (fromKind: string | undefined, pin: string): boolean =>
    !!(fromKind && opts.getSpec(fromKind)?.dataOut?.[pin])

  const elkEdges = edges
    // 两端都须是参与布局的节点 —— 用 layoutIds 而非 connected: 悬空边(指向已删/不存在节点)
    // 的 id 也在 connected 里, 漏给 ELK 会因未知 port 报错; layoutIds 才是真在图里的节点。
    .filter((e) => layoutIds.has(srcId(e.from)) && layoutIds.has(srcId(e.to)))
    .map((e, i) => {
      const dataEdge = isData(kindOf(srcId(e.from)), srcPin(e.from))
      // 连节点(不连端口) —— exec/data 分类仍按 from 的 pin 名判, 只是 ELK 边端点用节点 id.
      return {
        id: `e${i}`,
        sources: [srcId(e.from)],
        targets: [srcId(e.to)],
        layoutOptions: { __priority: dataEdge ? DATA_PRIORITY : EXEC_PRIORITY },
      }
    })

  return { id: 'root', children, edges: elkEdges } as unknown as ElkNode
}

// ── anchorOffset / placeDetached ───────────────────────────────────────────

export interface Pos {
  x: number
  y: number
}
export interface BBox {
  minX: number
  minY: number
  maxX: number
  maxY: number
}

// ELK 布局后坐标系整体偏移量：使新布局重心与旧布局重心对齐，防整图跳位。
export function anchorOffset(
  oldP: Record<string, Pos>,
  newP: Record<string, Pos>,
): { dx: number; dy: number } {
  const center = (m: Record<string, Pos>) => {
    const ks = Object.keys(m)
    if (!ks.length) return { x: 0, y: 0 }
    let sx = 0,
      sy = 0
    for (const k of ks) {
      sx += m[k].x
      sy += m[k].y
    }
    return { x: sx / ks.length, y: sy / ks.length }
  }
  const o = center(oldP),
    n = center(newP)
  return { dx: o.x - n.x, dy: o.y - n.y }
}

interface DetachedNode {
  id: string
  x: number
  y: number
  width: number
  height: number
}

// 判断游离节点是否与主图包围盒重叠（任一轴不重叠即视为在外侧）。
function overlaps(n: DetachedNode, b: BBox): boolean {
  return n.x < b.maxX && n.x + n.width > b.minX && n.y < b.maxY && n.y + n.height > b.minY
}

// 游离/注释节点安置：与主图包围盒重叠的，停到主轴垂直方向的外侧空白并按序堆叠；
// RIGHT 布局→停下方(沿 y 堆叠)，DOWN 布局→停右侧(沿 x 堆叠)。不重叠的保持原位。
const DETACHED_MARGIN = 200
export function placeDetached(
  detached: DetachedNode[],
  bbox: BBox,
  direction: 'RIGHT' | 'DOWN',
): Record<string, Pos> {
  const out: Record<string, Pos> = {}
  let cursorY = bbox.maxY + DETACHED_MARGIN
  let cursorX = bbox.maxX + DETACHED_MARGIN
  for (const n of detached) {
    if (!overlaps(n, bbox)) {
      out[n.id] = { x: n.x, y: n.y }
      continue
    }
    if (direction === 'RIGHT') {
      out[n.id] = { x: bbox.minX, y: cursorY }
      cursorY += n.height + 40
    } else {
      out[n.id] = { x: cursorX, y: bbox.minY }
      cursorX += n.width + 40
    }
  }
  return out
}
