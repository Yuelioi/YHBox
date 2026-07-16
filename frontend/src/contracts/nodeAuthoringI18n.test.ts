import { describe, expect, it } from 'vitest'
import authoring from '../../../contracts/node/3.1/builtin-authoring.json'
import en from '@/i18n/en'
import zh from '@/i18n/zh'

describe('Node Authoring Projection 3.1 i18n', () => {
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
