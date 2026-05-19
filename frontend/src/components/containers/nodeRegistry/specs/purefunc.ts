// frontend/src/components/containers/nodeRegistry/specs/purefunc.ts
// v4 §6 pure-function nodes (22): arithmetic / compare / logic / string / convert / select.
// Mirrors backend internal/services/container/nodekind/specs/purefunc.go.
//
// CRITICAL — type alignment with backend (v4 spec §6.2):
//   Lt/LtEq/Gt/GtEq/Eq/NotEq/And/Or/Not/Contains/ToBool → result = 'bool'
//   And/Or → data-in = 'bool'; Not → data-in = 'bool'
// (These differ from current pinSpec.ts which uses 'any' for those — backend is authoritative.)
import { register } from '../registry'
import type { NodeKindSpec, PinType } from '../index'

const purefuncVisualSuffix = { bg: 'bg-cyan-500/15', border: 'border-cyan-500/40' }

function pureFunc(args: {
  kind: string
  labelZh: string
  description?: string
  icon: string
  dataIn: Record<string, PinType>
  result: PinType
  literalDefaults: Record<string, unknown>
}): NodeKindSpec {
  return {
    kind: args.kind,
    group: 'purefunc',
    labelZh: args.labelZh,
    description: args.description ?? '',
    visual: { icon: args.icon, ...purefuncVisualSuffix },
    execIn: [],
    execOut: [],
    dataIn: args.dataIn,
    dataOut: { result: args.result },
    fields: [],
    defaults: { literal: args.literalDefaults },
    isPureData: true,
  }
}

// 算术 (6)
register(pureFunc({ kind: 'Add', labelZh: '加', icon: 'i-tabler-plus', dataIn: { a: 'number', b: 'number' }, result: 'number', literalDefaults: { a: 0, b: 0 } }))
register(pureFunc({ kind: 'Sub', labelZh: '减', icon: 'i-tabler-minus', dataIn: { a: 'number', b: 'number' }, result: 'number', literalDefaults: { a: 0, b: 0 } }))
register(pureFunc({ kind: 'Mul', labelZh: '乘', icon: 'i-tabler-x', dataIn: { a: 'number', b: 'number' }, result: 'number', literalDefaults: { a: 0, b: 0 } }))
register(pureFunc({ kind: 'Div', labelZh: '除', icon: 'i-tabler-divide', dataIn: { a: 'number', b: 'number' }, result: 'number', literalDefaults: { a: 0, b: 0 } }))
register(pureFunc({ kind: 'Mod', labelZh: '取余', icon: 'i-tabler-percentage', dataIn: { a: 'number', b: 'number' }, result: 'number', literalDefaults: { a: 0, b: 0 } }))
register(pureFunc({ kind: 'Neg', labelZh: '取负', icon: 'i-tabler-arrow-down-right', dataIn: { x: 'number' }, result: 'number', literalDefaults: { x: 0 } }))

// 比较 (6) — backend says bool, not any
register(pureFunc({ kind: 'Lt', labelZh: '<', icon: 'i-tabler-math-lower', dataIn: { a: 'number', b: 'number' }, result: 'bool', literalDefaults: { a: 0, b: 0 } }))
register(pureFunc({ kind: 'LtEq', labelZh: '≤', icon: 'i-tabler-math-equal-lower', dataIn: { a: 'number', b: 'number' }, result: 'bool', literalDefaults: { a: 0, b: 0 } }))
register(pureFunc({ kind: 'Gt', labelZh: '>', icon: 'i-tabler-math-greater', dataIn: { a: 'number', b: 'number' }, result: 'bool', literalDefaults: { a: 0, b: 0 } }))
register(pureFunc({ kind: 'GtEq', labelZh: '≥', icon: 'i-tabler-math-equal-greater', dataIn: { a: 'number', b: 'number' }, result: 'bool', literalDefaults: { a: 0, b: 0 } }))
register(pureFunc({ kind: 'Eq', labelZh: '==', icon: 'i-tabler-equal', dataIn: { a: 'any', b: 'any' }, result: 'bool', literalDefaults: {} }))
register(pureFunc({ kind: 'NotEq', labelZh: '!=', icon: 'i-tabler-equal-not', dataIn: { a: 'any', b: 'any' }, result: 'bool', literalDefaults: {} }))

// 逻辑 (3) — backend uses bool pins (v4 spec §6.3)
register(pureFunc({ kind: 'And', labelZh: '与', icon: 'i-tabler-ampersand', dataIn: { a: 'bool', b: 'bool' }, result: 'bool', literalDefaults: { a: true, b: true } }))
register(pureFunc({ kind: 'Or', labelZh: '或', icon: 'i-tabler-letter-o', dataIn: { a: 'bool', b: 'bool' }, result: 'bool', literalDefaults: { a: false, b: false } }))
register(pureFunc({ kind: 'Not', labelZh: '非', icon: 'i-tabler-exclamation-mark', dataIn: { x: 'bool' }, result: 'bool', literalDefaults: { x: false } }))

// 字符串 (3)
register(pureFunc({ kind: 'Concat', labelZh: '拼字符串', icon: 'i-tabler-plus-equal', dataIn: { a: 'string', b: 'string' }, result: 'string', literalDefaults: { a: '', b: '' } }))
register(pureFunc({ kind: 'Contains', labelZh: '包含', icon: 'i-tabler-search', dataIn: { haystack: 'string', needle: 'string' }, result: 'bool', literalDefaults: { haystack: '', needle: '' } }))
register(pureFunc({ kind: 'Length', labelZh: '长度', icon: 'i-tabler-ruler', dataIn: { s: 'string' }, result: 'number', literalDefaults: { s: '' } }))

// 转换 (3)
register(pureFunc({ kind: 'ToString', labelZh: '转字符串', icon: 'i-tabler-letter-t', dataIn: { x: 'any' }, result: 'string', literalDefaults: { x: '' } }))
register(pureFunc({ kind: 'ToNumber', labelZh: '转数字', icon: 'i-tabler-numbers', dataIn: { x: 'any' }, result: 'number', literalDefaults: { x: 0 } }))
register(pureFunc({ kind: 'ToBool', labelZh: '转布尔', icon: 'i-tabler-toggle-right', dataIn: { x: 'any' }, result: 'bool', literalDefaults: { x: false } }))

// Select 三元 (1)
register(pureFunc({ kind: 'Select', labelZh: '三元选择', icon: 'i-tabler-arrows-split', dataIn: { cond: 'bool', a: 'any', b: 'any' }, result: 'any', literalDefaults: { cond: true } }))
