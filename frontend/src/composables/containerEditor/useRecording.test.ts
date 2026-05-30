// useRecording 归属守卫单测 (③ 多窗口误报根因修):
// recording:completed 是全局广播 — 没在本窗发起录制 (ownsRecording=false) 时, 处理器必须静默 return,
// 不弹 toast / 不加节点. 否则非目标容器的编辑器窗口会误报 container_mismatch.
//
// 用真 i18n 实例 + 真 pinia 挂一个 host 组件跑 useRecording (composable 顶层调 useI18n + onMounted,
// 必须在 setup 上下文里). Events.On 被 mock 以捕获 'recording:completed' 处理器, 测试里直接触发它.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createApp, defineComponent, h, ref, computed, type Ref } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import type { Container, Graph } from '@/lib/backend'

// 捕获 composable 在 onMounted 里注册的事件处理器.
const handlers: Record<string, (ev: any) => any> = {}
vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: vi.fn((name: string, cb: (ev: any) => any) => {
      handlers[name] = cb
      return () => {}
    }),
    Emit: vi.fn(),
  },
}))

vi.mock('@/lib/backend', () => ({
  backend: {
    recording: {
      getState: vi.fn(async () => ({ phase: 'idle' })),
      start: vi.fn(),
      stop: vi.fn(),
      pause: vi.fn(),
      resume: vi.fn(),
    },
    tools: { openRecordingHUD: vi.fn(), closeRecordingHUD: vi.fn() },
    containers: { get: vi.fn(async () => ({ id: 'cOther', name: '别的容器' })) },
  },
}))

import { useRecording } from './useRecording'

function makeDraft(): Container {
  return {
    id: 'cMine',
    name: '我的容器',
    graph: { id: 'g', version: 1, nodes: [{ id: 'start', kind: 'Start', x: 0, y: 0, config: {} }], edges: [] },
  } as unknown as Container
}

function mountComposable(draft: Ref<Container | null>) {
  const toast = { add: vi.fn() }
  const activeGraph = computed<Graph | null>(() => draft.value?.graph ?? null)
  const app = createApp(
    defineComponent({
      setup() {
        useRecording({
          draft,
          activeGraph,
          syncFlowFromDraft: vi.fn(),
          refreshSubgraphStore: vi.fn(async () => {}),
          saveDraft: vi.fn(async () => {}),
          toast,
        })
        return () => h('div')
      },
    }),
  )
  app.use(createPinia())
  app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh: {} } }))
  app.mount(document.createElement('div'))
  return { toast, app }
}

describe('useRecording — recording:completed 归属守卫', () => {
  beforeEach(() => {
    for (const k of Object.keys(handlers)) delete handlers[k]
    vi.clearAllMocks()
  })

  it('onMounted 注册 recording:completed 处理器', () => {
    mountComposable(ref(makeDraft()))
    expect(typeof handlers['recording:completed']).toBe('function')
  })

  it('非 owner 窗口 (未发起录制) 收 completed → 静默: 不弹 toast, 不加节点', async () => {
    const draft = ref(makeDraft())
    const { toast } = mountComposable(draft)
    const before = draft.value!.graph.nodes.length

    // 模拟别的窗口录完的全局广播 — 本窗 ownsRecording 仍是 false.
    await handlers['recording:completed']({
      data: [{ subgraphID: 'sg-x', containerID: 'cOther', label: 'x', filterMode: 'precise' }],
    })

    expect(toast.add).not.toHaveBeenCalled()
    expect(draft.value!.graph.nodes.length).toBe(before) // 没加 Subgraph 节点
  })
})
