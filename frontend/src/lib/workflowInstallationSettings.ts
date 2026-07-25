import type { InstalledAutomationTargetProfile } from '@/lib/backend'

export interface TargetProfileRequirement {
  targetKind: string
  adapterKind: string
  profileVersion: string
}

export type ProfileSettingsIssue = 'invalid-json' | 'object-required' | null

export function compatibleAutomationTargets(
  candidates: InstalledAutomationTargetProfile[],
  requirement: TargetProfileRequirement,
): InstalledAutomationTargetProfile[] {
  return candidates.filter(
    (candidate) =>
      candidate.targetKind === requirement.targetKind &&
      candidate.adapterKind === requirement.adapterKind &&
      candidate.profileVersion === requirement.profileVersion,
  )
}

export function profileSettingsIssue(value: string): ProfileSettingsIssue {
  try {
    const parsed = JSON.parse(value)
    return parsed !== null && !Array.isArray(parsed) && typeof parsed === 'object'
      ? null
      : 'object-required'
  } catch {
    return 'invalid-json'
  }
}
