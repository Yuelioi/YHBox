import type {
  NodeProjection,
  ResourceLeaseBinding,
  TypeExpression,
  TypeProjection,
} from '../../../../contracts/node/3.1/authoring-projection'
import type { ParsedHandle } from './graphHandles'

export type ConnectionIssue =
  | 'direction'
  | 'channel'
  | 'port'
  | 'type'
  | 'carrier'
  | 'resource-lease'
  | 'instruction'

export interface ConnectionCompatibility {
  valid: boolean
  issue?: ConnectionIssue
  message?: string
  match?: 'exact' | 'assignable' | 'generic-bind'
}

export interface CompatibleCandidatePort {
  handle: ParsedHandle
  exact: boolean
  match?: ConnectionCompatibility['match']
}

export function projectedConnectionCompatibility(
  sourceProjection: NodeProjection,
  source: ParsedHandle,
  targetProjection: NodeProjection,
  target: ParsedHandle,
  types?: ReadonlyMap<string, TypeProjection>,
): ConnectionCompatibility {
  if (source.direction !== 'output' || target.direction !== 'input') {
    return invalid('direction', 'connection must run from an output to an input')
  }
  if (source.channel !== target.channel) {
    return invalid('channel', 'connection channels differ')
  }
  if (source.channel === 'data') {
    const output = sourceProjection.dataOutputs.find((port) => port.id === source.portId)
    const input = targetProjection.dataInputs.find((port) => port.id === target.portId)
    if (!output || !input) return invalid('port', 'data edge has invalid ports')
    const match = typeMatch(output.type.expression, input.type.expression, types)
    if (!match) {
      return invalid(
        'type',
        `data edge type ${typeLabel(output.type.expression)} is not assignable to ${typeLabel(input.type.expression)}`,
      )
    }
    if (output.carrier !== input.carrier) {
      return invalid('carrier', 'data edge carrier class differs')
    }
    if (!resourceLeaseAssignable(output.resourceLease, input.resourceLease)) {
      return invalid('resource-lease', 'data edge resource lease differs')
    }
    return { valid: true, match }
  }

  const output = sourceProjection.signals.find(
    (signal) =>
      signal.id === source.portId &&
      signal.direction === 'output' &&
      signal.channel === source.channel,
  )
  const input = targetProjection.signals.find(
    (signal) =>
      signal.id === target.portId &&
      signal.direction === 'input' &&
      signal.channel === target.channel,
  )
  if (!output || !input) return invalid('port', `${source.channel} edge has invalid signal ports`)
  if (!instructionAcceptsSignal(targetProjection, source.channel, target.portId)) {
    return invalid('instruction', `${target.channel} edge is not accepted by the instruction`)
  }
  return { valid: true }
}

export function compatibleCandidatePorts(
  anchorProjection: NodeProjection,
  anchor: ParsedHandle,
  candidate: NodeProjection,
  types?: ReadonlyMap<string, TypeProjection>,
): CompatibleCandidatePort[] {
  if (candidate.instruction.kind === 'run-root') return []
  const candidateDirection = anchor.direction === 'output' ? 'input' : 'output'
  return projectionHandles(candidate, candidateDirection)
    .map<CompatibleCandidatePort | null>((handle) => {
      const compatibility =
        anchor.direction === 'output'
          ? projectedConnectionCompatibility(anchorProjection, anchor, candidate, handle, types)
          : projectedConnectionCompatibility(candidate, handle, anchorProjection, anchor, types)
      if (!compatibility.valid) return null
      return {
        handle,
        exact: connectionIsExact(anchorProjection, anchor, candidate, handle),
        match: compatibility.match,
      }
    })
    .filter((port): port is CompatibleCandidatePort => port !== null)
    .sort(
      (left, right) =>
        Number(right.exact) - Number(left.exact) ||
        left.handle.portId.localeCompare(right.handle.portId),
    )
}

export function assignable(
  output: TypeExpression,
  input: TypeExpression,
  types?: ReadonlyMap<string, TypeProjection>,
): boolean {
  return typeMatch(output, input, types) !== null
}

export function typeMatch(
  output: TypeExpression,
  input: TypeExpression,
  types?: ReadonlyMap<string, TypeProjection>,
): ConnectionCompatibility['match'] | null {
  if (input.kind === 'variable') {
    if (output.kind === 'variable') return null
    if (!input.constraints?.length) return 'generic-bind'
    if (output.kind !== 'ref') return null
    const traits = new Set(types?.get(output.ref.typeId)?.traits ?? [])
    return input.constraints.every((constraint) => traits.has(constraint)) ? 'generic-bind' : null
  }
  if (output.kind === 'variable') {
    if (!output.constraints?.length) return 'generic-bind'
    if (input.kind !== 'ref') return null
    const traits = new Set(types?.get(input.ref.typeId)?.traits ?? [])
    return output.constraints.every((constraint) => traits.has(constraint)) ? 'generic-bind' : null
  }
  if (output.kind === 'union') {
    const matches = output.members.map((member) => typeMatch(member, input, types))
    return matches.every(Boolean) ? weakestMatch(matches) : null
  }
  if (input.kind === 'union') {
    const matches = input.members.map((member) => typeMatch(output, member, types)).filter(Boolean)
    return matches.length ? weakestMatch(matches) : null
  }
  if (output.kind !== input.kind) return null
  if (output.kind === 'ref' && input.kind === 'ref') {
    if (
      output.ref.typeId === input.ref.typeId &&
      output.ref.semanticDigest === input.ref.semanticDigest
    )
      return 'exact'
    const source = types?.get(output.ref.typeId)
    return source?.assignableTo.some(
      (target) =>
        target.typeId === input.ref.typeId && target.semanticDigest === input.ref.semanticDigest,
    )
      ? 'assignable'
      : null
  }
  if (output.kind === 'list' && input.kind === 'list') {
    return JSON.stringify(output.element) === JSON.stringify(input.element) ? 'exact' : null
  }
  return null
}

function weakestMatch(
  matches: Array<ConnectionCompatibility['match'] | null>,
): ConnectionCompatibility['match'] {
  if (matches.includes('generic-bind')) return 'generic-bind'
  if (matches.includes('assignable')) return 'assignable'
  return 'exact'
}

function typeLabel(expression: TypeExpression): string {
  if (expression.kind === 'ref')
    return expression.ref.typeId.split('/').at(-2) ?? expression.ref.typeId
  if (expression.kind === 'variable') return `$${expression.variable}`
  if (expression.kind === 'list') return `List<${typeLabel(expression.element)}>`
  return expression.members.map(typeLabel).join(' | ')
}

function projectionHandles(
  projection: NodeProjection,
  direction: ParsedHandle['direction'],
): ParsedHandle[] {
  const signals = projection.signals
    .filter((signal) => signal.direction === direction)
    .map((signal) => ({ channel: signal.channel, direction, portId: signal.id }))
  const data = (direction === 'input' ? projection.dataInputs : projection.dataOutputs).map(
    (port) => ({ channel: 'data' as const, direction, portId: port.id }),
  )
  return [...signals, ...data]
}

function connectionIsExact(
  anchorProjection: NodeProjection,
  anchor: ParsedHandle,
  candidate: NodeProjection,
  candidateHandle: ParsedHandle,
): boolean {
  if (anchor.channel !== 'data' || candidateHandle.channel !== 'data') return true
  const anchorPort =
    anchor.direction === 'output'
      ? anchorProjection.dataOutputs.find((port) => port.id === anchor.portId)
      : anchorProjection.dataInputs.find((port) => port.id === anchor.portId)
  const candidatePort =
    candidateHandle.direction === 'output'
      ? candidate.dataOutputs.find((port) => port.id === candidateHandle.portId)
      : candidate.dataInputs.find((port) => port.id === candidateHandle.portId)
  if (!anchorPort || !candidatePort) return false
  return (
    JSON.stringify(anchorPort.type.expression) === JSON.stringify(candidatePort.type.expression)
  )
}

function resourceLeaseAssignable(
  source: ResourceLeaseBinding | undefined,
  target: ResourceLeaseBinding | undefined,
): boolean {
  if (!source || !target) return !source && !target
  const allowed = new Set(source.operations)
  return target.operations.every((operation) => allowed.has(operation))
}

function instructionAcceptsSignal(
  projection: NodeProjection,
  channel: 'exec' | 'error',
  inputPort: string,
): boolean {
  const instruction = projection.instruction
  switch (instruction.kind) {
    case 'invoke':
      return true
    case 'run-root':
      return false
    case 'counted-loop': {
      const value = instruction.countedLoop
      return Boolean(
        value &&
        channel === 'exec' &&
        [value.entryInput, value.breakInput, value.continueInput].includes(inputPort),
      )
    }
    case 'for-each': {
      const value = instruction.forEach
      return Boolean(
        value &&
        channel === 'exec' &&
        [value.entryInput, value.breakInput, value.continueInput].includes(inputPort),
      )
    }
    case 'retry': {
      const value = instruction.retry
      return Boolean(
        value &&
        ((channel === 'exec' && inputPort === value.entryInput) ||
          (channel === 'error' && inputPort === value.retryInput)),
      )
    }
  }
}

function invalid(issue: ConnectionIssue, message: string): ConnectionCompatibility {
  return { valid: false, issue, message }
}
