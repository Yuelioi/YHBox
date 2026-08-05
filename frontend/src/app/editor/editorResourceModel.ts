import type {
  BlobRef,
  ResourceBinding,
  WorkflowResource,
  YottaWorkflowSource,
} from '../../../../contracts/workflow/current/workflow-source'

export function validBlob(blob: BlobRef): boolean {
  return (
    /^[a-z0-9][a-z0-9!#$&^_.+-]+\/[a-z0-9][a-z0-9!#$&^_.+-]+$/.test(blob.mediaType) &&
    /^sha256:[0-9a-f]{64}$/.test(blob.digest) &&
    Number.isSafeInteger(blob.size) &&
    blob.size >= 0
  )
}

export function normalizeTextSet(values: readonly string[]): string[] {
  const byKey = new Map<string, string>()
  for (const raw of values) {
    const value = raw.trim()
    if (value && !byKey.has(value.toLocaleLowerCase())) {
      byKey.set(value.toLocaleLowerCase(), value)
    }
  }
  return [...byKey.values()].sort()
}

export function normalizeWorkflowResource(resource: WorkflowResource): WorkflowResource {
  const next = clone(resource)
  next.id = next.id.trim()
  next.name = next.name.trim()
  if (!next.id) throw new Error('workflow resource ID is required')
  if (!next.name) throw new Error('workflow resource name is required')
  next.description = next.description?.trim() || undefined
  next.category = next.category?.trim() || undefined
  const tags = normalizeTextSet(next.tags ?? [])
  next.tags = tags.length ? tags : undefined
  return next
}

export function requireWorkflowResourceBinding(
  source: YottaWorkflowSource,
  binding: ResourceBinding,
): WorkflowResource {
  const resource = source.resources.find((candidate) => candidate.id === binding.resourceId)
  if (!resource) throw new Error(`workflow resource ${binding.resourceId} does not exist`)
  if (resource.kind === 'image') {
    if (
      !binding.variantId ||
      !resource.image?.variants.some((variant) => variant.id === binding.variantId)
    ) {
      throw new Error(`workflow image resource ${binding.resourceId} variant does not exist`)
    }
  } else if (binding.variantId) {
    throw new Error(`workflow resource ${binding.resourceId} does not accept a variant`)
  }
  return resource
}

export function workflowResourceReferenceCount(
  source: YottaWorkflowSource,
  resourceId: string,
): number {
  let count = 0
  for (const graph of source.graphs) {
    for (const owner of [...graph.nodes, ...(graph.calls ?? [])]) {
      for (const binding of Object.values(owner.bindings)) {
        if (binding.kind === 'resource' && binding.resource?.resourceId === resourceId) count++
      }
    }
  }
  return count
}

function clone<T>(value: T): T {
  return structuredClone(value)
}
