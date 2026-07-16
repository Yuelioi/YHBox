import authoring from '../../../contracts/node/3.1/builtin-authoring.json'
import type {
  FieldProjection,
  NodeProjection,
  TypeProjection,
  YottaNodeAuthoringProjection,
} from '../../../contracts/node/3.1/authoring-projection'

const document = authoring as unknown as YottaNodeAuthoringProjection

if (
  document.format !== 'yotta.node-authoring-projection' ||
  document.version !== '3.1' ||
  !document.projectionDigest.startsWith('sha256:') ||
  !document.body.catalogHash.startsWith('sha256:')
) {
  throw new Error('unsupported built-in node authoring projection')
}

export const builtinNodeProjections = new Map<string, NodeProjection>(
  document.body.nodes.map((projection) => [projection.nodeRef.nodeTypeId, projection]),
)

export const builtinTypeProjections = new Map<string, TypeProjection>(
  document.body.types.map((projection) => [projection.typeRef.typeId, projection]),
)

export const builtinAuthoringGeneration = Object.freeze({
  catalogHash: document.body.catalogHash,
  generatorVersion: document.body.generatorVersion,
  projectionDigest: document.projectionDigest,
})

export type { FieldProjection, NodeProjection, TypeProjection }
