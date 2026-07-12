import { describe, it, expect } from 'vitest'
import { summarizeProblems } from './problemsBar'
import type { ValidationError } from '@/lib/backend'

const err = (severity: 'error' | 'warning'): ValidationError =>
  ({ severity, code: 'X', graphPath: [], params: {} }) as unknown as ValidationError

describe('summarizeProblems', () => {
  it('empty -> pass', () => {
    expect(summarizeProblems([])).toEqual({ errorCount: 0, warnCount: 0, status: 'pass' })
  })
  it('error present -> fail', () => {
    expect(summarizeProblems([err('error'), err('warning')])).toEqual({
      errorCount: 1,
      warnCount: 1,
      status: 'fail',
    })
  })
  it('only warnings -> warn', () => {
    expect(summarizeProblems([err('warning'), err('warning')])).toEqual({
      errorCount: 0,
      warnCount: 2,
      status: 'warn',
    })
  })
})
