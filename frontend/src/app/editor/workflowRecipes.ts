import type { Edge, Node } from '../../../../contracts/workflow/3.1/workflow-source'
import type { NodeProjection } from '../../../../contracts/node/3.1/authoring-projection'
import type { EditorSession } from './EditorSession'

export type WorkflowRecipeID = 'analyze-color' | 'find-color-blobs'

export interface WorkflowRecipeDefinition {
  id: WorkflowRecipeID
  icon: string
  titleKey: string
  descriptionKey: string
  search: string
}

export const workflowRecipes: WorkflowRecipeDefinition[] = [
  {
    id: 'analyze-color',
    icon: 'color-filter',
    titleKey: 'workflow.recipes.analyze_color.title',
    descriptionKey: 'workflow.recipes.analyze_color.description',
    search: 'color detect analyze coverage match',
  },
  {
    id: 'find-color-blobs',
    icon: 'box-multiple',
    titleKey: 'workflow.recipes.find_color_blobs.title',
    descriptionKey: 'workflow.recipes.find_color_blobs.description',
    search: 'color blob locate connected component',
  },
]

const nodeIDs = {
  runStarted: 'https://schemas.yotta.dev/nodes/event/run-started',
  capture: 'https://schemas.yotta.dev/nodes/automation/capture-window',
  analyzeColor: 'https://schemas.yotta.dev/nodes/vision/analyze-color',
  findColorBlobs: 'https://schemas.yotta.dev/nodes/vision/find-color-blobs',
  listLength: 'https://schemas.yotta.dev/nodes/collection/length',
  greaterThan: 'https://schemas.yotta.dev/nodes/comparison/greater-than',
  branch: 'https://schemas.yotta.dev/nodes/control/branch',
} as const

export function insertWorkflowRecipe(
  session: EditorSession,
  recipeID: WorkflowRecipeID,
  origin: { x: number; y: number },
): string[] {
  const graph = session.currentGraph
  if (!graph) return []
  const built = buildWorkflowRecipe(recipeID, (typeID) => session.nodeProjection(typeID))
  const openRunRoot = graph.nodes.find((node) => {
    const projection = session.nodeInstanceProjection(node)
    return (
      projection?.instruction.kind === 'run-root' &&
      !graph.edges.some(
        (edge) =>
          edge.channel === 'exec' && edge.from.nodeId === node.id && edge.from.portId === 'started',
      )
    )
  })
  const includeRunRoot = !graph.nodes.some(
    (node) => session.nodeInstanceProjection(node)?.instruction.kind === 'run-root',
  )
  if (includeRunRoot) {
    const runRoot = createNode(
      requireProjection(session.nodeProjection(nodeIDs.runStarted)),
      'run',
      -280,
      0,
    )
    built.nodes.unshift(runRoot)
    built.edges.push(execEdge('run', 'started', 'capture', 'in'))
  }
  const inserted = session.insertNodeSelection(built, origin)
  if (openRunRoot && inserted[0]) {
    session.apply({
      kind: 'connect',
      edge: execEdge(openRunRoot.id, 'started', inserted[0], 'in'),
    })
  }
  return inserted
}

export function buildWorkflowRecipe(
  recipeID: WorkflowRecipeID,
  projectionFor: (typeID: string) => NodeProjection | undefined,
): { nodes: Node[]; edges: Edge[] } {
  const capture = createNode(requireProjection(projectionFor(nodeIDs.capture)), 'capture', 0, 0)
  const branch = createNode(requireProjection(projectionFor(nodeIDs.branch)), 'branch', 900, 0)
  if (recipeID === 'analyze-color') {
    const analyze = createNode(
      requireProjection(projectionFor(nodeIDs.analyzeColor)),
      'analyze',
      300,
      -40,
      {
        range: {
          kind: 'value',
          value: { space: 'hsv', minimum: [0, 50, 40], maximum: [15, 100, 100] },
        },
      },
    )
    const compare = createNode(
      requireProjection(projectionFor(nodeIDs.greaterThan)),
      'compare',
      620,
      20,
      { b: { kind: 'value', value: 0.2 } },
    )
    return {
      nodes: [capture, analyze, compare, branch],
      edges: [
        dataEdge('capture', 'image', 'analyze', 'image'),
        dataEdge('analyze', 'fraction', 'compare', 'a'),
        dataEdge('compare', 'result', 'branch', 'condition'),
        execEdge('capture', 'completed', 'branch', 'in'),
      ],
    }
  }

  const find = createNode(
    requireProjection(projectionFor(nodeIDs.findColorBlobs)),
    'find',
    280,
    -60,
    {
      range: {
        kind: 'value',
        value: { space: 'hsv', minimum: [0, 50, 40], maximum: [15, 100, 100] },
      },
    },
  )
  const length = createNode(
    requireProjection(projectionFor(nodeIDs.listLength)),
    'length',
    540,
    -80,
  )
  const compare = createNode(
    requireProjection(projectionFor(nodeIDs.greaterThan)),
    'compare',
    700,
    20,
    { b: { kind: 'value', value: 0 } },
  )
  return {
    nodes: [capture, find, length, compare, branch],
    edges: [
      dataEdge('capture', 'image', 'find', 'image'),
      dataEdge('find', 'blobs', 'length', 'list'),
      dataEdge('length', 'result', 'compare', 'a'),
      dataEdge('compare', 'result', 'branch', 'condition'),
      execEdge('capture', 'completed', 'branch', 'in'),
    ],
  }
}

function createNode(
  projection: NodeProjection,
  id: string,
  x: number,
  y: number,
  bindings: Node['bindings'] = {},
): Node {
  return {
    id,
    nodeRef: { ...projection.nodeRef },
    position: { x, y },
    config: {},
    bindings,
  }
}

function requireProjection(projection: NodeProjection | undefined): NodeProjection {
  if (!projection) throw new Error('workflow recipe requires an unavailable node contract')
  return projection
}

function dataEdge(fromNode: string, fromPort: string, toNode: string, toPort: string): Edge {
  return {
    channel: 'data',
    from: { nodeId: fromNode, portId: fromPort },
    to: { nodeId: toNode, portId: toPort },
  }
}

function execEdge(fromNode: string, fromPort: string, toNode: string, toPort: string): Edge {
  return {
    channel: 'exec',
    from: { nodeId: fromNode, portId: fromPort },
    to: { nodeId: toNode, portId: toPort },
  }
}
