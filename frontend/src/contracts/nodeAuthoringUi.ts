import type { FieldProjection } from '../../../contracts/node/current/authoring-projection'

export function projectedInitialConfig(
  config: Readonly<Record<string, unknown>> | undefined,
  _fields: readonly FieldProjection[],
): Record<string, unknown> {
  // JSON Schema defaults are annotations. Only an explicit user edit writes config.
  return Object.assign({}, config)
}

export function patchProjectedConfig(
  config: Readonly<Record<string, unknown>> | undefined,
  fieldID: string,
  value: unknown,
): Record<string, unknown> {
  const next: Record<string, unknown> = Object.assign({}, config)
  if (value === undefined) delete next[fieldID]
  else next[fieldID] = value
  return next
}

export function projectedConstraintTokens(field: FieldProjection): string[] {
  const { constraints } = field
  const tokens: string[] = []
  if (constraints.minLength !== undefined) tokens.push(`minLength: ${constraints.minLength}`)
  if (constraints.maxLength !== undefined) tokens.push(`maxLength: ${constraints.maxLength}`)
  if (constraints.minimum !== undefined) tokens.push(`minimum: ${String(constraints.minimum)}`)
  if (constraints.maximum !== undefined) tokens.push(`maximum: ${String(constraints.maximum)}`)
  if (constraints.minItems !== undefined) tokens.push(`minItems: ${constraints.minItems}`)
  if (constraints.maxItems !== undefined) tokens.push(`maxItems: ${constraints.maxItems}`)
  if (constraints.pattern) tokens.push(`pattern: ${constraints.pattern}`)
  return tokens
}
