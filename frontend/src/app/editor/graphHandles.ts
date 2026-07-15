import type { Edge } from '../../../../contracts/workflow/3.1/workflow-source'

export type HandleChannel = Edge['channel']
export type HandleDirection = 'input' | 'output'

export interface ParsedHandle {
  channel: HandleChannel
  direction: HandleDirection
  portId: string
}

export function graphHandle(
  channel: HandleChannel,
  direction: HandleDirection,
  portId: string,
): string {
  return `${channel}:${direction}:${encodeURIComponent(portId)}`
}

export function parseGraphHandle(value: string | null | undefined): ParsedHandle | null {
  if (!value) return null
  const [channel, direction, encodedPort, ...rest] = value.split(':')
  if (
    rest.length !== 0 ||
    (channel !== 'data' && channel !== 'exec' && channel !== 'error') ||
    (direction !== 'input' && direction !== 'output') ||
    !encodedPort
  ) {
    return null
  }
  return { channel, direction, portId: decodeURIComponent(encodedPort) }
}
