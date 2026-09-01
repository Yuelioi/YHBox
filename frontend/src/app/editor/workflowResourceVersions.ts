import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'

type ImageVariant = NonNullable<WorkflowResource['image']>['variants'][number]

export function applyCapturedImageVersion(
  resource: WorkflowResource,
  captured: ImageVariant,
  mode: 'replace' | 'append',
): WorkflowResource {
  const existing = resource.image?.variants
  if (!existing?.length) throw new Error('image resource has no current version')

  let variants: ImageVariant[]
  if (mode === 'replace') {
    variants = [
      { ...structuredClone(captured), id: existing[0].id },
      ...structuredClone(existing.slice(1)),
    ]
  } else {
    const ids = new Set(existing.map((variant) => variant.id))
    let id = captured.id
    for (let suffix = 2; ids.has(id); suffix++) id = `${captured.id}-${suffix}`
    variants = [...structuredClone(existing), { ...structuredClone(captured), id }]
  }
  variants.sort((left, right) => left.id.localeCompare(right.id))
  return {
    ...structuredClone(resource),
    image: { variants: variants as NonNullable<WorkflowResource['image']>['variants'] },
  }
}
