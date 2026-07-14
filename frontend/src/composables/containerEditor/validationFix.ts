import type { ValidationError } from '@/lib/backend'
import { safeCoerceForFix } from '@/components/containers/inline/coerceLiteral'

export type LiteralValidationFix =
  | { action: 'remove'; pin: string }
  | { action: 'replace'; pin: string; value: unknown }

export function literalValidationFix(
  issue: ValidationError,
  currentValue: unknown = issue.params?.value,
): LiteralValidationFix | null {
  const pin = typeof issue.params?.pin === 'string' ? issue.params.pin : ''
  if (!pin) return null

  if (issue.code === 'UNKNOWN_LITERAL_PIN') {
    return { action: 'remove', pin }
  }
  if (issue.code !== 'LITERAL_TYPE_MISMATCH') return null

  const expected = typeof issue.params?.expected === 'string' ? issue.params.expected : ''
  const value = safeCoerceForFix(currentValue, expected)
  return value === undefined ? null : { action: 'replace', pin, value }
}
