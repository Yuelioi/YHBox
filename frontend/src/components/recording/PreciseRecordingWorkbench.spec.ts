import { createApp, defineComponent, h, nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

const { pendingEvents } = vi.hoisted(() => ({
  pendingEvents: vi.fn(async (_pendingId: string, offset: number, limit: number) => ({
    items: Array.from({ length: limit }, (_, index) => ({
      tUs: (offset + index) * 1000,
      seq: offset + index,
      type: 1,
      a: 87,
      b: 0,
      c: 0,
    })),
    total: 2500,
    offset,
    limit,
  })),
}))

vi.mock('@/lib/backend', () => ({
  backend: {
    recording: { pendingEvents },
    clips: { events: vi.fn() },
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

import PreciseRecordingWorkbench from './PreciseRecordingWorkbench.vue'

const mounted: Array<ReturnType<typeof createApp>> = []

afterEach(() => {
  for (const app of mounted.splice(0)) app.unmount()
  document.body.innerHTML = ''
  pendingEvents.mockClear()
})

describe('PreciseRecordingWorkbench', () => {
  it('renders independent tracks, warns about missing relative calibration, and pages raw events', async () => {
    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(PreciseRecordingWorkbench, {
      preview: {
        eventCount: 2500,
        keyCount: 2,
        clickCount: 1,
        moveCount: 100,
        scrollCount: 1,
        tracks: [
          { kind: 'keyboard', count: 2, firstUs: 0, lastUs: 50_000 },
          { kind: 'absolute-motion', count: 50, firstUs: 10_000, lastUs: 500_000 },
          { kind: 'relative-motion', count: 50, firstUs: 20_000, lastUs: 500_000 },
        ],
      },
      environment: {
        baseResolution: [1920, 1080],
        mouseMode: 'mixed',
        mouseCounts360: 0,
      },
      durationUs: 500_000,
      trimStartUs: 0,
      trimEndUs: 500_000,
      pendingId: 'pending-1',
      editableTrim: true,
    })
    app.component(
      'UButton',
      defineComponent({
        inheritAttrs: false,
        props: { label: String, disabled: Boolean },
        emits: ['click'],
        setup(props, { attrs, emit }) {
          return () =>
            h(
              'button',
              { ...attrs, disabled: props.disabled, onClick: () => emit('click') },
              props.label,
            )
        },
      }),
    )
    for (const name of ['UIcon', 'UInputNumber'])
      app.component(name, defineComponent({ setup: () => () => h('span') }))
    app.component(
      'UFormField',
      defineComponent({
        setup:
          (_, { slots }) =>
          () =>
            h('div', slots.default?.()),
      }),
    )
    mounted.push(app)
    app.mount(root)
    await nextTick()

    expect(root.textContent).toContain('preciseWorkbench.track_keyboard')
    expect(root.textContent).toContain('preciseWorkbench.track_absolute_motion')
    expect(root.textContent).toContain('preciseWorkbench.track_relative_motion')
    expect(root.textContent).toContain('preciseWorkbench.calibration_warning')

    const rawButton = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('preciseWorkbench.recording_details'),
    )
    expect(rawButton).toBeTruthy()
    rawButton?.click()
    await Promise.resolve()
    await nextTick()

    expect(pendingEvents).toHaveBeenCalledWith('pending-1', 0, 100)
    expect(root.querySelectorAll('[data-recording-event-row]')).toHaveLength(100)
    expect(root.textContent).toContain('seq=99')
    expect(root.textContent).toContain('/ 2500')
  })

  it('describes the timeline honestly and lets a pointer choose a trim boundary', async () => {
    const root = document.createElement('div')
    document.body.append(root)
    const trimStart = vi.fn()
    const trimEnd = vi.fn()
    const app = createApp(PreciseRecordingWorkbench, {
      preview: {
        eventCount: 4,
        keyCount: 2,
        clickCount: 0,
        moveCount: 2,
        scrollCount: 0,
        tracks: [{ kind: 'keyboard', count: 2, firstUs: 0, lastUs: 1_000_000 }],
      },
      environment: {
        baseResolution: [1280, 720],
        mouseMode: 'relative',
        mouseCounts360: 4134,
      },
      durationUs: 1_000_000,
      trimStartUs: 0,
      trimEndUs: 1_000_000,
      pendingId: 'pending-trim',
      editableTrim: true,
      'onUpdate:trimStartUs': trimStart,
      'onUpdate:trimEndUs': trimEnd,
    })
    app.component(
      'UButton',
      defineComponent({
        inheritAttrs: false,
        props: { label: String, disabled: Boolean },
        emits: ['click'],
        setup(props, { attrs, emit }) {
          return () =>
            h(
              'button',
              { ...attrs, disabled: props.disabled, onClick: () => emit('click') },
              props.label,
            )
        },
      }),
    )
    for (const name of ['UIcon', 'UInputNumber'])
      app.component(name, defineComponent({ setup: () => () => h('span') }))
    app.component(
      'UFormField',
      defineComponent({
        setup:
          (_, { slots }) =>
          () =>
            h('div', slots.default?.()),
      }),
    )
    mounted.push(app)
    app.mount(root)
    await nextTick()

    expect(root.textContent).toContain('preciseWorkbench.timeline_hint')
    expect(root.textContent).toContain('4134')
    expect(root.textContent).not.toContain('preciseWorkbench.play_preview')
    const timeline = root.querySelector<HTMLElement>('[data-recording-trim-timeline]')
    expect(timeline).toBeTruthy()
    vi.spyOn(timeline!, 'getBoundingClientRect').mockReturnValue({
      x: 0,
      y: 0,
      left: 0,
      top: 0,
      right: 100,
      bottom: 20,
      width: 100,
      height: 20,
      toJSON: () => ({}),
    })
    timeline?.dispatchEvent(new MouseEvent('click', { bubbles: true, clientX: 25 }))
    expect(trimStart).toHaveBeenCalledWith(250_000)
    expect(trimEnd).not.toHaveBeenCalled()
  })
})
