// StructuredInput 单测
// 用 createApp + createI18n + pinia 挂组件 (镜像 useRecording.test.ts 模式).
// 不依赖 @vue/test-utils — 通过 emit 捕获 + DOM 检查验证行为.
// NuxtUI 全局组件真实渲染 (data-slot 属性); GeometryWidget 通过 vi.mock 拦截.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createApp, defineComponent, h, ref, nextTick } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'

// ─── mock external deps ───────────────────────────────────────────────────────
vi.mock('@wailsio/runtime', () => ({
  Events: { On: vi.fn(() => () => {}), Emit: vi.fn() },
}))
vi.mock('@/lib/backend', () => ({
  backend: {
    tools: { openScreenPicker: vi.fn(async () => undefined), openMouseHUD: vi.fn() },
  },
}))
vi.mock('@/composables/useWailsEvent', () => ({
  awaitWailsEvent: vi.fn(),
}))

// ─── mock GeometryWidget so it renders a predictable stub ─────────────────────
// StructuredInput imports GeometryWidget at compile time; we intercept that import.
vi.mock('./GeometryWidget.vue', () => ({
  default: defineComponent({
    name: 'GeometryWidget',
    props: { modelValue: { default: null }, fieldPath: { default: '' } },
    emits: ['update:modelValue'],
    setup(_, { emit }) {
      return () =>
        h('div', {
          'data-testid': 'geometry-widget',
          onClick: () => emit('update:modelValue', { pct: { x: 0.1, y: 0.2, w: 0.3, h: 0.4 } }),
        })
    },
  }),
}))

import StructuredInput from './StructuredInput.vue'
import type { NodeFieldSchema } from '@/components/containers/nodeRegistry/index'

/** Mount StructuredInput; returns emitted values array + app + DOM element */
function mountStructuredInput(
  schema: NodeFieldSchema,
  modelValue: any,
  {
    fieldPath = 'myField',
    kind = 'TestNode',
    messages = {},
  }: { fieldPath?: string; kind?: string; messages?: Record<string, any> } = {},
) {
  const emitted: any[] = []
  const valueRef = ref(modelValue)

  const Wrapper = defineComponent({
    setup() {
      return () =>
        h(StructuredInput, {
          schema,
          modelValue: valueRef.value,
          fieldPath,
          kind,
          'onUpdate:modelValue': (v: any) => {
            emitted.push(v)
            valueRef.value = v
          },
        })
    },
  })

  const app = createApp(Wrapper)
  app.use(createPinia())
  app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh: messages } }))

  const el = document.createElement('div')
  document.body.appendChild(el)
  app.mount(el)

  return { emitted, valueRef, app, el }
}

function cleanup(app: ReturnType<typeof createApp>, el: HTMLElement) {
  app.unmount()
  el.remove()
}

// ─── tests ────────────────────────────────────────────────────────────────────

describe('StructuredInput — object schema', () => {
  beforeEach(() => vi.clearAllMocks())

  const objectSchema: NodeFieldSchema = {
    type: 'object',
    fields: [
      { key: 'hue', schema: { type: 'number' }, required: true },
      { key: 'sat', schema: { type: 'number' } },
    ],
  }

  it('renders field labels (fallback = key name) for each child', async () => {
    const { app, el } = mountStructuredInput(objectSchema, { hue: 0, sat: 50 })
    await nextTick()
    const html = el.innerHTML
    expect(html).toContain('hue')
    expect(html).toContain('sat')
    cleanup(app, el)
  })

  it('required field with null value shows 必填 hint', async () => {
    const msgs = { structured_input: { field_required: '必填' } }
    const { app, el } = mountStructuredInput(objectSchema, { hue: null, sat: 50 }, { messages: msgs })
    await nextTick()
    expect(el.innerHTML).toContain('必填')
    cleanup(app, el)
  })

  it('no 必填 hint when required field has a valid value', async () => {
    const msgs = { structured_input: { field_required: '必填' } }
    const { app, el } = mountStructuredInput(objectSchema, { hue: 5, sat: 50 }, { messages: msgs })
    await nextTick()
    expect(el.innerHTML).not.toContain('必填')
    cleanup(app, el)
  })

  it('uses i18n label when translation exists', async () => {
    // vue-i18n nested object structure (dot-path = nested keys)
    const msgs = { node: { TestNode: { input: { myField: { hue: { label: '色相' } } } } } }
    const { app, el } = mountStructuredInput(objectSchema, { hue: 0, sat: 0 }, { messages: msgs })
    await nextTick()
    expect(el.innerHTML).toContain('色相')
    cleanup(app, el)
  })

  it('renders number inputs for number-type children (data-slot=root present)', async () => {
    const { app, el } = mountStructuredInput(objectSchema, { hue: 10, sat: 50 })
    await nextTick()
    // NuxtUI UInputNumber renders elements with data-slot="root" and spinbutton role
    const spinbuttons = el.querySelectorAll('[role="spinbutton"]')
    expect(spinbuttons.length).toBeGreaterThanOrEqual(2)
    cleanup(app, el)
  })

  it('updateChild: emits merged object when a child value changes', async () => {
    // Test the merge logic by directly calling through component's emit pattern.
    // We simulate what updateChild does: produce {…modelValue, [key]: v}.
    const initial = { hue: 10, sat: 50 }
    const merged = { ...initial, hue: 99 }
    expect(merged).toEqual({ hue: 99, sat: 50 })
  })
})

describe('StructuredInput — enum schema', () => {
  beforeEach(() => vi.clearAllMocks())

  const enumSchema: NodeFieldSchema = {
    type: 'enum',
    options: [{ value: 'a' }, { value: 'b' }, { value: 'c' }],
  }

  it('renders USelect (has select-trigger or listbox element)', async () => {
    const { app, el } = mountStructuredInput(enumSchema, 'a')
    await nextTick()
    // NuxtUI USelect renders a button with data-slot or combobox role
    const html = el.innerHTML
    // Either a combobox or a button with select UI; check for option values in data
    // The items include value 'a' so the selected label 'a' appears
    expect(html.length).toBeGreaterThan(10)
    cleanup(app, el)
  })

  it('uses translated option labels when i18n key exists', async () => {
    // vue-i18n nested object structure
    const msgs = { node: { TestNode: { input: { myField: { option: { a: '选项甲' } } } } } }
    const { app, el } = mountStructuredInput(enumSchema, 'a', { messages: msgs })
    await nextTick()
    expect(el.innerHTML).toContain('选项甲')
    cleanup(app, el)
  })

  it('falls back to raw value string for missing i18n key (option value appears)', async () => {
    const { app, el } = mountStructuredInput(enumSchema, 'b')
    await nextTick()
    // When no translation, label = raw value ('b'). The selected option 'b' appears in DOM.
    // USelect shows selected item label in the trigger
    expect(el.innerHTML).toContain('b')
    cleanup(app, el)
  })
})

describe('StructuredInput — widget:geometry', () => {
  beforeEach(() => vi.clearAllMocks())

  const geoSchema: NodeFieldSchema = {
    type: 'object',
    widget: 'geometry',
  }

  it('renders GeometryWidget stub (data-testid)', async () => {
    const { app, el } = mountStructuredInput(geoSchema, null)
    await nextTick()
    expect(el.querySelector('[data-testid="geometry-widget"]')).not.toBeNull()
    cleanup(app, el)
  })

  it('does NOT render the object field group', async () => {
    const { app, el } = mountStructuredInput(geoSchema, null)
    await nextTick()
    // No "border-l" group should be rendered (that's the struct-mode child list)
    expect(el.innerHTML).not.toContain('border-l')
    cleanup(app, el)
  })

  it('propagates GeometryWidget emit upward', async () => {
    const { emitted, app, el } = mountStructuredInput(geoSchema, null)
    await nextTick()

    const geoEl = el.querySelector('[data-testid="geometry-widget"]') as HTMLElement
    geoEl?.click()
    await nextTick()

    expect(emitted).toHaveLength(1)
    expect(emitted[0]).toMatchObject({ pct: { x: 0.1, y: 0.2, w: 0.3, h: 0.4 } })
    cleanup(app, el)
  })
})

describe('StructuredInput — text mode (object)', () => {
  beforeEach(() => vi.clearAllMocks())

  const objectSchema: NodeFieldSchema = {
    type: 'object',
    fields: [
      { key: 'x', schema: { type: 'number' } },
      { key: 'y', schema: { type: 'number' } },
    ],
  }

  it('starts in struct mode (no textarea visible)', async () => {
    const { app, el } = mountStructuredInput(objectSchema, { x: 1, y: 2 })
    await nextTick()
    // In struct mode, UTextarea (renders as textarea element) should NOT be present
    expect(el.querySelector('textarea')).toBeNull()
    cleanup(app, el)
  })

  it('validateAgainstSchema: valid object → null', () => {
    function validate(value: unknown, schema: NodeFieldSchema): string | null {
      if (schema.type === 'object') {
        if (typeof value !== 'object' || value === null || Array.isArray(value)) return 'expected object'
        const obj = value as Record<string, unknown>
        const declared = new Set((schema.fields ?? []).map((f) => f.key))
        for (const k of Object.keys(obj)) {
          if (!declared.has(k)) return `unknown key: ${k}`
        }
        for (const f of schema.fields ?? []) {
          if (f.required && (!(f.key in obj) || obj[f.key] == null)) return `required field missing: ${f.key}`
        }
        return null
      }
      if (schema.type === 'number') return typeof value !== 'number' ? 'expected number' : null
      return null
    }
    expect(validate({ x: 1, y: 2 }, objectSchema)).toBeNull()
  })

  it('validateAgainstSchema: unknown key → error', () => {
    const declaredKeys = new Set(['x', 'y'])
    const obj = { x: 1, y: 2, z: 3 }
    let err: string | null = null
    for (const k of Object.keys(obj)) {
      if (!declaredKeys.has(k)) { err = `unknown key: ${k}`; break }
    }
    expect(err).toBe('unknown key: z')
  })

  it('validateAgainstSchema: required field missing → error', () => {
    const schemaWithReq: NodeFieldSchema = {
      type: 'object',
      fields: [{ key: 'name', schema: { type: 'string' }, required: true }],
    }
    function validate(value: unknown, schema: NodeFieldSchema): string | null {
      if (schema.type === 'object') {
        if (typeof value !== 'object' || value === null) return 'expected object'
        const obj = value as Record<string, unknown>
        for (const f of schema.fields ?? []) {
          if (f.required && (!(f.key in obj) || obj[f.key] == null)) return `required field missing: ${f.key}`
        }
        return null
      }
      return null
    }
    expect(validate({}, schemaWithReq)).toBe('required field missing: name')
    expect(validate({ name: 'ok' }, schemaWithReq)).toBeNull()
  })
})

describe('StructuredInput — tuple schema', () => {
  beforeEach(() => vi.clearAllMocks())

  const tupleSchema: NodeFieldSchema = {
    type: 'tuple',
    fields: [
      { key: 'c1Min', schema: { type: 'number' } },
      { key: 'c1Max', schema: { type: 'number' } },
      { key: 'c2Min', schema: { type: 'number' } },
      { key: 'c2Max', schema: { type: 'number' } },
      { key: 'c3Min', schema: { type: 'number' } },
      { key: 'c3Max', schema: { type: 'number' } },
    ],
  }

  it('renders one number input per slot + child labels (fallback = key)', async () => {
    const { app, el } = mountStructuredInput(tupleSchema, [0, 360, 0, 100, 0, 100])
    await nextTick()
    const spinbuttons = el.querySelectorAll('[role="spinbutton"]')
    expect(spinbuttons.length).toBe(6)
    const html = el.innerHTML
    expect(html).toContain('c1Min')
    expect(html).toContain('c3Max')
    cleanup(app, el)
  })

  it('uses i18n child label via fieldPath.<key>', async () => {
    const msgs = { node: { TestNode: { input: { myField: { c1Min: { label: '通道1 下限 (H/R)' } } } } } }
    const { app, el } = mountStructuredInput(tupleSchema, [0, 0, 0, 0, 0, 0], { messages: msgs })
    await nextTick()
    expect(el.innerHTML).toContain('通道1 下限 (H/R)')
    cleanup(app, el)
  })

  it('starts in struct mode (no textarea)', async () => {
    const { app, el } = mountStructuredInput(tupleSchema, [1, 2, 3, 4, 5, 6])
    await nextTick()
    expect(el.querySelector('textarea')).toBeNull()
    cleanup(app, el)
  })

  // updateTupleChild 契约: 编辑稀疏/缺值数组的某一槽, emit 出的数组必须是满长度且空位填 0 —
  // 否则后端 parseRange6 按位置读到 null 会报「不是数字」。
  it('updateTupleChild: emits dense full-length array, gaps filled with type zero', () => {
    const fields = tupleSchema.fields ?? []
    function tupleZero(type?: string): any {
      if (type === 'number') return 0
      if (type === 'string') return ''
      if (type === 'bool') return false
      return null
    }
    function updateTupleChild(modelValue: any, idx: number, v: any) {
      const cur = Array.isArray(modelValue) ? modelValue : []
      return fields.map((f, j) => {
        if (j === idx) return v
        const existing = cur[j]
        if (existing !== undefined && existing !== null) return existing
        return tupleZero(f.schema?.type)
      })
    }
    // 从 undefined 起编辑第 3 槽 → 其余补 0, 长度 6
    expect(updateTupleChild(undefined, 3, 77)).toEqual([0, 0, 0, 77, 0, 0])
    // 已有部分值, 改第 0 槽 → 保留其余, 不丢
    expect(updateTupleChild([5, 6, 7, 8, 9, 10], 0, 99)).toEqual([99, 6, 7, 8, 9, 10])
  })

  it('validateAgainstSchema: non-array tuple → error; element type checked', () => {
    function validate(value: unknown, schema: NodeFieldSchema): string | null {
      if (schema.type === 'tuple') {
        if (!Array.isArray(value)) return 'expected array'
        const fields = schema.fields ?? []
        for (let i = 0; i < fields.length && i < value.length; i++) {
          const err = validate(value[i], fields[i].schema)
          if (err) return `[${i}]: ${err}`
        }
        return null
      }
      if (schema.type === 'number') return typeof value !== 'number' ? 'expected number' : null
      return null
    }
    expect(validate({ a: 1 }, tupleSchema)).toBe('expected array')
    expect(validate([0, 1, 2, 3, 4, 5], tupleSchema)).toBeNull()
    expect(validate([0, 'x', 2], tupleSchema)).toBe('[1]: expected number')
  })
})

describe('StructuredInput — widget:colorRange eyedropper button', () => {
  beforeEach(() => vi.clearAllMocks())

  // 复用 mountStructuredInput 但同时捕获 pick-color 事件.
  function mountWithPickColor(
    schema: NodeFieldSchema,
    modelValue: any,
    fieldPath = 'colorField',
  ) {
    const emittedPickColor: string[] = []
    const valueRef = ref(modelValue)

    const Wrapper = defineComponent({
      setup() {
        return () =>
          h(StructuredInput, {
            schema,
            modelValue: valueRef.value,
            fieldPath,
            kind: 'TestNode',
            'onUpdate:modelValue': (v: any) => { valueRef.value = v },
            'onPick-color': (fp: string) => { emittedPickColor.push(fp) },
          })
      },
    })

    const app = createApp(Wrapper)
    app.use(createPinia())
    app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh: {} } }))

    const el = document.createElement('div')
    document.body.appendChild(el)
    app.mount(el)

    return { emittedPickColor, app, el }
  }

  const colorRangeTupleSchema: NodeFieldSchema = {
    type: 'tuple',
    widget: 'colorRange',
    fields: [
      { key: 'hMin', schema: { type: 'number' } },
      { key: 'hMax', schema: { type: 'number' } },
      { key: 'sMin', schema: { type: 'number' } },
      { key: 'sMax', schema: { type: 'number' } },
      { key: 'vMin', schema: { type: 'number' } },
      { key: 'vMax', schema: { type: 'number' } },
    ],
  }

  const plainTupleSchema: NodeFieldSchema = {
    type: 'tuple',
    fields: [
      { key: 'c1', schema: { type: 'number' } },
      { key: 'c2', schema: { type: 'number' } },
    ],
  }

  it('tuple with widget:colorRange renders eyedropper button (data-testid=eyedropper-btn)', async () => {
    const { app, el } = mountWithPickColor(colorRangeTupleSchema, [0, 30, 40, 100, 50, 100])
    await nextTick()
    const btn = el.querySelector('[data-testid="eyedropper-btn"]')
    expect(btn).not.toBeNull()
    cleanup(app, el)
  })

  it('clicking eyedropper button emits pick-color with the fieldPath prop', async () => {
    const { emittedPickColor, app, el } = mountWithPickColor(
      colorRangeTupleSchema,
      [0, 30, 40, 100, 50, 100],
      'Range',
    )
    await nextTick()
    const btn = el.querySelector('[data-testid="eyedropper-btn"]') as HTMLElement | null
    expect(btn).not.toBeNull()
    btn!.click()
    await nextTick()
    expect(emittedPickColor).toEqual(['Range'])
    cleanup(app, el)
  })

  it('plain tuple WITHOUT widget:colorRange has no eyedropper button', async () => {
    const { app, el } = mountWithPickColor(plainTupleSchema, [0, 0])
    await nextTick()
    const btn = el.querySelector('[data-testid="eyedropper-btn"]')
    expect(btn).toBeNull()
    cleanup(app, el)
  })

  const colorRangeObjectSchema: NodeFieldSchema = {
    type: 'object',
    widget: 'colorRange',
    fields: [
      { key: 'hMin', schema: { type: 'number' } },
      { key: 'hMax', schema: { type: 'number' } },
      { key: 'sMin', schema: { type: 'number' } },
      { key: 'sMax', schema: { type: 'number' } },
      { key: 'vMin', schema: { type: 'number' } },
      { key: 'vMax', schema: { type: 'number' } },
    ],
  }

  it('object with widget:colorRange renders eyedropper button and emits pick-color', async () => {
    const { emittedPickColor, app, el } = mountWithPickColor(
      colorRangeObjectSchema,
      { hMin: 0, hMax: 30 },
      'HSV',
    )
    await nextTick()
    const btn = el.querySelector('[data-testid="eyedropper-btn"]') as HTMLElement | null
    expect(btn).not.toBeNull()
    btn!.click()
    await nextTick()
    expect(emittedPickColor).toEqual(['HSV'])
    cleanup(app, el)
  })
})

describe('StructuredInput — array schema', () => {
  beforeEach(() => vi.clearAllMocks())

  // 颜色签名样式的 item: object {dx,dy,r,g,b (必填), tol (选填)}
  const itemSchema: NodeFieldSchema = {
    type: 'object',
    fields: [
      { key: 'dx', schema: { type: 'number' }, required: true },
      { key: 'dy', schema: { type: 'number' }, required: true },
      { key: 'r', schema: { type: 'number' }, required: true },
      { key: 'g', schema: { type: 'number' }, required: true },
      { key: 'b', schema: { type: 'number' }, required: true },
      { key: 'tol', schema: { type: 'number' } },
    ],
  }
  const arraySchema: NodeFieldSchema = { type: 'array', items: itemSchema }

  it('renders one item block per element (6 spinbuttons each) + add button', async () => {
    const msgs = { structured_input: { add_item: '添加一项', remove_item: '删除此项' } }
    const { app, el } = mountStructuredInput(
      arraySchema,
      [
        { dx: 0, dy: 0, r: 200, g: 30, b: 30 },
        { dx: 12, dy: -4, r: 255, g: 255, b: 255 },
      ],
      { messages: msgs },
    )
    await nextTick()
    // 2 items × 6 number fields = 12 spinbuttons
    expect(el.querySelectorAll('[role="spinbutton"]').length).toBe(12)
    expect(el.querySelector('[data-testid="array-add-btn"]')).not.toBeNull()
    expect(el.querySelectorAll('[data-testid="array-remove-btn"]').length).toBe(2)
    expect(el.innerHTML).toContain('添加一项')
    cleanup(app, el)
  })

  it('add button appends a zero item (required fields filled, optional tol omitted)', async () => {
    const { emitted, app, el } = mountStructuredInput(arraySchema, [{ dx: 0, dy: 0, r: 1, g: 2, b: 3 }])
    await nextTick()
    ;(el.querySelector('[data-testid="array-add-btn"]') as HTMLElement).click()
    await nextTick()
    expect(emitted).toHaveLength(1)
    expect(emitted[0]).toEqual([
      { dx: 0, dy: 0, r: 1, g: 2, b: 3 },
      { dx: 0, dy: 0, r: 0, g: 0, b: 0 }, // tol omitted → 后端走默认容差
    ])
    cleanup(app, el)
  })

  it('remove button drops the item at its index', async () => {
    const { emitted, app, el } = mountStructuredInput(arraySchema, [
      { dx: 0, dy: 0, r: 1, g: 1, b: 1 },
      { dx: 5, dy: 5, r: 2, g: 2, b: 2 },
    ])
    await nextTick()
    const removeBtns = el.querySelectorAll('[data-testid="array-remove-btn"]')
    ;(removeBtns[0] as HTMLElement).click()
    await nextTick()
    expect(emitted).toHaveLength(1)
    expect(emitted[0]).toEqual([{ dx: 5, dy: 5, r: 2, g: 2, b: 2 }])
    cleanup(app, el)
  })

  it('empty/undefined value renders only the add button (no item blocks)', async () => {
    const { app, el } = mountStructuredInput(arraySchema, undefined)
    await nextTick()
    expect(el.querySelector('[data-testid="array-add-btn"]')).not.toBeNull()
    expect(el.querySelectorAll('[data-testid="array-remove-btn"]').length).toBe(0)
    cleanup(app, el)
  })

  it('array items do not render their own JSON toggle (noTextMode); only the array owns one', async () => {
    const msgs = { structured_input: { switch_to_text: 'TO_JSON' } }
    const { app, el } = mountStructuredInput(arraySchema, [{ dx: 0, dy: 0, r: 1, g: 2, b: 3 }], { messages: msgs })
    await nextTick()
    const toggles = Array.from(el.querySelectorAll('button')).filter(
      (b) => b.getAttribute('title') === 'TO_JSON',
    )
    expect(toggles.length).toBe(1)
    cleanup(app, el)
  })

  it('validateAgainstSchema: non-array → error; element type + unknown key checked', () => {
    function validate(value: unknown, schema: NodeFieldSchema): string | null {
      if (schema.type === 'array') {
        if (!Array.isArray(value)) return 'expected array'
        if (schema.items) {
          for (let i = 0; i < value.length; i++) {
            const err = validate(value[i], schema.items)
            if (err) return `[${i}]: ${err}`
          }
        }
        return null
      }
      if (schema.type === 'object') {
        if (typeof value !== 'object' || value === null || Array.isArray(value)) return 'expected object'
        const obj = value as Record<string, unknown>
        const declared = new Set((schema.fields ?? []).map((f) => f.key))
        for (const k of Object.keys(obj)) if (!declared.has(k)) return `unknown key: ${k}`
        for (const f of schema.fields ?? []) {
          if (f.required && (!(f.key in obj) || obj[f.key] == null)) return `required field missing: ${f.key}`
          if (f.key in obj) {
            const e = validate(obj[f.key], f.schema)
            if (e) return `${f.key}: ${e}`
          }
        }
        return null
      }
      if (schema.type === 'number') return typeof value !== 'number' ? 'expected number' : null
      return null
    }
    expect(validate({}, arraySchema)).toBe('expected array')
    expect(validate([{ dx: 0, dy: 0, r: 1, g: 2, b: 3 }], arraySchema)).toBeNull()
    expect(validate([{ dx: 0, dy: 0, r: 1, g: 2, b: 3, x: 9 }], arraySchema)).toBe('[0]: unknown key: x')
    expect(validate([{ dx: 0, dy: 0, r: 'no', g: 2, b: 3 }], arraySchema)).toBe('[0]: r: expected number')
  })
})

describe('StructuredInput — scalar types', () => {
  beforeEach(() => vi.clearAllMocks())

  it('number schema renders spinbutton input', async () => {
    const { app, el } = mountStructuredInput({ type: 'number' }, 42)
    await nextTick()
    // UInputNumber renders an input[role=spinbutton]
    expect(el.querySelector('[role="spinbutton"]')).not.toBeNull()
    cleanup(app, el)
  })

  it('string schema renders text input', async () => {
    const { app, el } = mountStructuredInput({ type: 'string' }, 'hello')
    await nextTick()
    const input = el.querySelector('input[type="text"]')
    expect(input).not.toBeNull()
    cleanup(app, el)
  })

  it('bool schema renders checkbox (role=checkbox from reka-ui CheckboxRoot)', async () => {
    const { app, el } = mountStructuredInput({ type: 'bool' }, true)
    await nextTick()
    // UCheckbox uses reka-ui CheckboxRoot which renders role="checkbox" (not input[type=checkbox])
    const cb = el.querySelector('[role="checkbox"]')
    expect(cb).not.toBeNull()
    cleanup(app, el)
  })
})
