import { describe, it, expect, beforeEach } from 'vitest'
import { register, __resetForTests } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec } from '@/components/containers/nodeRegistry'
import { bindableFields } from './bindableFields'

// 最小 NodeKindSpec — 只填 bindableFields/pinsFor 用到的结构字段, 其余给空默认.
function makeSpec(partial: Partial<NodeKindSpec> & { kind: string }): NodeKindSpec {
  return {
    group: 'detect',
    labelZh: `node.${partial.kind}.label`,
    description: `node.${partial.kind}.description`,
    example: `node.${partial.kind}.example`,
    visual: { icon: '', bg: '', border: '' },
    execIn: ['In'],
    execOut: [],
    dataIn: {},
    dataOut: {},
    fields: [],
    defaults: {},
    ...partial,
  }
}

describe('bindableFields (Spec C 单一来源)', () => {
  beforeEach(() => __resetForTests())

  it('非纯数据 + exec 出口 Data 字段 (DetectColor 风格) → 全部字段', () => {
    register(
      makeSpec({
        kind: 'DetectColor',
        execOut: ['Found', 'NotFound'],
        dataOut: { Count: 'number', Center: 'point' },
      }),
    )
    expect(bindableFields('DetectColor', null)).toEqual(['Count', 'Center'])
  })

  it('region 风格 (Loop: Body 出口 Index) → [Index]', () => {
    register(makeSpec({ kind: 'Loop', execOut: ['Body', 'Done'], dataOut: { Index: 'number' } }))
    expect(bindableFields('Loop', {})).toEqual(['Index'])
  })

  it('纯数据节点 (isPureData) → 空 (输出是连线源, 不可绑)', () => {
    register(makeSpec({ kind: 'GetVar', isPureData: true, execIn: [], dataOut: { value: 'any' } }))
    expect(bindableFields('GetVar', null)).toEqual([])
  })

  it('未注册 kind → 空', () => {
    expect(bindableFields('Nonexistent', null)).toEqual([])
  })
})
