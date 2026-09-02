import { describe, expect, it } from 'vitest'
import { eligibleDiagnosticProfiles, explainAIProfileEligibility } from './aiProfileEligibility'

const profile = (toolCalling: boolean, evaluation: 'unverified' | 'approved' | 'rejected') => ({
  slot: `model-${toolCalling}-${evaluation}`,
  capabilities: { toolCalling },
  evaluation,
})

describe('AI diagnostic profile eligibility', () => {
  it('explains the current two-profile state instead of presenting a dead select', () => {
    const profiles = [profile(false, 'unverified'), profile(false, 'unverified')]
    expect(eligibleDiagnosticProfiles(profiles)).toEqual([])
    expect(explainAIProfileEligibility(profiles)).toBe('tool-calling-required')
  })

  it('allows an unverified tool-calling model to be selected for diagnostics', () => {
    const profiles = [profile(true, 'unverified'), profile(true, 'approved')]
    expect(eligibleDiagnosticProfiles(profiles).map((item) => item.slot)).toEqual([
      'model-true-unverified',
      'model-true-approved',
    ])
    expect(explainAIProfileEligibility(profiles)).toBe('ready')
  })
})
