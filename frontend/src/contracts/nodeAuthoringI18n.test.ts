import { describe, expect, it } from 'vitest'
import authoring from '../../../contracts/node/current/builtin-authoring'
import en from '@/i18n/en'
import zh from '@/i18n/zh'

describe('current Node Authoring Projection i18n', () => {
  it('resolves every projected title and description key in both locales', () => {
    const keys = collectMessageKeys(authoring)
    expect(keys.size).toBeGreaterThan(100)
    for (const key of keys) {
      expect(resolveMessage(zh, key), `zh:${key}`).toBeTypeOf('string')
      expect(resolveMessage(en, key), `en:${key}`).toBeTypeOf('string')
    }
  })

  it('contains no legacy registry Kind blocks', () => {
    expect(Object.keys(zh.node).filter((key) => /^[A-Z]/.test(key))).toEqual([])
    expect(Object.keys(en.node).filter((key) => /^[A-Z]/.test(key))).toEqual([])
  })

  it('provides localized fallback labels for every built-in port and signal without a title key', () => {
    const ids = new Set<string>()
    for (const node of authoring.body.nodes) {
      for (const port of [...node.dataInputs, ...node.dataOutputs]) {
        if (!('titleKey' in port) || !port.titleKey) ids.add(port.id)
      }
      for (const signal of node.signals) ids.add(signal.id)
    }

    expect(ids.size).toBeGreaterThan(50)
    for (const id of ids) {
      const key = `workflow.node.port.${id}`
      expect(resolveMessage(zh, key), `zh:${key}`).toBeTypeOf('string')
      expect(resolveMessage(en, key), `en:${key}`).toBeTypeOf('string')
    }
  })
})

function collectMessageKeys(value: unknown, result = new Set<string>()): Set<string> {
  if (Array.isArray(value)) {
    for (const item of value) collectMessageKeys(item, result)
    return result
  }
  if (!value || typeof value !== 'object') return result
  for (const [key, nested] of Object.entries(value)) {
    if ((key === 'titleKey' || key === 'descriptionKey') && typeof nested === 'string') {
      result.add(nested)
    } else {
      collectMessageKeys(nested, result)
    }
  }
  return result
}

function resolveMessage(messages: unknown, key: string): unknown {
  let current = messages
  for (const segment of key.split('.')) {
    if (!current || typeof current !== 'object') return undefined
    current = (current as Record<string, unknown>)[segment]
  }
  return current
}
