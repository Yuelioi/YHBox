import { describe, expect, it } from 'vitest'
import authoringDocument from '../../../../contracts/node/current/builtin-authoring'
import type { YottaNodeAuthoringProjection } from '../../../../contracts/node/current/authoring-projection'
import { buildStateTypeChoices } from './stateVariableTypes'

const authoring = authoringDocument as unknown as YottaNodeAuthoringProjection

describe('state variable type choices', () => {
  it('offers every concrete durable Catalog type with a valid initial value', () => {
    const choices = buildStateTypeChoices(authoring.body.types)
    const ids = new Set(choices.map((choice) => choice.id))
    for (const suffix of [
      '/core/json/v1',
      '/filesystem/metadata/v1',
      '/geometry/region/v1',
      '/geometry/point/v1',
      '/vision/color-blob/v1',
      '/vision/color-range/v1',
      '/vision/qr-code/v1',
      '/vision/template-match/v1',
    ]) {
      expect(
        [...ids].some((id) => id.endsWith(suffix)),
        suffix,
      ).toBe(true)
    }
    expect(choices.every((choice) => choice.defaultValue !== undefined)).toBe(true)
  })

  it('offers a key chord list compatible with Press Keys instead of only one key code', () => {
    const choices = buildStateTypeChoices(authoring.body.types)
    expect(choices.find((choice) => choice.id === 'key-chord')).toMatchObject({
      expression: {
        kind: 'list',
        element: {
          kind: 'ref',
          ref: { typeId: 'https://schemas.yotta.dev/types/automation/key-code/v1' },
        },
      },
      defaultValue: [],
      editorAdapter: 'key-chord',
    })
  })
})
