import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import type { EditorCommand, EditorSession } from './EditorSession'
import { createEditorCanvasLayoutController } from './EditorCanvasLayoutController'

function createFixture() {
  const commands: EditorCommand[] = []
  const session = {
    source: { workflow: { graphs: [] } },
    currentGraph: {
      id: 'main',
      kind: 'root',
      name: 'Main',
      nodes: [
        {
          id: 'node-a',
          type: 'test',
          position: { x: 10, y: 20 },
          input: {},
        },
        {
          id: 'node-b',
          type: 'test',
          position: { x: 300, y: 80 },
          input: {},
        },
      ],
      calls: [
        {
          id: 'call-a',
          graphId: 'child',
          position: { x: 500, y: 100 },
          inputBindings: {},
        },
      ],
      annotations: [
        {
          id: 'note-a',
          text: 'note',
          position: { x: 700, y: 140 },
        },
      ],
      edges: [],
    },
  } as unknown as EditorSession
  const canvasElement = ref<HTMLElement | null>(null)
  const selectedNodeIds = ref(new Set(['node-a', 'node-b']))
  const controller = createEditorCanvasLayoutController({
    session,
    canvasElement,
    selectedNodeIds,
    findNode: () => undefined,
    fitView: vi.fn(async () => undefined),
    getViewport: () => ({ x: 40, y: -20, zoom: 0.75 }),
    applyCommand: (command) => {
      commands.push(command)
      return true
    },
    layoutErrorTitle: () => 'layout failed',
    showError: vi.fn(),
  })
  return { canvasElement, commands, controller, selectedNodeIds }
}

describe('EditorCanvasLayoutController', () => {
  it('aligns the selected nodes through one editor command', async () => {
    const { commands, controller } = createFixture()

    await controller.execute({ kind: 'align', mode: 'left' })

    expect(commands).toEqual([
      {
        kind: 'move-nodes',
        positions: [
          { nodeId: 'node-a', position: { x: 10, y: 20 } },
          { nodeId: 'node-b', position: { x: 10, y: 80 } },
        ],
      },
    ])
  })

  it('routes node, graph call, and annotation positions to their authoring commands', () => {
    const { commands, controller } = createFixture()

    expect(
      controller.applyPositions([
        { nodeId: 'node-a', position: { x: 1, y: 2 } },
        { nodeId: 'call-a', position: { x: 3, y: 4 } },
        { nodeId: 'note-a', position: { x: 5, y: 6 } },
      ]),
    ).toBe(true)

    expect(commands.map((command) => command.kind)).toEqual([
      'move-nodes',
      'update-graph-call',
      'update-annotation',
    ])
  })

  it('clears transient snap guides through the command seam', async () => {
    const { controller } = createFixture()
    controller.snapGuides.value = { x: 10, y: 20 }

    await controller.execute({ kind: 'clear-guides' })

    expect(controller.snapGuides.value).toEqual({})
  })

  it('keeps snap guides in the canvas-local coordinate system', () => {
    const { canvasElement, controller } = createFixture()
    canvasElement.value = { querySelectorAll: () => [] } as unknown as HTMLElement
    const event = {
      node: { id: 'node-a', position: { x: 66, y: 20 }, dimensions: {} },
      nodes: [],
      event: new MouseEvent('mousemove'),
    }

    controller.dragPositions(event as never)

    expect(controller.snapGuides.value.x).toBe(265)
  })

  it('makes one node-width gap after two selected nodes through one move command', async () => {
    const { commands, controller } = createFixture()

    await controller.execute({ kind: 'make-space' })

    expect(commands).toEqual([
      {
        kind: 'move-nodes',
        positions: [{ nodeId: 'node-b', position: { x: 580, y: 80 } }],
      },
    ])
  })
})
