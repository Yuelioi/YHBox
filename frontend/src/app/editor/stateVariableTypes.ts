import type {
  TypeExpression,
  TypeProjection,
} from '../../../../contracts/node/3.1/authoring-projection'

const KEY_CODE_TYPE_ID = 'https://schemas.yotta.dev/types/automation/key-code/v1'

export interface StateTypeChoice {
  id: string
  expression: TypeExpression
  projection: TypeProjection
  defaultValue: unknown
  editorAdapter?: 'key-chord'
  titleKey?: string
}

export function buildStateTypeChoices(types: readonly TypeProjection[]): StateTypeChoice[] {
  const choices = types.flatMap((projection): StateTypeChoice[] => {
    if (!projection.traits.includes('durable')) return []
    const defaultValue = defaultStateValue(projection)
    if (defaultValue === undefined) return []
    return [
      {
        id: projection.typeRef.typeId,
        expression: { kind: 'ref', ref: { ...projection.typeRef } },
        projection,
        defaultValue,
      },
    ]
  })
  const keyCode = types.find((type) => type.typeRef.typeId === KEY_CODE_TYPE_ID)
  if (keyCode) {
    choices.push({
      id: 'key-chord',
      expression: {
        kind: 'list',
        element: { kind: 'ref', ref: { ...keyCode.typeRef } },
      },
      projection: keyCode,
      defaultValue: [],
      editorAdapter: 'key-chord',
      titleKey: 'workflow.state_panel.key_chord_type',
    })
  }
  return choices
}

export function defaultStateValue(type: TypeProjection): unknown {
  if (type.examples.length) return structuredClone(type.examples[0])
  switch (type.control) {
    case 'text':
      return ''
    case 'number':
    case 'integer':
      return 0
    case 'toggle':
      return false
    case 'select':
      return type.constraints.enum[0]
    case 'list':
      return []
    case 'json':
      return null
    case 'object':
      return undefined
  }
}

export function stateTypeChoiceForExpression(
  choices: readonly StateTypeChoice[],
  expression: TypeExpression,
): StateTypeChoice | undefined {
  return choices.find((choice) => sameTypeExpression(choice.expression, expression))
}

function sameTypeExpression(left: TypeExpression, right: TypeExpression): boolean {
  if (left.kind !== right.kind) return false
  if (left.kind === 'ref' && right.kind === 'ref') {
    return (
      left.ref.typeId === right.ref.typeId && left.ref.semanticDigest === right.ref.semanticDigest
    )
  }
  if (left.kind === 'list' && right.kind === 'list') {
    return sameTypeExpression(left.element, right.element)
  }
  return false
}
