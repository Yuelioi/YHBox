import { describe, expect, it } from 'vitest'
import type { ValidationError } from '@/lib/backend'
import { literalValidationFix } from './validationFix'

function issue(code: string, params: Record<string, unknown>): ValidationError {
  return { severity: 'warning', code, graphPath: ['main'], nodeId: 'node-1', params }
}

describe('literalValidationFix', () => {
  it('removes a value that belongs to no declared input', () => {
    expect(literalValidationFix(issue('UNKNOWN_LITERAL_PIN', { pin: 'LegacyValue' }))).toEqual({
      action: 'remove',
      pin: 'LegacyValue',
    })
  })

  it('keeps the existing safe literal coercion', () => {
    expect(
      literalValidationFix(
        issue('LITERAL_TYPE_MISMATCH', { pin: 'Duration', expected: 'number', value: '500' }),
      ),
    ).toEqual({ action: 'replace', pin: 'Duration', value: 500 })
  })

  it('does not offer ambiguous or unrelated fixes', () => {
    expect(
      literalValidationFix(
        issue('LITERAL_TYPE_MISMATCH', { pin: 'Duration', expected: 'number', value: '500ms' }),
      ),
    ).toBeNull()
    expect(literalValidationFix(issue('INVALID_PIN', { pin: 'Missing' }))).toBeNull()
  })
})
