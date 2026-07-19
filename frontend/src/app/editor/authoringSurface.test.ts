import { describe, expect, it } from 'vitest'
import type {
  NodeProjection,
  PortProjection,
} from '../../../../contracts/node/3.1/authoring-projection'
import {
  effectiveTargetSlot,
  projectAuthoringSurface,
  resolvePortAdapter,
} from './authoringSurface'

const pointPort = {
  id: 'point',
  binding: 'default-available',
  carrier: 'durable',
  hasDefault: true,
  default: { x: 0.5, y: 0.5, unit: 'ratio' },
  importance: 'primary',
  inlinePriority: 100,
  order: 2,
  type: {
    expression: { kind: 'ref', ref: { typeId: 'point', semanticDigest: 'sha256:x' } },
    label: 'point',
    control: 'object',
    color: '#0f0',
    typeIds: ['point'],
    representations: [{ kind: 'inline-json', codec: 'yotta.jcs/v1' }],
    lifecycle: 'durable',
    constraints: { enum: [] },
    examples: [],
    editorAdapter: 'point',
  },
} as PortProjection

function projection(inputs: PortProjection[]): NodeProjection {
  return {
    nodeRef: { nodeTypeId: 'node', semanticDigest: 'sha256:x', version: '1.0.0' },
    availability: 'target-required',
    capabilities: [],
    configFields: [],
    dataInputs: inputs,
    dataOutputs: [],
    errors: [],
    execution: {
      cache: 'none',
      cancellation: 'cooperative',
      class: 'effect',
      determinism: 'recorded',
      effects: [],
      evaluation: 'push',
      retry: 'never',
      timeout: 'required',
    },
    hostFeatureRequirements: [],
    instruction: { kind: 'invoke', invoke: {} },
    signals: [],
    stateAccesses: [],
    statusEvents: [],
    tags: [],
  }
}

describe('authoring surface', () => {
  it('resolves adapters by type and excludes connected inputs from the three inline slots', () => {
    expect(resolvePortAdapter(pointPort)).toBe('point')
    const inputs: PortProjection[] = [0, 1, 2, 3].map(
      (index) =>
        ({
          ...pointPort,
          id: `point-${index}`,
          order: index + 1,
          inlinePriority: 100 - index,
          editorAdapter: 'number',
          type: { ...pointPort.type, editorAdapter: 'number', control: 'number' },
        }) as PortProjection,
    )
    const surface = projectAuthoringSurface(projection(inputs), undefined, new Set(['point-0']))
    expect(surface.inlineInputs.map((item) => item.port.id)).toEqual([
      'point-1',
      'point-2',
      'point-3',
    ])
    expect(surface.groups.common).toHaveLength(4)
  })

  it('keeps composite task editors out of the compact canvas node surface', () => {
    const compositeInputs = (['color-range', 'point', 'region'] as const).map(
      (editorAdapter, index) =>
        ({
          ...pointPort,
          id: editorAdapter,
          inlinePriority: 100 - index,
          editorAdapter,
          type: { ...pointPort.type, editorAdapter },
        }) as PortProjection,
    )
    const scalarInput = {
      ...pointPort,
      id: 'threshold',
      inlinePriority: 50,
      editorAdapter: 'number',
      type: { ...pointPort.type, editorAdapter: 'number', control: 'number' },
    } as PortProjection

    const surface = projectAuthoringSurface(projection([...compositeInputs, scalarInput]))

    expect(surface.inlineInputs.map((item) => item.port.id)).toEqual(['threshold'])
    expect(
      surface.groups.common.flatMap((item) => (item.kind === 'input' ? [item.port.id] : [])),
    ).toEqual(['color-range', 'point', 'region', 'threshold'])
  })

  it('puts missing required fields before common and advanced fields', () => {
    const required = {
      ...pointPort,
      id: 'required',
      binding: 'required',
      hasDefault: false,
    } as PortProjection
    const advanced = { ...pointPort, id: 'advanced', importance: 'advanced' } as PortProjection
    const surface = projectAuthoringSurface(projection([required, pointPort, advanced]))
    expect(surface.groups.required.map((item) => item.key)).toEqual(['input:required'])
    expect(surface.groups.common.map((item) => item.key)).toEqual(['input:point'])
    expect(surface.groups.advanced.map((item) => item.key)).toEqual(['input:advanced'])
  })

  it('resolves node target overrides before workflow defaults', () => {
    const source = projection([])
    source.capabilities = [
      {
        capability: { capabilityId: 'capture', semanticDigest: 'sha256:x' },
        consent: 'once',
        credential: 'none',
        operations: ['capture'],
        requirementId: 'target',
        risk: 'sensitive',
        scope: {},
        targetKinds: ['desktop-window'],
        targetSlot: 'target',
        targetSlotConfigKey: 'slot',
      },
    ]
    const node = {
      id: 'node',
      nodeRef: source.nodeRef,
      position: { x: 0, y: 0 },
      config: { slot: 'override' } as Record<string, unknown>,
      bindings: {},
    }
    expect(effectiveTargetSlot(source, node, [{ target: 'target', slot: 'default' }])).toBe(
      'override',
    )
    node.config = {}
    expect(effectiveTargetSlot(source, node, [{ target: 'target', slot: 'default' }])).toBe(
      'default',
    )
  })
})
