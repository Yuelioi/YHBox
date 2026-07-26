import type { Variable } from '../../../../contracts/workflow/current/workflow-source'

export const STATE_VARIABLE_PAGE_SIZE = 100

export function filterStateVariables(
  variables: Variable[],
  query: string,
  typeLabel: (variable: Variable) => string,
): Variable[] {
  const normalized = query.trim().toLocaleLowerCase()
  if (!normalized) return variables
  return variables.filter((variable) =>
    [variable.name, typeLabel(variable)].some((value) =>
      value.toLocaleLowerCase().includes(normalized),
    ),
  )
}
