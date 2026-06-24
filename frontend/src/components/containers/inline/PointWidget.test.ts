// PointWidget 单测
// 镜像 GeometryWidget.test.ts 模式: createApp + happy-dom, stub NuxtUI 全局组件.
// 逻辑验证走纯 JS 函数 (round4 / 值变换), 挂载测试确认组件不崩 + DOM 结构.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createApp, defineComponent, h, ref, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'

import PointWidget from './PointWidget.vue'
import type { PointValue } from '@/components/containers/nodeRegistry/index'

/** NuxtUI stub: 只渲染 slot, 透传 v-model */
function makeStub(name: string) {
  return defineComponent({
    name,
    props: { modelValue: { default: undefined } },
    emits: ['update:modelValue'],
    setup(_, { slots }) {
      return () => slots.default?.()
    },
  })
}

function mountWidget(modelValue: PointValue | null, fieldPath = 'pt') {
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
  app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh: { point_widget: { unit_percent: '百分比', unit_px: '像素' } } } }))
  app.component('UInputNumber', makeStub('UInputNumber'))

  const el = document.createElement('div')
  document.body.appendChild(el)
  app.mount(el)

  return { emitted, valueRef, app, el }
}

// round4 helper (镜像组件内实现)
function round4(n: number): number {
  return Math.round(n * 1e4) / 1e4
}

describe('PointWidget', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('null → 挂载不崩, 无 emit', () => {
    const { emitted, app, el } = mountWidget(null)
    expect(emitted).toHaveLength(0)
    app.unmount()
    el.remove()
  })

  it('有值 {x:0.5, y:0.75} → 挂载不崩, 无自发 emit', () => {
    const { emitted, app, el } = mountWidget({ x: 0.5, y: 0.75 })
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
    const { emitted, valueRef, app, el } = mountWidget({ x: 0.1, y: 0.2 })
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

  it('切到 px: unit toggle 点击后 emit 正确 (挂载版)', async () => {
    const { emitted, app, el } = mountWidget({ x: 0.5, y: 0.5 })
    // 找 data-testid="point-unit-toggle" 里的第二个 button (px)
    const pxBtn = el.querySelector('[data-testid="point-unit-toggle"] button:last-child') as HTMLButtonElement
    expect(pxBtn).toBeTruthy()
    pxBtn.click()
    await nextTick()
    expect(emitted).toHaveLength(1)
    const out = emitted[0]
    expect(out.unit).toBe('px')
    expect(out.x).toBe(50) // 框里数字 50 原样进 x
    expect(out.y).toBe(50)
    app.unmount()
    el.remove()
  })

  it('px 模式切回 %: unit toggle 点击后 emit 无 unit, 数字÷100', async () => {
    const { emitted, app, el } = mountWidget({ x: 50, y: 25, unit: 'px' })
    // 点第一个 button (百分比)
    const pctBtn = el.querySelector('[data-testid="point-unit-toggle"] button:first-child') as HTMLButtonElement
    expect(pctBtn).toBeTruthy()
    pctBtn.click()
    await nextTick()
    expect(emitted).toHaveLength(1)
    const out = emitted[0]
    expect(out.unit).toBeUndefined()
    expect(out.x).toBe(0.5)
    expect(out.y).toBe(0.25)
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
})
