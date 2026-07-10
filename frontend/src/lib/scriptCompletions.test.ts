// scriptCompletions.test.ts — 节点函数补全签名推导 (非 exec pin 进签名, exec pin 排除)
// + 语法快速反馈与 $变量引用提取的纯函数。
import { describe, it, expect } from 'vitest'
import type { Spec } from '@bindings/github.com/yottaapp/yotta/internal/node'
import {
  nodeFnCompletions,
  scriptExitItemsForKind,
  scriptDollarRefs,
  scriptSyntaxErrors,
  SUGAR_COMPLETIONS,
  scriptTemplateItemsForPin,
  scriptTemplateInsertMode,
  scriptTemplateInsertText,
  scriptAssetItemsForPin,
  scriptPinValueInsertText,
  scriptPointInsertText,
  scriptGeometryInsertText,
  scriptColorPickTargetForPin,
  scriptColorInsertText,
  scriptAIConnectionItemsForPin,
  scriptAsyncDropdownTargetForPin,
  scriptCurrentCallInputSnapshot,
} from './scriptCompletions'

function fakeSpec(
  kind: string,
  inputs: { name: string; type: string }[],
  outputs: { name: string; type: string }[] = [],
): Spec {
  return { kind, inputs, outputs } as unknown as Spec
}

describe('nodeFnCompletions', () => {
  const specs = new Map<string, Spec>([
    [
      'ClickAt',
      fakeSpec('ClickAt', [
        { name: 'In', type: 'Exec' },
        { name: 'XRatio', type: 'Number' },
        { name: 'YRatio', type: 'Number' },
      ]),
    ],
    ['Random', fakeSpec('Random', [{ name: 'Min', type: 'Number' }])],
  ])

  it('detail 含非 exec pin, 不含 exec pin In', () => {
    const items = nodeFnCompletions(['ClickAt'], specs)
    expect(items).toHaveLength(1)
    expect(items[0].label).toBe('ClickAt')
    expect(items[0].detail).toContain('XRatio')
    expect(items[0].detail).toContain('YRatio')
    expect(items[0].detail).not.toContain('In,')
    expect(items[0].detail).toBe('ClickAt({XRatio, YRatio})')
  })

  it('labelOf 提供时拼进 detail', () => {
    const items = nodeFnCompletions(['Random'], specs, () => '随机数')
    expect(items[0].detail).toBe('Random({Min}) · 随机数')
  })

  it('specs 缺 kind 时签名为空括号, 不崩', () => {
    const items = nodeFnCompletions(['Unknown'], specs)
    expect(items[0].detail).toBe('Unknown({})')
  })

  it('apply 是回调 (光标定位用), 不是纯字符串', () => {
    const items = nodeFnCompletions(['ClickAt'], specs)
    expect(typeof items[0].apply).toBe('function')
  })
})

describe('SUGAR_COMPLETIONS', () => {
  it('糖函数全集齐且都是 function 类型', () => {
    const labels = SUGAR_COMPLETIONS.map((c) => c.label)
    expect(labels).toEqual(
      expect.arrayContaining(['params.get', 'sleep', 'log.info', 'Exit.Found']),
    )
    // 变量读写不走糖 (用 $hp / GetVar/SetVar 节点函数), vars.* 已删。
    expect(labels).not.toContain('vars.get')
    expect(SUGAR_COMPLETIONS.every((c) => c.type === 'function' || c.type === 'constant')).toBe(
      true,
    )
  })
})

describe('scriptExitItemsForKind', () => {
  it('uses node Spec exec outputs and maps standard exits to Exit constants', () => {
    const specs = new Map<string, Spec>([
      [
        'CheckTemplate',
        fakeSpec(
          'CheckTemplate',
          [],
          [
            { name: 'Found', type: 'Exec' },
            { name: 'NotFound', type: 'Exec' },
            { name: 'Point', type: 'Point' },
          ],
        ),
      ],
    ])

    expect(scriptExitItemsForKind('CheckTemplate', specs).map((i) => i.insert)).toEqual([
      'Exit.Found',
      'Exit.NotFound',
    ])
  })

  it('falls back to string literals for non-standard exit names', () => {
    const specs = new Map<string, Spec>([
      ['CustomNode', fakeSpec('CustomNode', [], [{ name: 'Skipped', type: 'Exec' }])],
    ])

    expect(scriptExitItemsForKind('CustomNode', specs).map((i) => i.insert)).toEqual(['"Skipped"'])
  })
})

describe('scriptTemplateItemsForPin', () => {
  it('uses TemplateGUID semantic/widget pins and displays asset names while inserting GUIDs', () => {
    const specs = new Map<string, Spec>([
      [
        'ClickTemplate',
        {
          kind: 'ClickTemplate',
          inputs: [
            {
              name: 'Templates',
              type: 'String',
              semantic: 'TemplateGUID',
              widget: { kind: 'template-picker' },
            },
          ],
        } as unknown as Spec,
      ],
    ])

    const items = scriptTemplateItemsForPin('ClickTemplate', 'Templates', specs, [
      {
        guid: 'tpl-guid-1',
        kind: 'template',
        name: '开始按钮',
        category: 'fishing',
        tags: ['start', 'ui'],
        variantCount: 2,
      },
    ])

    expect(items).toEqual([
      expect.objectContaining({
        label: '开始按钮',
        value: 'tpl-guid-1',
        detail: 'fishing · start, ui · tpl-guid-1',
        type: 'enum',
      }),
    ])
  })

  it('returns no items for ordinary string pins', () => {
    const specs = new Map<string, Spec>([
      [
        'Log',
        {
          kind: 'Log',
          inputs: [{ name: 'Message', type: 'String' }],
        } as unknown as Spec,
      ],
    ])

    expect(scriptTemplateItemsForPin('Log', 'Message', specs, [])).toEqual([])
  })
})

describe('scriptTemplateInsertText', () => {
  it('detects template insert mode from the current document shape', () => {
    const bare = 'ClickTemplate({Templates: })'
    expect(scriptTemplateInsertMode(bare, bare.indexOf('})'))).toBe('bare')

    const array = 'ClickTemplate({Templates: []})'
    expect(scriptTemplateInsertMode(array, array.indexOf(']'))).toBe('array')

    const string = 'ClickTemplate({Templates: "tpl"})'
    expect(scriptTemplateInsertMode(string, string.indexOf('tpl') + 2)).toBe('string')
  })

  it('inserts arrays in bare TemplateGUID value positions', () => {
    expect(scriptTemplateInsertText('tpl-guid-1', 'bare')).toBe('["tpl-guid-1"]')
  })

  it('inserts string literals inside arrays and raw ids inside strings', () => {
    expect(scriptTemplateInsertText('tpl-guid-1', 'array')).toBe('"tpl-guid-1"')
    expect(scriptTemplateInsertText('tpl-guid-1', 'string')).toBe('tpl-guid-1')
  })
})

describe('scriptAssetItemsForPin', () => {
  it('uses ClipID semantic pins and inserts clip ids as a single string literal', () => {
    const specs = new Map<string, Spec>([
      [
        'PlayClip',
        {
          kind: 'PlayClip',
          inputs: [
            {
              name: 'ClipID',
              type: 'String',
              semantic: 'ClipID',
            },
          ],
        } as unknown as Spec,
      ],
    ])

    const items = scriptAssetItemsForPin('PlayClip', 'ClipID', specs, [
      {
        guid: 'clip-guid-1',
        kind: 'clip',
        name: '开局连招',
        category: 'fishing',
        tags: ['bait'],
        variantCount: 0,
      },
      {
        guid: 'tpl-guid-1',
        kind: 'template',
        name: '按钮模板',
        variantCount: 1,
      },
    ])

    expect(items).toEqual([
      expect.objectContaining({
        label: '开局连招',
        value: 'clip-guid-1',
        detail: 'fishing · bait · clip-guid-1',
        insertMode: 'string',
      }),
    ])
    expect(scriptPinValueInsertText(items[0], 'bare', false)).toBe('"clip-guid-1"')
  })
})

describe('scriptAIConnectionItemsForPin', () => {
  it('uses ai-connection widget pins and inserts connection ids as strings', () => {
    const specs = new Map<string, Spec>([
      [
        'AI',
        {
          kind: 'AI',
          inputs: [
            {
              name: 'Connection',
              type: 'String',
              widget: { kind: 'ai-connection' },
            },
          ],
        } as unknown as Spec,
      ],
    ])

    const items = scriptAIConnectionItemsForPin('AI', 'Connection', specs, [
      {
        id: 'openai-main',
        label: 'OpenAI 主连接',
        protocol: 'openai',
        baseURL: 'https://api.openai.com/v1',
      },
    ])

    expect(items).toEqual([
      expect.objectContaining({
        label: 'OpenAI 主连接',
        value: 'openai-main',
        detail: 'openai · https://api.openai.com/v1 · openai-main',
        insertMode: 'string',
      }),
    ])
    expect(scriptPinValueInsertText(items[0], 'bare', false)).toBe('"openai-main"')
  })
})

describe('scriptAsyncDropdownTargetForPin', () => {
  it('detects async-dropdown pins and exposes their async source', () => {
    const specs = new Map<string, Spec>([
      [
        'AndroidTarget',
        {
          kind: 'AndroidTarget',
          inputs: [
            {
              name: 'Serial',
              type: 'String',
              widget: { kind: 'async-dropdown', props: { asyncSource: 'androidADBDevices' } },
            },
          ],
        } as unknown as Spec,
      ],
    ])

    expect(scriptAsyncDropdownTargetForPin('AndroidTarget', 'Serial', specs)).toEqual({
      asyncSource: 'androidADBDevices',
    })
  })

  it('returns null for pins without async dropdown metadata', () => {
    const specs = new Map<string, Spec>([
      [
        'Log',
        {
          kind: 'Log',
          inputs: [{ name: 'Message', type: 'String' }],
        } as unknown as Spec,
      ],
    ])

    expect(scriptAsyncDropdownTargetForPin('Log', 'Message', specs)).toBeNull()
  })
})

describe('scriptCurrentCallInputSnapshot', () => {
  it('extracts simple sibling literal inputs from the current node call', () => {
    const doc = 'AndroidStartApp({Serial: "emulator-5554", Retry: 2, Force: true, Package: })'
    expect(scriptCurrentCallInputSnapshot(doc, doc.indexOf('})'))).toEqual({
      Serial: 'emulator-5554',
      Retry: 2,
      Force: true,
    })
  })

  it('ignores incomplete and complex inputs', () => {
    const doc = 'AndroidStartApp({Serial: params.get("serial"), Package: })'
    expect(scriptCurrentCallInputSnapshot(doc, doc.indexOf('})'))).toEqual({})
  })
})

describe('script screen picker insert text', () => {
  it('formats point picker payload as a Point object literal', () => {
    expect(scriptPointInsertText({ xRatio: 0.123456, yRatio: 0.987654 })).toBe(
      '{ x: 0.1235, y: 0.9877 }',
    )
  })

  it('formats rect picker payload as a Geometry object literal', () => {
    expect(scriptGeometryInsertText({ region: [0.1, 0.2, 0.3, 0.4] })).toBe(
      '{ pct: { x: 0.1, y: 0.2, w: 0.3, h: 0.4 } }',
    )
  })
})

describe('script color picker helpers', () => {
  const specs = new Map<string, Spec>([
    [
      'DetectColor',
      {
        kind: 'DetectColor',
        inputs: [
          { name: 'Mode', type: 'String' },
          {
            name: 'Range',
            type: 'JSON',
            schema: { type: 'tuple', widget: 'colorRange' },
          },
        ],
      } as unknown as Spec,
    ],
    [
      'DetectColorHSV',
      {
        kind: 'DetectColorHSV',
        inputs: [
          {
            name: 'HSV',
            type: 'JSON',
            schema: { type: 'object', widget: 'colorRange' },
          },
        ],
      } as unknown as Spec,
    ],
  ])

  it('detects tuple color range pins and reads rgb mode from the current call', () => {
    const doc = 'DetectColor({ Mode: "rgb", Range: })'
    const target = scriptColorPickTargetForPin(
      'DetectColor',
      'Range',
      specs,
      doc,
      doc.indexOf('})'),
    )
    expect(target).toEqual({ colorSpace: 'rgb', shape: 'tuple' })
  })

  it('detects HSV object color pins', () => {
    expect(scriptColorPickTargetForPin('DetectColorHSV', 'HSV', specs, '', 0)).toEqual({
      colorSpace: 'hsv',
      shape: 'object',
    })
  })

  it('formats color picker payload as tuple or HSV object script literals', () => {
    const payload = { range: [10, 20, 30, 40, 50, 60], hueWrap: false }
    expect(scriptColorInsertText(payload, 'tuple')).toBe('[10, 20, 30, 40, 50, 60]')
    expect(scriptColorInsertText(payload, 'object')).toBe(
      '{ hMin: 10, hMax: 20, sMin: 30, sMax: 40, vMin: 50, vMax: 60 }',
    )
  })
})

describe('scriptSyntaxErrors', () => {
  it('合法脚本无诊断', () => {
    expect(scriptSyntaxErrors('let a = 1\nif (a > 0) {\n  sleep(100)\n}\n')).toEqual([])
  })

  it('空文档无诊断', () => {
    expect(scriptSyntaxErrors('')).toEqual([])
  })

  it('括号不闭合报错且行号对', () => {
    const errs = scriptSyntaxErrors('let a = 1\nif (a > 0 {\n}')
    expect(errs.length).toBeGreaterThan(0)
    expect(errs[0].line).toBe(2)
  })

  it('同一行级联错误只报一次', () => {
    const errs = scriptSyntaxErrors('if ((( {')
    const lines = errs.map((e) => e.line)
    expect(new Set(lines).size).toBe(lines.length)
  })
})

describe('scriptDollarRefs', () => {
  it('提取 $引用, 字符串与注释里的 $ 不命中', () => {
    const { refs } = scriptDollarRefs('let a = $hp\nlog.info("$fake")\n// $comment\n')
    expect(refs.map((r) => r.name)).toEqual(['hp'])
  })

  it('本地 let $x 定义记入 defined, 不算外部引用', () => {
    const { refs, defined } = scriptDollarRefs('let $x = 1\nlog.info($x + $hp)')
    expect(defined.has('x')).toBe(true)
    expect(refs.map((r) => r.name).sort()).toEqual(['hp', 'x'])
  })

  it('裸 $ 不命中', () => {
    const { refs } = scriptDollarRefs('let $ = 1')
    expect(refs).toEqual([])
  })
})
