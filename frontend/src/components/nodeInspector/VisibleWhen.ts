// frontend/src/components/nodeInspector/VisibleWhen.ts
// VisibleRule 求值 — 支持单字段 (Equals/NotEquals/In) + AND/OR 组合 (递归).
// Claude r4 #5 + DS r2/r3: Switch 节点必撞 AND 条件, Phase 0 mock 必须验证.
import type { VisibleRule } from '@bindings/yhbox/internal/node'

export function evalVisibleRule(rule: VisibleRule | null | undefined, config: Record<string, any>): boolean {
  if (!rule) return true

  // 组合: AND 全 true, OR 至少 1 true
  if (rule.and && rule.and.length > 0) {
    return rule.and.every(r => evalVisibleRule(r, config))
  }
  if (rule.or && rule.or.length > 0) {
    return rule.or.some(r => evalVisibleRule(r, config))
  }

  // 单字段
  if (!rule.field) return true
  const v = config[rule.field]
  if (rule.equals !== undefined && rule.equals !== null) return v === rule.equals
  if (rule.notEquals !== undefined && rule.notEquals !== null) return v !== rule.notEquals
  if (rule.in && rule.in.length > 0) return rule.in.includes(v)
  return true
}
