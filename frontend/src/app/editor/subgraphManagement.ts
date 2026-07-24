import type { Graph } from '../../../../contracts/workflow/current/workflow-source'

export interface GraphDefinitionSource {
  entryGraph: string
  graphs: Graph[]
}

export interface GraphCallReference {
  parentGraphId: string
  parentGraphName: string
  callId: string
  callLabel: string
}

export interface GraphDefinitionSummary {
  id: string
  kind: Graph['kind']
  name: string
  shortId: string
  duplicateName: boolean
  callCount: number
  references: GraphCallReference[]
  entryBound: boolean
  dataInputCount: number
  dataOutputCount: number
  execExitCount: number
  errorExitCount: number
  interfaceHealthy: boolean
}

export function projectGraphDefinitions(
  source: GraphDefinitionSource,
  rawQuery = '',
): GraphDefinitionSummary[] {
  const names = new Map<string, number>()
  for (const graph of source.graphs) {
    const key = normalized(graphName(graph))
    names.set(key, (names.get(key) ?? 0) + 1)
  }

  const references = new Map<string, GraphCallReference[]>()
  for (const parent of source.graphs) {
    for (const call of parent.calls ?? []) {
      const locations = references.get(call.graphId) ?? []
      const definition = source.graphs.find((candidate) => candidate.id === call.graphId)
      locations.push({
        parentGraphId: parent.id,
        parentGraphName: graphName(parent),
        callId: call.id,
        callLabel: (call.label ?? '').trim() || graphName(definition),
      })
      references.set(call.graphId, locations)
    }
  }

  const query = normalized(rawQuery)
  return source.graphs
    .map((graph): GraphDefinitionSummary => {
      const exits = graph.exits ?? []
      const locations = references.get(graph.id) ?? []
      const name = graphName(graph)
      return {
        id: graph.id,
        kind: graph.kind,
        name,
        shortId: shortGraphId(graph.id),
        duplicateName: (names.get(normalized(name)) ?? 0) > 1,
        callCount: locations.length,
        references: locations,
        entryBound: (graph.entries?.length ?? 0) === 1,
        dataInputCount: graph.inputs.length,
        dataOutputCount: graph.outputs.length,
        execExitCount: exits.filter((exit) => exit.channel === 'exec').length,
        errorExitCount: exits.filter((exit) => exit.channel === 'error').length,
        interfaceHealthy:
          graph.kind === 'main' || ((graph.entries?.length ?? 0) === 1 && exits.length > 0),
      }
    })
    .filter((item) => !query || graphSearchText(item).includes(query))
    .sort((left, right) => {
      if (left.id === source.entryGraph) return -1
      if (right.id === source.entryGraph) return 1
      const byName = left.name.localeCompare(right.name)
      return byName || left.id.localeCompare(right.id)
    })
}

function graphName(graph: Graph | undefined): string {
  return graph?.name?.trim() || graph?.id || ''
}

function shortGraphId(id: string): string {
  if (id.length <= 18) return id
  return `${id.slice(0, 10)}…${id.slice(-6)}`
}

function graphSearchText(item: GraphDefinitionSummary): string {
  return normalized(
    [
      item.name,
      item.id,
      item.shortId,
      ...item.references.flatMap((reference) => [
        reference.parentGraphName,
        reference.parentGraphId,
        reference.callLabel,
        reference.callId,
      ]),
    ].join(' '),
  )
}

function normalized(value: string): string {
  return value.trim().toLocaleLowerCase()
}
