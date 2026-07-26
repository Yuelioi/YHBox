import { createApp, defineComponent, h, nextTick, ref, computed } from 'vue'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { afterEach, describe, expect, it } from 'vitest'
import type { Node, NodeProjection } from './EditorSession'
import {
  createWorkflowNodeGestureState,
  projectWorkflowFlowNodes,
  WORKFLOW_NODE_DRAG_HANDLE,
} from './workflowFlowProjection'

const mountedApps: Array<ReturnType<typeof createApp>> = []

afterEach(() => {
  for (const app of mountedApps.splice(0)) app.unmount()
  document.body.innerHTML = ''
})

describe('workflow flow projection', () => {
  it('does not replace a live gesture position when selection changes', async () => {
    const selected = ref(false)
    const durablePosition = { x: 40, y: 60 }
    const node = {
      id: 'node-a',
      nodeRef: { nodeTypeId: 'test-node' },
      position: durablePosition,
    } as Node
    const projection = {} as NodeProjection
    const gestures = createWorkflowNodeGestureState()
    const projectedNodes = computed(() =>
      projectWorkflowFlowNodes([node], () => projection, gestures.positions),
    )
    const flow = useVueFlow('projection-drift')
    const Harness = defineComponent({
      setup: () => () => h(VueFlow, { id: 'projection-drift', nodes: projectedNodes.value }),
    })
    const root = document.createElement('div')
    root.style.width = '800px'
    root.style.height = '600px'
    document.body.append(root)
    const app = createApp(Harness)
    mountedApps.push(app)
    app.mount(root)
    await nextTick()

    flow.updateNode('node-a', { position: { x: 320, y: 240 } })
    expect(flow.findNode('node-a')?.position).toEqual({ x: 320, y: 240 })
    expect(projectedNodes.value[0]?.dragHandle).toBe(WORKFLOW_NODE_DRAG_HANDLE)

    selected.value = true
    await nextTick()

    expect(flow.findNode('node-a')?.position).toEqual({ x: 320, y: 240 })
  })

  it('does not replace a live gesture position when the source refreshes', async () => {
    const durablePosition = { x: 40, y: 60 }
    const domainNodes = ref([
      {
        id: 'node-a',
        nodeRef: { nodeTypeId: 'test-node' },
        position: durablePosition,
      } as Node,
    ])
    const projection = {} as NodeProjection
    const gestures = createWorkflowNodeGestureState()
    const projectedNodes = computed(() =>
      projectWorkflowFlowNodes(domainNodes.value, () => projection, gestures.positions),
    )
    const flow = useVueFlow('projection-source-refresh')
    const Harness = defineComponent({
      setup: () => () =>
        h(VueFlow, { id: 'projection-source-refresh', nodes: projectedNodes.value }),
    })
    const root = document.createElement('div')
    root.style.width = '800px'
    root.style.height = '600px'
    document.body.append(root)
    const app = createApp(Harness)
    mountedApps.push(app)
    app.mount(root)
    await nextTick()

    gestures.track('node-a', { x: 320, y: 240 })
    flow.updateNode('node-a', { position: { x: 320, y: 240 } })
    domainNodes.value = [{ ...domainNodes.value[0]!, label: 'renamed during drag' }]
    await nextTick()

    expect(flow.findNode('node-a')?.position).toEqual({ x: 320, y: 240 })
  })
})
