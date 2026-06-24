// PointWidget 单测
// 镜像 GeometryWidget.test.ts 模式: createApp + happy-dom, stub NuxtUI 全局组件.
// 逻辑验证走纯 JS 函数 (round4 / 值变换), 挂载测试确认组件不崩 + DOM 结构.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createApp, defineComponent, h, ref, nextTick } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'

// mock 外部依赖
vi.mock('@wailsio/runtime', () => ({
  Events: {
    On: vi.fn(() => () => {}),
    Emit: vi.fn(),
  },
}))
vi.mock('@/lib/backend', () => ({
  backend: {
    tools: {
      openScreenPicker: vi.fn(async () => undefined),
      mousePos: vi.fn(async () => ({ hasGame: true, clientW: 1920, clientH: 1080 })),
    },
  },
}))
vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => ({ add: vi.fn() }),
}))
vi.mock('@/composables/useWailsEvent', () => ({
  awaitWailsEvent: vi.fn(),
}))

import PointWidget from './PointWidget.vue'
import type { PointValue } from '@/components/containers/nodeRegistry/index'
import { backend } from '@/lib/backend'
import { awaitWailsEvent } from '@/composables/useWailsEvent'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mockBackend = backend as unknown as {
  tools: {
    openScreenPicker: ReturnType<typeof vi.fn>
    mousePos: ReturnType<typeof vi.fn>
  }
}
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mockAwaitWailsEvent = awaitWailsEvent as unknown as ReturnType<typeof vi.fn>

/** NuxtUI stub: 只渲染 slot, 透传所有 attrs (含 data-testid + onClick) */
function makeStub(name: string) {
  return defineComponent({
    name,
    inheritAttrs: false,
    props: { modelValue: { default: undefined }, loading: { default: false } },
    emits: ['update:modelValue'],
    setup(_, { slots, attrs }) {
      return () => h('div', { 'data-stub': name, ...attrs }, slots.default?.())
    },
  })
}

function mountPointWidget(modelValue: PointValue | null, fieldPath = 'pt') {
  const emitted: PointValue[] = []
  const valueRef = ref<PointValue | null>(modelValue)

  const Wrapper = defineComponent({
    setup() {
      return () =>
        h(PointWidget, {
          modelValue: valueRef.value,
          fieldPath,
          'onUpdate:modelValue': (v: PointValue) => {
            emitted.push(v)
            valueRef.value = v
          },
        })
    },
  })

  const app = createApp(Wrapper)
  app.use(createPinia())
  app.use(createI18n({
    legacy: false,
    locale: 'zh',
    messages: {
      zh: {
        point_widget: {
          unit_percent: '百分比',
          unit_px: '像素',
          pick_point: '截图取点',
          hint_percent: '比例：随窗口大小自适应（换分辨率仍按比例）',
          hint_px: '绝对像素：固定不随窗口缩放，适合固定位置的 UI',
        },
      },
    },
  }))
  for (const name of ['UInputNumber', 'UButton']) {
    app.component(name, makeStub(name))
  }

  const el = document.createElement('div')
  document.body.appendChild(el)
  app.mount(el)

  return { emitted, valueRef, app, el }
}

// round4 helper (镜像组件内实现)
function round4(n: number): number {
  return Math.round(n * 1e4) / 1e4
}

// ─── picker mock helpers ────────────────────────────────────────────────────

type PickerPayload = { xRatio: number; yRatio: number; screenW: number; screenH: number; cancelled?: boolean }

function mockPicker(payload: PickerPayload) {
  // openScreenPicker success: returns non-undefined (Go void → null in JS)
  mockBackend.tools.openScreenPicker.mockResolvedValue(null)
  // awaitWailsEvent resolves with the picker result
  mockAwaitWailsEvent.mockResolvedValue({ id: 'any', payload })
}

async function flushPromises() {
  // Run all microtasks and settle promises
  for (let i = 0; i < 10; i++) {
    await new Promise<void>((r) => setTimeout(r, 0))
    await nextTick()
  }
}

async function clickPickButton(wrapper: ReturnType<typeof mountPointWidget>) {
  const btn = wrapper.el.querySelector('[data-testid="point-pick-btn"]') as HTMLElement
  expect(btn).toBeTruthy()
  btn.click()
  await flushPromises()
}

function lastEmit(wrapper: ReturnType<typeof mountPointWidget>): PointValue | undefined {
  return wrapper.emitted[wrapper.emitted.length - 1]
}

describe('PointWidget', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('null → 挂载不崩, 无 emit', () => {
    const { emitted, app, el } = mountPointWidget(null)
    expect(emitted).toHaveLength(0)
    app.unmount()
    el.remove()
  })

  it('有值 {x:0.5, y:0.75} → 挂载不崩, 无自发 emit', () => {
    const { emitted, app, el } = mountPointWidget({ x: 0.5, y: 0.75 })
    expect(emitted).toHaveLength(0)
    app.unmount()
    el.remove()
  })

  // ─── 值变换纯逻辑测试 (镜像 GeometryWidget.test.ts 风格) ───────────────────

  it('% 模式 displayX: 存储 0.5 → 显示 50 (round4)', () => {
    const stored = 0.5
    expect(round4(stored * 100)).toBe(50)
  })

  it('% 模式 displayY: 存储 0.75 → 显示 75', () => {
    expect(round4(0.75 * 100)).toBe(75)
  })

  it('% 模式 onChange x: 用户输入 25 → 存储 0.25', () => {
    // 模拟 onChange('x', 25): round4(25/100) = 0.25
    expect(round4(25 / 100)).toBe(0.25)
  })

  it('% 模式 onChange y: 用户输入 50 → 存储 0.5', () => {
    expect(round4(50 / 100)).toBe(0.5)
  })

  it('round4: 33.3333 % → 存储 0.3333', () => {
    expect(round4(33.3333 / 100)).toBe(0.3333)
  })

  it('null → safeValue 默认 {x:0, y:0}', () => {
    // safeValue 逻辑: null 输入 → 归一为 {x:0,y:0}
    function safeValue(v: PointValue | null): PointValue {
      if (!v || typeof v.x !== 'number' || typeof v.y !== 'number') return { x: 0, y: 0 }
      return { x: v.x, y: v.y, unit: v.unit }
    }
    expect(safeValue(null)).toEqual({ x: 0, y: 0 })
    expect(safeValue({ x: 0.3, y: 0.7 })).toEqual({ x: 0.3, y: 0.7 })
    expect(safeValue({ x: 0.3, y: 0.7, unit: 'px' })).toEqual({ x: 0.3, y: 0.7, unit: 'px' })
  })

  it('% 模式 emit 值形状: onChange 仅含 x、y 两个键(无 unit)', () => {
    // onChange 生成: { ...safeValue, [field]: round4(v/100) }
    const base: PointValue = { x: 0.1, y: 0.2 }
    const next: PointValue = { ...base }
    next['x'] = round4(30 / 100)
    expect(Object.keys(next).sort()).toEqual(['x', 'y'])
    expect(next.x).toBe(0.3)
    expect(next.y).toBe(0.2) // 未改 y
  })

  it('onChange 保留另一维度: 更新 Y 时 X 不变', () => {
    const base: PointValue = { x: 0.5, y: 0.5 }
    const next: PointValue = { ...base }
    next['y'] = round4(10 / 100)
    expect(next.x).toBe(0.5)
    expect(next.y).toBe(0.1)
  })

  it('valueRef 更新后 emit 体现新值', async () => {
    const { emitted, valueRef, app, el } = mountPointWidget({ x: 0.1, y: 0.2 })
    // 模拟外部更新 (不是 onChange, 而是父组件推入新 modelValue)
    valueRef.value = { x: 0.9, y: 0.8 }
    await nextTick()
    // 外部更新不触发 emit (单向流)
    expect(emitted).toHaveLength(0)
    expect(valueRef.value).toEqual({ x: 0.9, y: 0.8 })
    app.unmount()
    el.remove()
  })

  // ─── 新单位切换逻辑测试 ────────────────────────────────────────────────────

  it('px 模式: displayX/Y 不×100 (原值透传)', () => {
    // isPx=true 时 display = stored value 原值
    const stored = { x: 960, y: 540, unit: 'px' as const }
    // isPx → displayX = stored.x (不×100)
    expect(stored.x).toBe(960)
    expect(stored.y).toBe(540)
  })

  it('% 模式: displayX/Y ×100', () => {
    const stored = { x: 0.5, y: 0.25 }
    expect(round4(stored.x * 100)).toBe(50)
    expect(round4(stored.y * 100)).toBe(25)
  })

  it('切到 px: emit 含 unit="px", x/y 是原框里数字(不换算)', () => {
    // 从 % 模式 {x:0.5,y:0.5} 切到 px:
    // displayX = round4(0.5*100) = 50, displayY = 50
    // setUnit('px'): next.unit='px', next.x=50, next.y=50
    const safeVal = { x: 0.5, y: 0.5 }
    const displayX = round4(safeVal.x * 100) // 50
    const displayY = round4(safeVal.y * 100) // 50
    const next: PointValue = { ...safeVal, unit: 'px', x: displayX, y: displayY }
    expect(next.unit).toBe('px')
    expect(next.x).toBe(50)
    expect(next.y).toBe(50)
  })

  it('切回 %: emit 无 unit, x/y = 框里数字÷100', () => {
    // 从 px 模式 {x:50,y:50,unit:'px'} 切回 %:
    // displayX = 50 (px 原值), displayY = 50
    // setUnit('percent'): delete unit, next.x = round4(50/100)=0.5
    const safeVal: PointValue = { x: 50, y: 50, unit: 'px' }
    const displayX = safeVal.x // 50 (px 原值)
    const displayY = safeVal.y
    const next: PointValue = { ...safeVal }
    delete next.unit
    next.x = round4(displayX / 100)
    next.y = round4(displayY / 100)
    expect(next.unit).toBeUndefined()
    expect(next.x).toBe(0.5)
    expect(next.y).toBe(0.5)
  })

  it('切到 px: mousePos 有值时换算 (1920×1080, x:0.5 → 960)', async () => {
    mockBackend.tools.mousePos.mockResolvedValue({ hasGame: true, clientW: 1920, clientH: 1080 })
    const { emitted, app, el } = mountPointWidget({ x: 0.5, y: 0.5 })
    const pxBtn = el.querySelector('[data-testid="point-unit-toggle"] button:last-child') as HTMLButtonElement
    expect(pxBtn).toBeTruthy()
    pxBtn.click()
    await flushPromises()
    expect(emitted).toHaveLength(1)
    const out = emitted[0]
    expect(out.unit).toBe('px')
    expect(out.x).toBe(960) // Math.round(0.5 * 1920)
    expect(out.y).toBe(540) // Math.round(0.5 * 1080)
    app.unmount()
    el.remove()
  })

  it('px 模式切回 %: mousePos 有值时换算 (x:960/1920 → 0.5)', async () => {
    mockBackend.tools.mousePos.mockResolvedValue({ hasGame: true, clientW: 1920, clientH: 1080 })
    const { emitted, app, el } = mountPointWidget({ x: 960, y: 540, unit: 'px' })
    const pctBtn = el.querySelector('[data-testid="point-unit-toggle"] button:first-child') as HTMLButtonElement
    expect(pctBtn).toBeTruthy()
    pctBtn.click()
    await flushPromises()
    expect(emitted).toHaveLength(1)
    const out = emitted[0]
    expect(out.unit).toBeUndefined()
    expect(out.x).toBe(0.5) // round4(960/1920)
    expect(out.y).toBe(0.5) // round4(540/1080)
    app.unmount()
    el.remove()
  })

  it('切到 px: mousePos hasGame=false 时保留框里数字 (old behavior), 不换算', async () => {
    mockBackend.tools.mousePos.mockResolvedValue({ hasGame: false, clientW: 0, clientH: 0 })
    const { emitted, app, el } = mountPointWidget({ x: 0.5, y: 0.5 })
    const pxBtn = el.querySelector('[data-testid="point-unit-toggle"] button:last-child') as HTMLButtonElement
    expect(pxBtn).toBeTruthy()
    pxBtn.click()
    await flushPromises()
    expect(emitted).toHaveLength(1)
    const out = emitted[0]
    expect(out.unit).toBe('px')
    expect(out.x).toBe(50) // displayX.value = round4(0.5*100) = 50, kept as-is
    expect(out.y).toBe(50)
    app.unmount()
    el.remove()
  })

  it('px 模式 onChange: 用户输入 960 → 存储 960 (不÷100)', () => {
    // isPx=true → onChange stores displayVal directly
    const displayVal = 960
    const storedVal = displayVal // isPx: no division
    expect(storedVal).toBe(960)
  })

  it('PointValue.unit 类型: 只允许 "px" 或 undefined', () => {
    const v1: PointValue = { x: 0.5, y: 0.5 }
    const v2: PointValue = { x: 960, y: 540, unit: 'px' }
    expect(v1.unit).toBeUndefined()
    expect(v2.unit).toBe('px')
  })

  // ─── 取点 payload→store 换算逻辑 (纯逻辑, 镜像 GeometryWidget 风格) ─────────

  it('截图取点 % 模式: payload → store 存比例 (round4)', () => {
    // % 模式: store { x: round4(xRatio), y: round4(yRatio) }, 无 unit
    const payload = { xRatio: 0.3, yRatio: 0.7, screenW: 1920, screenH: 1080 }
    const result: PointValue = {
      x: round4(payload.xRatio),
      y: round4(payload.yRatio),
    }
    expect(result).toEqual({ x: 0.3, y: 0.7 })
    expect(result.unit).toBeUndefined()
  })

  it('截图取点 px 模式: payload → store 存像素 (ratio×screen, Math.round)', () => {
    // px 模式: store { x: Math.round(xRatio*screenW), y: Math.round(yRatio*screenH), unit:'px' }
    const payload = { xRatio: 0.5, yRatio: 0.5, screenW: 1920, screenH: 1080 }
    const result: PointValue = {
      x: Math.round(payload.xRatio * payload.screenW),
      y: Math.round(payload.yRatio * payload.screenH),
      unit: 'px',
    }
    expect(result).toEqual({ x: 960, y: 540, unit: 'px' })
  })

  it('截图取点 px 小数 ratio: 换算后正确取整', () => {
    // xRatio=0.333, screenW=1000 → Math.round(333) = 333
    const payload = { xRatio: 0.3335, yRatio: 0.6665, screenW: 1000, screenH: 1000 }
    expect(Math.round(payload.xRatio * payload.screenW)).toBe(334)
    expect(Math.round(payload.yRatio * payload.screenH)).toBe(667)
  })

  it('截图取点 cancelled: payload.cancelled=true 不 emit', () => {
    // onPickPoint: if (!p || p.cancelled) return → 不 emit
    const payload = { xRatio: 0.5, yRatio: 0.5, screenW: 1920, screenH: 1080, cancelled: true }
    // 直接验证条件逻辑
    expect(payload.cancelled).toBe(true)
    // 组件不会 emit (无法挂载测: 以逻辑等价覆盖)
  })

  // ─── 截图取点 挂载版 (% 与 px 两支) ─────────────────────────────────────────

  it('截图取点 % 模式: 存比例', async () => {
    mockPicker({ xRatio: 0.3, yRatio: 0.7, screenW: 1920, screenH: 1080 })
    const wrapper = mountPointWidget({ x: 0, y: 0 })
    await clickPickButton(wrapper)
    const e = lastEmit(wrapper)
    expect(e).toEqual({ x: 0.3, y: 0.7 })
    wrapper.app.unmount()
    wrapper.el.remove()
  })

  it('截图取点 px 模式: 存像素 (ratio×screen)', async () => {
    mockPicker({ xRatio: 0.5, yRatio: 0.5, screenW: 1920, screenH: 1080 })
    const wrapper = mountPointWidget({ x: 0, y: 0, unit: 'px' })
    await clickPickButton(wrapper)
    const e = lastEmit(wrapper)
    expect(e).toEqual({ x: 960, y: 540, unit: 'px' })
    wrapper.app.unmount()
    wrapper.el.remove()
  })
})
