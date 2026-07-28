import { describe, expect, it } from 'vitest'
import { projectionLabel, projectionLabelTitle } from './projectionLabels'

const messages: Record<string, string> = {
  'node.example.input.value.title': '专属名称',
  'workflow.node.port.value': '通用名称',
}
const t = (key: string) => messages[key] ?? key
const te = (key: string) => key in messages

describe('projection labels', () => {
  it('prefers a node-specific title and then a common semantic label', () => {
    expect(
      projectionLabel({ id: 'value', titleKey: 'node.example.input.value.title' }, t, te),
    ).toBe('专属名称')
    expect(projectionLabel({ id: 'value' }, t, te)).toBe('通用名称')
  })

  it('keeps unknown extension identifiers stable and exposes IDs in localized tooltips', () => {
    expect(projectionLabel({ id: 'plugin-value' }, t, te)).toBe('plugin-value')
    expect(projectionLabelTitle('通用名称', 'value')).toBe('通用名称 · value')
    expect(projectionLabelTitle('plugin-value', 'plugin-value')).toBe('plugin-value')
  })
})
