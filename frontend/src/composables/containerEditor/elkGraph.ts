// elkGraph.ts —— 自动布局纯函数集(可单测，无 vue-flow / 全局 registry 依赖)。
import type { GraphNode, GraphEdge } from '@/lib/backend'
import type { ElkNode } from './elkConfig'

export interface Size { width: number; height: number }

const ROW_H = 26 // 每个 pin 行约高
export function estimateNodeSize(kind: string, cfg: Record<string, any> = {}): Size {
  switch (kind) {
    case 'CommentBox':
      return { width: Number(cfg.width) || 320, height: Number(cfg.height) || 160 }
    case 'Switch': {
      const n = Array.isArray(cfg.cases) ? cfg.cases.length : 1
      return { width: 240, height: 90 + n * ROW_H }
    }
    case 'Parallel':
    case 'Race': {
      const n = Number(cfg.n) || 2
      return { width: 240, height: 90 + n * ROW_H }
    }
    case 'Expr': {
      const n = Array.isArray(cfg.inputs) ? cfg.inputs.length : 1
      return { width: 240, height: 80 + n * ROW_H }
    }
    default:
      return { width: 220, height: 90 }
  }
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
  // 有边时只保留被连接的节点；无边时全部参与（如单独展示动态 pin）。
  const connected = new Set<string>()
  for (const e of edges) {
    connected.add(srcId(e.from))
    connected.add(srcId(e.to))
  }
  const layoutNodes = edges.length > 0 ? nodes.filter((n) => connected.has(n.id)) : nodes

  const inSide = opts.direction === 'RIGHT' ? 'WEST' : 'NORTH'
  const outSide = opts.direction === 'RIGHT' ? 'EAST' : 'SOUTH'

  const children: ElkNode[] = layoutNodes.map((n) => {
    const spec = opts.getSpec(n.kind)
    const cfg = n.config ?? {}
    const size = opts.getDims(n.id, n.kind) ?? estimateNodeSize(n.kind, cfg)
    const ports = spec
      ? [
          ...inputPins(spec, cfg).map((p) => ({
            id: `${n.id}::${p}`,
            layoutOptions: { 'elk.port.side': inSide },
          })),
          ...outputPins(spec, cfg).map((p) => ({
            id: `${n.id}::${p}`,
            layoutOptions: { 'elk.port.side': outSide },
          })),
        ]
      : []
    return {
      id: n.id,
      width: size.width,
      height: size.height,
      ports,
      layoutOptions: { 'elk.portConstraints': 'FIXED_ORDER' },
    } as unknown as ElkNode
  })

  const kindOf = (id: string) => layoutNodes.find((n) => n.id === id)?.kind
  // exec 边：from 节点的 spec.dataOut 里没有该 pin → 视为 exec 边，priority 高。
  const isData = (fromKind: string | undefined, pin: string): boolean =>
    !!(fromKind && opts.getSpec(fromKind)?.dataOut?.[pin])

  const elkEdges = edges
    .filter((e) => connected.has(srcId(e.from)) && connected.has(srcId(e.to)))
    .map((e, i) => {
      const dataEdge = isData(kindOf(srcId(e.from)), srcPin(e.from))
      return {
        id: `e${i}`,
        sources: [`${srcId(e.from)}::${srcPin(e.from)}`],
        targets: [`${srcId(e.to)}::${srcPin(e.to)}`],
        layoutOptions: { __priority: dataEdge ? DATA_PRIORITY : EXEC_PRIORITY },
      }
    })

  return { id: 'root', children, edges: elkEdges } as unknown as ElkNode
}
