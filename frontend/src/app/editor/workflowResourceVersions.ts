import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'

type ImageVariant = NonNullable<WorkflowResource['image']>['variants'][number]

export class WorkflowResourceVersionError extends Error {
  constructor(readonly id: string) {
    super(id)
    this.name = 'WorkflowResourceVersionError'
  }
}

export function applyCapturedImageVersion(
  resource: WorkflowResource,
  captured: ImageVariant,
  mode: 'replace' | 'append',
  replaceVariantId?: string,
): WorkflowResource {
  const existing = resource.image?.variants
  if (!existing?.length)
    throw new WorkflowResourceVersionError('workflow.resource.image_version_missing')

  let variants: ImageVariant[]
  if (mode === 'replace') {
    const targetId = replaceVariantId ?? existing[0].id
    if (!existing.some((variant) => variant.id === targetId)) {
      throw new WorkflowResourceVersionError('workflow.resource.recapture_target_stale')
    }
    variants = existing.map((variant) =>
      variant.id === targetId
        ? { ...clonePortable(captured), id: targetId }
        : clonePortable(variant),
    )
  } else {
    const ids = new Set(existing.map((variant) => variant.id))
    let id = captured.id
    for (let suffix = 2; ids.has(id); suffix++) id = `${captured.id}-${suffix}`
    variants = [...clonePortable(existing), { ...clonePortable(captured), id }]
  }
  variants.sort((left, right) => left.id.localeCompare(right.id))
  return {
    ...clonePortable(resource),
    image: { variants: variants as NonNullable<WorkflowResource['image']>['variants'] },
  }
}

function clonePortable<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}
