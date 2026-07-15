import authoring from '../../../contracts/node/3.1/builtin-authoring.json'
import type {
  FieldProjection,
  NodeProjection,
  TypeProjection,
  YottaNodeAuthoringProjection31,
} from '../../../contracts/node/3.1/authoring-projection'

const document31 = authoring as unknown as YottaNodeAuthoringProjection31

if (
  document31.format !== 'yotta.node-authoring-projection' ||
  document31.version !== '3.1' ||
  !document31.projectionDigest.startsWith('sha256:') ||
  !document31.body.catalogHash.startsWith('sha256:')
) {
  throw new Error('unsupported built-in node authoring projection')
}

export const builtinNodeProjections31 = new Map<string, NodeProjection>(
  document31.body.nodes.map((projection) => [projection.nodeRef.nodeTypeId, projection]),
)

export const builtinTypeProjections31 = new Map<string, TypeProjection>(
  document31.body.types.map((projection) => [projection.typeRef.typeId, projection]),
)

export const builtinAuthoringGeneration31 = Object.freeze({
  catalogHash: document31.body.catalogHash,
  generatorVersion: document31.body.generatorVersion,
  projectionDigest: document31.projectionDigest,
})

export type {
  FieldProjection as FieldProjection31,
  NodeProjection as NodeProjection31,
  TypeProjection as TypeProjection31,
}
