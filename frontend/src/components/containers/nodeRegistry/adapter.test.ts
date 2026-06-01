// adapter.test.ts — 验证 deriveFields schema 透传行为 (D1 结构化输入基建).
import { describe, it, expect } from 'vitest'
import { deriveFields } from './adapter'
import type { InputSpec } from '@bindings/yhbox/internal/node'

describe('deriveFields schema passthrough', () => {
  it('InputSpec.schema 透传到 FieldSchema.schema (object + fields)', () => {
    const fakeInput = {
      name: 'hsv',
      type: 'Geometry', // 非 Exec → 不跳过
      schema: {
        type: 'object',
        fields: [{ key: 'hMin', schema: { type: 'number' } }],
      },
    } as unknown as InputSpec

    const fields = deriveFields('TestNode', [fakeInput])
    expect(fields).toHaveLength(1)
    const f = fields[0]
    expect(f.key).toBe('hsv')
    expect(f.schema).toBeDefined()
    expect(f.schema!.type).toBe('object')
    expect(f.schema!.fields).toHaveLength(1)
    expect(f.schema!.fields![0].key).toBe('hMin')
    expect(f.schema!.fields![0].schema.type).toBe('number')
  })

  it('geometry widget: schema.widget === "geometry"', () => {
    const fakeInput = {
      name: 'region',
      type: 'Geometry',
      schema: {
        type: 'object',
        widget: 'geometry',
      },
    } as unknown as InputSpec

    const fields = deriveFields('TestNode', [fakeInput])
    expect(fields).toHaveLength(1)
    expect(fields[0].schema!.widget).toBe('geometry')
  })

  it('Exec 类型输入跳过, 不出现在 fields', () => {
    const execInput = {
      name: 'in',
      type: 'Exec',
      schema: { type: 'object' },
    } as unknown as InputSpec

    const fields = deriveFields('TestNode', [execInput])
    expect(fields).toHaveLength(0)
  })

  it('schema 缺失时 FieldSchema.schema 为 undefined', () => {
    const plainInput = {
      name: 'value',
      type: 'Number',
    } as unknown as InputSpec

    const fields = deriveFields('TestNode', [plainInput])
    expect(fields).toHaveLength(1)
    expect(fields[0].schema).toBeUndefined()
  })

  it('key-capture widget kind 映射成 key-capture field type (接 KeyCapture 控件)', () => {
    const vkInput = {
      name: 'VK',
      type: 'String',
      widget: { kind: 'key-capture', props: {} },
    } as unknown as InputSpec

    const fields = deriveFields('KeyPress', [vkInput])
    expect(fields).toHaveLength(1)
    expect(fields[0].type).toBe('key-capture')
    expect(fields[0].widgetKind).toBe('key-capture')
  })
})
