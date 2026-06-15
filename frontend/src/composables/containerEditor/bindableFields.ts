import { getSpec } from '@/components/containers/nodeRegistry/registry'
import { pinsFor } from '@/components/containers/pinSpec'

// bindableFields — 某节点可被「输出捕获」绑定到变量的字段名 (Spec C 前端单一来源)。
//
// = 非纯数据节点 (isPureData=false) 所有 exec 出口携带的 Data 字段名。
//   splitOutputs 已把 exec 出口的 Data 字段 flatten 进 s.dataOut, 故 pinsFor().dataOut 即可;
//   非纯数据节点没有独立 (非 exec) data 输出 pin —— 数据全走 exec 出口 Data 字段。
//   region 节点 (Loop/ForEach) 的 Index/Item 由后端声明为 Body 出口 Data 字段 → 自动列入。
// 纯数据节点 (GetVar/Now/Expr/PureFunc/...) 无可绑字段 —— 其输出是连线源, 非捕获 (无 fire/routeResult hook)。
//
// 与后端 node.BindableFields 集合一致 (单一来源, 不各维护白名单)。Inspector「输出」组 +
// useVarMutations (rename/count/usage/cascade) 共用此派生。
export function bindableFields(
  kind: string,
  config?: Record<string, unknown> | null,
): string[] {
  const s = getSpec(kind)
  if (!s || s.isPureData) return []
  return pinsFor(kind, config).dataOut
}
