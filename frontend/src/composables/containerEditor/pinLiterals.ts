// 算"未连线的 data-in pin"+ 画布内联编辑写回 API。
// NodeInspector (literal section) 与 ContainerFlowNode (画布内联 input) 共用此纯函数。
//
// 连线判定按 target pin 名 (e.to) — 边占不占某 data-in 由 target slot 决定, 不看 source。
// pin 名在单节点内唯一, exec 边落 exec-in pin (名不同) 天然被 dataIn 名取交集忽略。
import type { ComputedRef, InjectionKey } from 'vue'
import type { PinType } from '@/components/containers/nodeRegistry/index'
import { getSpec } from '@/components/containers/nodeRegistry/registry'

export interface LiteralPin {
  name: string
  type: PinType
}

// 有专属 Inspector section 的节点 — 其 input 全由 bespoke section 编辑, 不渲染通用 PinInput
// section, 也不出画布内联。与"值存哪 (config.literal)"正交 (input-editing-unification §3.6):
// 渲染抑制 = 节点身份, 存储 = 判别规则。
//   - WindowTarget / MouseCalibration: 声明式 singleton, 专属捕获/校准 UI。
//   - Subgraph: 1:1 绑定 section 编辑 SubgraphID/Params。
//   - PlayClip: clip 绑定 section 编辑 ClipID + keepRanges。
//   - Switch: SwitchInspector 编辑 value/cases (注: Switch FE/BE schema 另有既有漂移,
//     非本 spec 范围 — 不在这里叠第三套编辑面)。
// (CommentBox 是 IsVisualOnly + 独立组件, 天然不经此路径, 无需列入。)
const BESPOKE_EDITOR_KINDS = new Set([
  'WindowTarget',
  'MouseCalibration',
  'Subgraph',
  'PlayClip',
  'Switch',
])

export function unconnectedDataInPins(
  kind: string,
  dataIn: Record<string, PinType>,
  config: Record<string, unknown> | null | undefined,
  edges: { from: string; to: string }[],
  nodeId: string,
): LiteralPin[] {
  if (BESPOKE_EDITOR_KINDS.has(kind)) return []
  const incoming = new Set<string>()
  for (const e of edges) {
    const [tgt, pin] = (e.to ?? '').split('.')
    if (tgt === nodeId && pin) incoming.add(pin)
  }
  const out: LiteralPin[] = []
  if (getSpec(kind)?.dynamicInputs) {
    // 动态输入声明存 config.Inputs[] (PascalCase Name/Type/Var, 跟后端 ParseDynamicInputDecls 一致)。
    // 绑定项 (Var 非空) 不是 pin, 无字面量框 (值直读变量)。
    const inputs = (config?.Inputs ?? []) as Array<{ Name?: string; Type?: string; Var?: string }>
    for (const inp of inputs) {
      if (inp?.Var) continue
      if (inp?.Name && !incoming.has(inp.Name)) {
        out.push({ name: inp.Name, type: (inp.Type ?? 'any') as PinType })
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
  // 批量写多个 config.literal 键 (CommentBox 内联编辑一次写 Title/Content/Color, 单次 mutation).
  patchNodeLiteral: (nodeId: string, patch: Record<string, unknown>) => void
  edges: ComputedRef<{ from: string; to: string }[]>
}

export const ContainerCanvasApiKey: InjectionKey<ContainerCanvasApi> = Symbol('containerCanvasApi')
