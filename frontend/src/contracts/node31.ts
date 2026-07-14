import presentation from '../../../contracts/node/3.1/builtin-presentation.json'
import type {
  DataInputPort,
  DataOutputPort,
  TypeExpression,
  YottaNodeContract31,
} from '../../../contracts/node/3.1/node-contract'

interface PresentationDocument31 {
  format: 'yotta.node-presentation'
  version: '3.1'
  presentationDigest: string
  body: {
    generatorVersion: string
    types: unknown[]
    nodes: YottaNodeContract31[]
  }
}

export interface DataPortProjection31 {
  id: string
  direction: 'input' | 'output'
  type: TypeExpression
  typeLabel: string
  bindingHint: 'required' | 'optional' | 'default-available' | 'output'
}

export interface NodeProjection31 {
  nodeRef: YottaNodeContract31['nodeRef']
  titleKey?: string
  descriptionKey?: string
  category?: string
  execution: YottaNodeContract31['semantic']['execution']
  dataInputs: DataPortProjection31[]
  dataOutputs: DataPortProjection31[]
  execInputs: string[]
  execOutputs: string[]
  errorOutputs: string[]
  statusOutputs: string[]
  configSchemaRoot: string
}

const document31 = presentation as unknown as PresentationDocument31

if (document31.format !== 'yotta.node-presentation' || document31.version !== '3.1') {
  throw new Error('unsupported built-in node presentation contract')
}

export const builtinNodeContracts31 = new Map(
  document31.body.nodes.map((contract) => [contract.nodeRef.nodeTypeId, contract]),
)

export const builtinNodeProjections31 = new Map(
  [...builtinNodeContracts31].map(([nodeTypeID, contract]) => [
    nodeTypeID,
    projectNodeContract31(contract),
  ]),
)

export function projectNodeContract31(contract: YottaNodeContract31): NodeProjection31 {
  const ports = contract.semantic.ports
  return {
    nodeRef: contract.nodeRef,
    titleKey: contract.authoring.titleKey,
    descriptionKey: contract.authoring.descriptionKey,
    category: contract.authoring.category,
    execution: contract.semantic.execution,
    dataInputs: ports.dataInputs.map(projectInput),
    dataOutputs: ports.dataOutputs.map(projectOutput),
    execInputs: ports.execInputs.map((port) => port.id),
    execOutputs: ports.execOutputs.map((port) => port.id),
    errorOutputs: ports.errorOutputs.map((port) => port.id),
    statusOutputs: ports.statusOutputs.map((port) => port.id),
    configSchemaRoot: contract.semantic.configSchemaRoot,
  }
}

export function typeExpressionLabel31(expression: TypeExpression): string {
  switch (expression.kind) {
    case 'ref':
      return expression.ref.typeId
    case 'list':
      return `list<${typeExpressionLabel31(expression.element)}>`
    case 'union':
      return expression.members.map(typeExpressionLabel31).join(' | ')
    case 'variable': {
      const constraints = expression.constraints?.length
        ? `: ${expression.constraints.join(' & ')}`
        : ''
      return `$${expression.variable}${constraints}`
    }
  }
}

function projectInput(port: DataInputPort): DataPortProjection31 {
  return {
    id: port.id,
    direction: 'input',
    type: port.type,
    typeLabel: typeExpressionLabel31(port.type),
    bindingHint:
      port.default !== undefined ? 'default-available' : port.required ? 'required' : 'optional',
  }
}

function projectOutput(port: DataOutputPort): DataPortProjection31 {
  return {
    id: port.id,
    direction: 'output',
    type: port.type,
    typeLabel: typeExpressionLabel31(port.type),
    bindingHint: 'output',
  }
}
