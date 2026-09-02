export interface AIProfileEligibilityInput {
  slot: string
  capabilities: { toolCalling: boolean }
  evaluation: 'unverified' | 'approved' | 'rejected'
}

export type AIProfileEligibilityReason = 'ready' | 'no-profiles' | 'tool-calling-required'

export function eligibleDiagnosticProfiles<T extends AIProfileEligibilityInput>(
  profiles: readonly T[],
): T[] {
  return profiles.filter((profile) => profile.capabilities.toolCalling)
}

export function explainAIProfileEligibility(
  profiles: readonly AIProfileEligibilityInput[],
): AIProfileEligibilityReason {
  if (profiles.length === 0) return 'no-profiles'
  if (!profiles.some((profile) => profile.capabilities.toolCalling)) return 'tool-calling-required'
  return 'ready'
}
