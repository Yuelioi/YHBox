// frontend/src/components/containers/nodeRegistry/specs/expr.ts
// Expr (1) — dynamic data-in pins derived from config.inputs[].
// Mirrors backend internal/services/container/nodekind/specs/expr.go.
import { register } from '../registry'
import type { PinType } from '../index'

register({
  kind: 'Expr',
  group: 'variables',
  labelZh: '表达式',
  description: '',
  visual: { icon: 'i-tabler-math', bg: 'bg-amber-500/15', border: 'border-amber-500/40' },
  execIn: [],
  execOut: [],
  dataIn: {},
  dataInDynamicFn: (cfg) => {
    const out: Record<string, PinType> = {}
    const inputs = Array.isArray(cfg?.inputs) ? (cfg!.inputs as unknown[]) : []
    for (const raw of inputs) {
      const obj = raw as Record<string, unknown> | null
      const name = typeof obj?.name === 'string' ? obj.name : ''
      const type = typeof obj?.type === 'string' ? (obj.type as PinType) : ('' as PinType | '')
      if (!name || !type) continue
      out[name] = type as PinType
    }
    return out
  },
  dataOut: { value: 'any' },
  fields: [
    {
      key: 'expr',
      label: '表达式',
      type: 'text',
      placeholder: 'i + 1',
    },
    {
      key: 'outType',
      label: '输出类型',
      type: 'select',
      options: [
        { value: 'auto', label: 'auto (默认, 推断)' },
        { value: 'number', label: 'number' },
        { value: 'bool', label: 'bool' },
        { value: 'string', label: 'string' },
        { value: 'point', label: 'point' },
        { value: 'any', label: 'any' },
      ],
    },
  ],
  defaults: { expr: '', outType: 'auto', inputs: [] },
  isPureData: true,
})
