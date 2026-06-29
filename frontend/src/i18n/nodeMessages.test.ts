import { describe, expect, it } from 'vitest'
import zh from './zh'
import en from './en'

function resolveKey(messages: unknown, key: string): unknown {
  return key.split('.').reduce<unknown>((cur, part) => {
    if (!cur || typeof cur !== 'object') return undefined
    return (cur as Record<string, unknown>)[part]
  }, messages)
}

describe('node i18n messages', () => {
  const requiredKeys = [
    'node.AndroidStartApp.label',
    'node.AndroidStartApp.description',
    'node.AndroidStartApp.input.Package.label',
    'node.AndroidStartApp.input.Package.hint',
    'node.AndroidStartApp.output.Done.label',
    'node.AndroidStopApp.label',
    'node.AndroidStopApp.description',
    'node.AndroidStopApp.input.Package.label',
    'node.AndroidStopApp.input.Package.hint',
    'node.AndroidStopApp.output.Done.label',
  ]

  it.each([
    ['zh', zh],
    ['en', en],
  ])('%s has Android app lifecycle node keys', (_locale, messages) => {
    for (const key of requiredKeys) {
      const value = resolveKey(messages, key)
      expect(value, key).toEqual(expect.any(String))
      expect(String(value), key).not.toBe('')
    }
  })
})
