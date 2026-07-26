import type {
  BlobRef,
  ResourceBinding,
  WorkflowResource,
} from '../../../../contracts/workflow/current/workflow-source'

export type WorkspaceResourceKind = 'macro' | 'clip' | 'template'
export type WorkspaceResourceScope = 'workflow' | 'library'

export interface ResourceLocateRequest {
  requestId: number
  kind: WorkspaceResourceKind
  scope: WorkspaceResourceScope
  id: string
  variantId?: string
}

export type ResourceLocation = Omit<ResourceLocateRequest, 'requestId'>

export interface ResolvedWorkflowResourceBinding {
  resource: WorkflowResource
  blob: BlobRef
  resolution?: [number, number]
}

export function resolveWorkflowResourceBinding(
  resources: readonly WorkflowResource[],
  binding: ResourceBinding | undefined,
): ResolvedWorkflowResourceBinding | undefined {
  if (!binding) return undefined
  const resource = resources.find((candidate) => candidate.id === binding.resourceId)
  if (!resource) return undefined
  if (resource.kind === 'image') {
    const variant = resource.image?.variants.find((candidate) => candidate.id === binding.variantId)
    return variant ? { resource, blob: variant.blob, resolution: variant.resolution } : undefined
  }
  if (binding.variantId) return undefined
  const blob = resource.kind === 'macro' ? resource.macro?.blob : resource.inputClip?.blob
  return blob ? { resource, blob } : undefined
}

export function workspaceResourceKind(resource: WorkflowResource): WorkspaceResourceKind {
  if (resource.kind === 'image') return 'template'
  return resource.kind === 'input-clip' ? 'clip' : 'macro'
}
