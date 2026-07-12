import { describe, expect, it } from 'vitest'
import type { NodeKindSpec, PinType } from './nodeRegistry'
import { filterInlineNodeCandidates, type InlineNodeCandidateContext } from './inlineNodeCandidates'

function spec(partial: Partial<NodeKindSpec> & Pick<NodeKindSpec, 'kind'>): NodeKindSpec {
  return {
    kind: partial.kind,
    group: partial.group ?? 'io',
    labelZh: '',
    description: '',
    example: '',
    visual: { icon: '', bg: '', border: '' },
    execIn: partial.execIn ?? [],
    execOut: partial.execOut ?? [],
    dataIn: partial.dataIn ?? {},
    dataOut: partial.dataOut ?? {},
    fields: [],
    defaults: {},
    isPureData: partial.isPureData,
    isVisualOnly: partial.isVisualOnly,
    excludeFromPalette: partial.excludeFromPalette,
  }
}

describe('filterInlineNodeCandidates', () => {
  it('keeps palette-eligible visual nodes for plain canvas menus', () => {
    const specs = [
      spec({ kind: 'CommentBox', isVisualOnly: true }),
      spec({ kind: 'SubgraphInput', excludeFromPalette: true }),
      spec({ kind: 'Log', execIn: ['In'], execOut: ['Done'] }),
    ]

    const got = filterInlineNodeCandidates(specs).map((s) => s.kind)

    expect(got).toEqual(['CommentBox', 'Log'])
  })

  it('filters output data pins to nodes that can consume that pin type', () => {
    const specs = [
      spec({
        kind: 'ReadTextFile',
        execIn: ['In'],
        execOut: ['Done'],
        dataOut: { Text: 'string' },
      }),
      spec({ kind: 'Log', execIn: ['In'], execOut: ['Done'], dataIn: { Message: 'any' } }),
      spec({ kind: 'Fetch', execIn: ['In'], execOut: ['Done'], dataIn: { URL: 'string' } }),
      spec({ kind: 'ClickAt', execIn: ['In'], execOut: ['Done'], dataIn: { Point: 'point' } }),
    ]

    const got = filterInlineNodeCandidates(specs, { side: 'output', pinType: 'string' }).map(
      (s) => s.kind,
    )

    expect(got).toEqual(['Log', 'Fetch'])
  })

  it('filters input data pins to nodes that can produce that pin type', () => {
    const specs = [
      spec({
        kind: 'ReadTextFile',
        execIn: ['In'],
        execOut: ['Done'],
        dataOut: { Text: 'string' },
      }),
      spec({ kind: 'Fetch', execIn: ['In'], execOut: ['Done'], dataOut: { Body: 'string' } }),
      spec({ kind: 'Now', isPureData: true, dataOut: { Value: 'number' } }),
      spec({
        kind: 'ClickTemplate',
        execIn: ['In'],
        execOut: ['Found'],
        dataOut: { Point: 'point' },
      }),
    ]

    const got = filterInlineNodeCandidates(specs, { side: 'input', pinType: 'string' }).map(
      (s) => s.kind,
    )

    expect(got).toEqual(['ReadTextFile', 'Fetch'])
  })

  it.each([
    ['number'],
    ['bool'],
    ['string'],
    ['point'],
    ['any'],
    ['list'],
    ['file'],
  ] satisfies Array<[PinType]>)('filters output data pins for %s consumers', (pinType) => {
    const specs = [
      spec({
        kind: 'ExactConsumer',
        execIn: ['In'],
        execOut: ['Done'],
        dataIn: { Value: pinType },
      }),
      spec({ kind: 'AnyConsumer', execIn: ['In'], execOut: ['Done'], dataIn: { Value: 'any' } }),
      spec({
        kind: 'WrongConsumer',
        execIn: ['In'],
        execOut: ['Done'],
        dataIn: { Value: incompatibleTypeFor(pinType) },
      }),
      spec({
        kind: 'ProducerOnly',
        execIn: ['In'],
        execOut: ['Done'],
        dataOut: { Value: pinType },
      }),
      spec({
        kind: 'Marker',
        execIn: ['In'],
        execOut: ['Done'],
        dataIn: { Value: pinType },
        excludeFromPalette: true,
      }),
    ]

    const got = filterInlineNodeCandidates(specs, { side: 'output', pinType }).map((s) => s.kind)

    expect(got).toEqual(
      pinType === 'any'
        ? ['ExactConsumer', 'AnyConsumer', 'WrongConsumer']
        : ['ExactConsumer', 'AnyConsumer'],
    )
  })

  it('does not include warning-only coercion targets in output data candidates', () => {
    const specs = [
      spec({
        kind: 'NumberConsumer',
        execIn: ['In'],
        execOut: ['Done'],
        dataIn: { Value: 'number' },
      }),
      spec({ kind: 'AnyConsumer', execIn: ['In'], execOut: ['Done'], dataIn: { Value: 'any' } }),
      spec({
        kind: 'StringConsumer',
        execIn: ['In'],
        execOut: ['Done'],
        dataIn: { Value: 'string' },
      }),
      spec({ kind: 'BoolConsumer', execIn: ['In'], execOut: ['Done'], dataIn: { Value: 'bool' } }),
    ]

    const got = filterInlineNodeCandidates(specs, { side: 'output', pinType: 'number' }).map(
      (s) => s.kind,
    )

    expect(got).toEqual(['NumberConsumer', 'AnyConsumer'])
  })

  it.each([
    ['number'],
    ['bool'],
    ['string'],
    ['point'],
    ['any'],
    ['list'],
    ['file'],
  ] satisfies Array<[PinType]>)('filters input data pins for %s producers', (pinType) => {
    const specs = [
      spec({
        kind: 'ExactProducer',
        execIn: ['In'],
        execOut: ['Done'],
        dataOut: { Value: pinType },
      }),
      spec({ kind: 'AnyProducer', execIn: ['In'], execOut: ['Done'], dataOut: { Value: 'any' } }),
      spec({
        kind: 'WrongProducer',
        execIn: ['In'],
        execOut: ['Done'],
        dataOut: { Value: incompatibleTypeFor(pinType) },
      }),
      spec({ kind: 'ConsumerOnly', execIn: ['In'], execOut: ['Done'], dataIn: { Value: pinType } }),
      spec({
        kind: 'Marker',
        execIn: ['In'],
        execOut: ['Done'],
        dataOut: { Value: pinType },
        excludeFromPalette: true,
      }),
    ]

    const got = filterInlineNodeCandidates(specs, { side: 'input', pinType }).map((s) => s.kind)

    expect(got).toEqual(
      pinType === 'any'
        ? ['ExactProducer', 'AnyProducer', 'WrongProducer']
        : ['ExactProducer', 'AnyProducer'],
    )
  })

  it('does not include warning-only coercion sources in input data candidates', () => {
    const specs = [
      spec({ kind: 'BoolProducer', execIn: ['In'], execOut: ['Done'], dataOut: { Value: 'bool' } }),
      spec({ kind: 'AnyProducer', execIn: ['In'], execOut: ['Done'], dataOut: { Value: 'any' } }),
      spec({
        kind: 'NumberProducer',
        execIn: ['In'],
        execOut: ['Done'],
        dataOut: { Value: 'number' },
      }),
      spec({
        kind: 'StringProducer',
        execIn: ['In'],
        execOut: ['Done'],
        dataOut: { Value: 'string' },
      }),
    ]

    const got = filterInlineNodeCandidates(specs, { side: 'input', pinType: 'bool' }).map(
      (s) => s.kind,
    )

    expect(got).toEqual(['BoolProducer', 'AnyProducer'])
  })

  it('filters exec output pins to executable nodes only', () => {
    const specs = [
      spec({ kind: 'Log', execIn: ['In'], execOut: ['Done'], dataIn: { Message: 'any' } }),
      spec({
        kind: 'ParseJSON',
        isPureData: true,
        dataIn: { Text: 'string' },
        dataOut: { JSON: 'any' },
      }),
      spec({ kind: 'MakePoint', isPureData: true, dataOut: { Result: 'point' } }),
      spec({ kind: 'CommentBox', isVisualOnly: true }),
      spec({ kind: 'SubgraphInput', execOut: ['Out'], excludeFromPalette: true }),
    ]

    const ctx: InlineNodeCandidateContext = { side: 'output', isExec: true }
    const got = filterInlineNodeCandidates(specs, ctx).map((s) => s.kind)

    expect(got).toEqual(['Log'])
  })

  it('filters exec input pins to nodes that can run before it', () => {
    const specs = [
      spec({ kind: 'If', execIn: ['In'], execOut: ['Then', 'Else'] }),
      spec({ kind: 'Start', execOut: ['Out'] }),
      spec({
        kind: 'ParseJSON',
        isPureData: true,
        dataIn: { Text: 'string' },
        dataOut: { JSON: 'any' },
      }),
      spec({ kind: 'CommentBox', isVisualOnly: true }),
    ]

    const got = filterInlineNodeCandidates(specs, { side: 'input', isExec: true }).map(
      (s) => s.kind,
    )

    expect(got).toEqual(['If', 'Start'])
  })
})

function incompatibleTypeFor(pinType: PinType): PinType {
  switch (pinType) {
    case 'number':
      return 'point'
    case 'bool':
      return 'list'
    case 'string':
      return 'file'
    case 'point':
      return 'number'
    case 'any':
      return 'point'
    case 'list':
      return 'bool'
    case 'file':
      return 'string'
  }
}
