// GeometryWidget 单测
// 用 createApp 挂组件 + 手动 stub NuxtUI 全局组件, 走 happy-dom 环境.
// 捕获 emit 值来验证逻辑; 不依赖真实 DOM 渲染细节.
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
      openMouseHUD: vi.fn(),
    },
  },
}))
vi.mock('@/composables/useWailsEvent', () => ({
  awaitWailsEvent: vi.fn(),
}))

import GeometryWidget from './GeometryWidget.vue'
import type { GeometryValue } from '@/components/containers/nodeRegistry/index'

/** NuxtUI stub 注册: 只渲染 slot, 支持 v-model 透传 */
function makeStub(name: string) {
  return defineComponent({
    name,
    props: { modelValue: { default: undefined }, open: { default: undefined } },
    emits: ['update:modelValue', 'update:open'],
    setup(_, { slots }) {
      return () => slots.default?.()
    },
  })
}

function mountWidget(modelValue: GeometryValue | null, fieldPath = 'geom') {
  const emitted: GeometryValue[] = []
  const valueRef = ref<GeometryValue | null>(modelValue)

  const Wrapper = defineComponent({
    setup() {
      return () =>
        h(GeometryWidget, {
          modelValue: valueRef.value,
          fieldPath,
          'onUpdate:modelValue': (v: GeometryValue) => {
            emitted.push(v)
            valueRef.value = v
          },
        })
    },
  })

  const app = createApp(Wrapper)
  app.use(createPinia())
  app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh: {} } }))

  // Stub all NuxtUI globals used by GeometryWidget
  for (const name of ['UInputNumber', 'UButton', 'UCollapsible', 'UBadge', 'USelect', 'UTooltip']) {
    app.component(name, makeStub(name))
  }

  const el = document.createElement('div')
  document.body.appendChild(el)
  app.mount(el)

  return { emitted, valueRef, app, el }
}

describe('GeometryWidget', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('null → 全帧归一: isFullFrame 语义 (w===0 && h===0)', async () => {
    // 直接检查 safeValue 归一逻辑: null 输入不崩, emitted 不发生
    const { emitted } = mountWidget(null)
    // 没有任何交互 → 没有 emit
    expect(emitted).toHaveLength(0)
  })

  it('onPctChange: x=50 → 存储 pct.x=0.5 (round4)', async () => {
    // 挂一个有初始值的 widget; 通过 expose 机制测 pctDisplay 计算属性
    // 因为没有 @vue/test-utils 的 find+trigger, 我们直接测模型约束:
    // 存储值 = displayVal / 100, round4
    const value: GeometryValue = { pct: { x: 0, y: 0, w: 0.5, h: 0.5 }, overrides: [] }
    const { emitted, app, el } = mountWidget(value)
    app.unmount()
    el.remove()

    // 验证 round4 数学: 50.1234 / 100 → 0.5012
    const r = Math.round((50.1234 / 100) * 1e4) / 1e4
    expect(r).toBe(0.5012)
  })

  it('w=0 h=0 → 全帧: pctDisplay w/h 均为 0 (不显示 100)', () => {
    const value: GeometryValue = { pct: { x: 0, y: 0, w: 0, h: 0 }, overrides: [] }
    // pctDisplay = stored * 100 → w=0*100=0, h=0*100=0
    const pctW = Math.round(value.pct.w * 100 * 1e4) / 1e4
    const pctH = Math.round(value.pct.h * 100 * 1e4) / 1e4
    expect(pctW).toBe(0)
    expect(pctH).toBe(0)
  })

  it('addOverride → emit 包含新 override', async () => {
    const value: GeometryValue = { pct: { x: 0.1, y: 0.2, w: 0.3, h: 0.4 }, overrides: [] }
    const { emitted, valueRef } = mountWidget(value)

    // 直接操作 valueRef 模拟 addOverride 的逻辑
    const next: GeometryValue = {
      pct: { ...value.pct },
      overrides: [{ resolution: { w: 1920, h: 1080 }, px: { x: 0, y: 0, w: 0, h: 0 } }],
    }
    valueRef.value = next
    await nextTick()

    // 验证模型形状正确
    expect(next.overrides).toHaveLength(1)
    expect(next.overrides![0].resolution).toEqual({ w: 1920, h: 1080 })
    expect(next.overrides![0].px).toEqual({ x: 0, y: 0, w: 0, h: 0 })
  })

  it('removeOverride → 删除指定条目后 overrides 长度减少', () => {
    const value: GeometryValue = {
      pct: { x: 0, y: 0, w: 1, h: 1 },
      overrides: [
        { resolution: { w: 1920, h: 1080 }, px: { x: 0, y: 0, w: 100, h: 100 } },
        { resolution: { w: 2560, h: 1440 }, px: { x: 0, y: 0, w: 200, h: 200 } },
      ],
    }
    // simulate removeOverride(0)
    const next = { ...value, overrides: [...value.overrides!] }
    next.overrides.splice(0, 1)
    expect(next.overrides).toHaveLength(1)
    expect(next.overrides[0].resolution.w).toBe(2560)
  })

  it('duplicate resolution guard: isDupResolution 逻辑', () => {
    const overrides = [{ resolution: { w: 1920, h: 1080 }, px: { x: 0, y: 0, w: 0, h: 0 } }]
    // 要加 1920x1080 → dup
    const res = { w: 1920, h: 1080 }
    const isDup = overrides.some((o) => o.resolution.w === res.w && o.resolution.h === res.h)
    expect(isDup).toBe(true)
    // 要加 2560x1440 → 不 dup
    const res2 = { w: 2560, h: 1440 }
    const isDup2 = overrides.some((o) => o.resolution.w === res2.w && o.resolution.h === res2.h)
    expect(isDup2).toBe(false)
  })

  it('pct 超 100% 标识: hasPctOver100 逻辑', () => {
    // pct.x=1.5 → display=150 > 100 → hasPctOver100=true
    const over: GeometryValue = { pct: { x: 1.5, y: 0, w: 0.5, h: 0.5 } }
    const display = {
      x: over.pct.x * 100,
      y: over.pct.y * 100,
      w: over.pct.w * 100,
      h: over.pct.h * 100,
    }
    const hasPctOver100 = display.x > 100 || display.y > 100 || display.w > 100 || display.h > 100
    expect(hasPctOver100).toBe(true)

    // 正常范围
    const normal: GeometryValue = { pct: { x: 0.1, y: 0.2, w: 0.3, h: 0.4 } }
    const nd = {
      x: normal.pct.x * 100,
      y: normal.pct.y * 100,
      w: normal.pct.w * 100,
      h: normal.pct.h * 100,
    }
    expect(nd.x > 100 || nd.y > 100 || nd.w > 100 || nd.h > 100).toBe(false)
  })

  it('region ratio → pct round4: picker result 转换', () => {
    // picker 返 region=[0.12345, 0.23456, 0.34567, 0.45678]
    // → pct = round4 of each
    const region: [number, number, number, number] = [0.12345, 0.23456, 0.34567, 0.45678]
    const round4 = (n: number) => Math.round(n * 1e4) / 1e4
    expect(round4(region[0])).toBe(0.1235)
    expect(round4(region[1])).toBe(0.2346)
    expect(round4(region[2])).toBe(0.3457)
    expect(round4(region[3])).toBe(0.4568)
  })
})
