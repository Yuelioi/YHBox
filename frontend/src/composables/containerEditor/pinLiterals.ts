// 算"未连线的 data-in pin"+ 画布内联编辑写回 API。
// NodeInspector (literal section) 与 ContainerFlowNode (画布内联 input) 共用此纯函数。
//
// 连线判定按 target pin 名 (e.to) — 边占不占某 data-in 由 target slot 决定, 不看 source。
// pin 名在单节点内唯一, exec 边落 exec-in pin (名不同) 天然被 dataIn 名取交集忽略。
import type { ComputedRef, InjectionKey } from 'vue'
import type { PinType } from '@/components/containers/nodeRegistry/index'

export interface LiteralPin {
  name: string
  type: PinType
}

export function unconnectedDataInPins(
  kind: string,
  dataIn: Record<string, PinType>,
  config: Record<string, unknown> | null | undefined,
  edges: { from: string; to: string }[],
  nodeId: string,
): LiteralPin[] {
  const incoming = new Set<string>()
  for (const e of edges) {
    const [tgt, pin] = (e.to ?? '').split('.')
    if (tgt === nodeId && pin) incoming.add(pin)
  }
  const out: LiteralPin[] = []
  if (kind === 'Expr') {
    const inputs = (config?.inputs ?? []) as Array<{ name?: string; type?: string }>
    for (const inp of inputs) {
      if (inp?.name && !incoming.has(inp.name)) {
        out.push({ name: inp.name, type: (inp.type ?? 'any') as PinType })
      }
    }
  }
  for (const [name, type] of Object.entries(dataIn)) {
    if (!incoming.has(name)) out.push({ name, type })
  }
  return out
}

// ContainerCanvasApi — ContainerEditorView provide, ContainerFlowNode inject。
// 让画布自定义节点够得到 view 的 draft mutation + 活跃图 edges, 不直接耦合 useVueFlow。
export interface ContainerCanvasApi {
  setPinLiteral: (nodeId: string, pin: string, value: unknown) => void
  edges: ComputedRef<{ from: string; to: string }[]>
}

export const ContainerCanvasApiKey: InjectionKey<ContainerCanvasApi> = Symbol('containerCanvasApi')
