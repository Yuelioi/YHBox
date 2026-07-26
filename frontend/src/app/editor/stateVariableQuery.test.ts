import { describe, expect, it } from 'vitest'
import type { Variable } from '../../../../contracts/workflow/current/workflow-source'
import { filterStateVariables, STATE_VARIABLE_PAGE_SIZE } from './stateVariableQuery'

describe('state variable query', () => {
  it('keeps 1000 typed states searchable while the panel renders bounded pages', () => {
    const variables = Array.from(
      { length: 1_000 },
      (_, index): Variable => ({
        name: `state_${String(index).padStart(4, '0')}`,
        type: {
          kind: 'ref',
          ref: {
            typeId: index % 2 === 0 ? 'type/integer/v1' : 'type/string/v1',
            semanticDigest: 'sha256:test',
          },
        },
        default: index % 2 === 0 ? 0 : '',
      }),
    )

    expect(STATE_VARIABLE_PAGE_SIZE).toBe(100)
    expect(filterStateVariables(variables, 'state_09', () => 'Integer')).toHaveLength(100)
    expect(
      filterStateVariables(variables, 'string', (variable) =>
        variable.type.kind === 'ref' && variable.type.ref.typeId.includes('string')
          ? 'String'
          : 'Integer',
      ),
    ).toHaveLength(500)
    expect(filterStateVariables(variables, 'STATE_0999', () => 'String')[0]?.name).toBe(
      'state_0999',
    )
  })
})
