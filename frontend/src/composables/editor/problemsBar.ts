// frontend/src/composables/editor/problemsBar.ts
import type { ValidationError } from '@/lib/backend'

export interface ProblemsSummary {
  errorCount: number
  warnCount: number
  status: 'fail' | 'warn' | 'pass' // fail=有 error; warn=仅 warning; pass=0 问题
}

export function summarizeProblems(errors: ValidationError[]): ProblemsSummary {
  const errorCount = errors.filter((e) => e.severity === 'error').length
  const warnCount = errors.filter((e) => e.severity === 'warning').length
  const status = errorCount > 0 ? 'fail' : warnCount > 0 ? 'warn' : 'pass'
  return { errorCount, warnCount, status }
}
