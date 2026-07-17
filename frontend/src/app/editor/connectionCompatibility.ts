import type {
  NodeProjection,
  ResourceLeaseBinding,
  TypeExpression,
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
}

export interface CompatibleCandidatePort {
  handle: ParsedHandle
  exact: boolean
}

export function projectedConnectionCompatibility(
  sourceProjection: NodeProjection,
  source: ParsedHandle,
  targetProjection: NodeProjection,
  target: ParsedHandle,
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
    if (!assignable(output.type.expression, input.type.expression)) {
      return invalid('type', 'data edge is not assignable')
    }
    if (output.carrier !== input.carrier) {
      return invalid('carrier', 'data edge carrier class differs')
    }
    if (!resourceLeaseAssignable(output.resourceLease, input.resourceLease)) {
      return invalid('resource-lease', 'data edge resource lease differs')
    }
    return { valid: true }
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
): CompatibleCandidatePort[] {
  if (candidate.instruction.kind === 'run-root') return []
  const candidateDirection = anchor.direction === 'output' ? 'input' : 'output'
  return projectionHandles(candidate, candidateDirection)
    .map((handle) => {
      const compatibility =
        anchor.direction === 'output'
          ? projectedConnectionCompatibility(anchorProjection, anchor, candidate, handle)
          : projectedConnectionCompatibility(candidate, handle, anchorProjection, anchor)
      if (!compatibility.valid) return null
      return { handle, exact: connectionIsExact(anchorProjection, anchor, candidate, handle) }
    })
    .filter((port): port is CompatibleCandidatePort => port !== null)
    .sort(
      (left, right) =>
        Number(right.exact) - Number(left.exact) ||
        left.handle.portId.localeCompare(right.handle.portId),
    )
}

export function assignable(output: TypeExpression, input: TypeExpression): boolean {
  if (output.kind === 'variable' || input.kind === 'variable') return false
  if (output.kind === 'union') return output.members.every((member) => assignable(member, input))
  if (input.kind === 'union') return input.members.some((member) => assignable(output, member))
  if (output.kind !== input.kind) return false
  if (output.kind === 'ref' && input.kind === 'ref') {
    return (
      output.ref.typeId === input.ref.typeId &&
      output.ref.semanticDigest === input.ref.semanticDigest
    )
  }
  if (output.kind === 'list' && input.kind === 'list')
    return assignable(output.element, input.element)
  return false
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
