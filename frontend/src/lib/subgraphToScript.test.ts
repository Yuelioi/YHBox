import { describe, expect, it } from 'vitest'
import type { NodeKindSpec } from '@/components/containers/nodeRegistry'
import type { GraphEdge, GraphNode, Subgraph } from '@/lib/backend'
import { subgraphToScript, type ConvertCtx } from './subgraphToScript'

// ---- 测试用 spec / 子图 stub ----

function makeSpec(p: {
  kind: string
  execIn?: string[]
  execOut?: string[]
  errorOut?: string[]
  dataIn?: Record<string, any>
  dataOut?: Record<string, any>
  defaults?: Record<string, any>
  isPureData?: boolean
  dynamicInputs?: boolean
}): NodeKindSpec {
  return {
    kind: p.kind,
    group: 'system',
    labelZh: `node.${p.kind}.label`,
    description: `node.${p.kind}.description`,
    example: `node.${p.kind}.example`,
    visual: { icon: '', bg: '', border: '' },
    execIn: p.execIn ?? (p.isPureData ? [] : ['In']),
    execOut: p.execOut ?? (p.isPureData ? [] : ['Done']),
    errorOut: p.errorOut,
    dataIn: p.dataIn ?? {},
    dataOut: p.dataOut ?? {},
    fields: [],
    defaults: p.defaults ?? {},
    isPureData: p.isPureData,
    dynamicInputs: p.dynamicInputs,
  }
}

const SPECS: Record<string, NodeKindSpec> = {
  KeyHoldStart: makeSpec({ kind: 'KeyHoldStart', dataIn: { VK: 'string' } }),
  Sleep: makeSpec({
    kind: 'Sleep',
    dataIn: { Duration: 'number', JitterPct: 'number' },
    defaults: { literal: { JitterPct: 0 } },
  }),
  ClickAt: makeSpec({
    kind: 'ClickAt',
    dataIn: { Point: 'point', XRatio: 'number' },
  }),
  CheckTemplate: makeSpec({
    kind: 'CheckTemplate',
    execOut: ['Found', 'NotFound'],
    dataIn: { Template: 'string', CapFound: 'string' },
    dataOut: { Point: 'point' },
    defaults: { literal: { CapFound: '' } },
  }),
  Now: makeSpec({ kind: 'Now', isPureData: true, dataOut: { Ms: 'number' } }),
  TwoOut: makeSpec({
    kind: 'TwoOut',
    isPureData: true,
    dataOut: { A: 'number', B: 'number' },
  }),
  GetParam: makeSpec({
    kind: 'GetParam',
    isPureData: true,
    dataIn: { ParamName: 'string' },
    dataOut: { Value: 'any' },
  }),
  Expr: makeSpec({
    kind: 'Expr',
    isPureData: true,
    dynamicInputs: true,
    dataIn: { Expression: 'string' },
    dataOut: { Result: 'any' },
  }),
  Loop: makeSpec({ kind: 'Loop', execOut: ['Body', 'Done'] }),
  Break: makeSpec({ kind: 'Break', execOut: [] }),
  Continue: makeSpec({ kind: 'Continue', execOut: [] }),
  ManualOnly: makeSpec({ kind: 'ManualOnly' }),
  DynamicOnly: makeSpec({ kind: 'DynamicOnly', dynamicInputs: true }),
  Subgraph: makeSpec({
    kind: 'Subgraph',
    execOut: [],
    errorOut: ['Fail'],
    dataIn: { SubgraphID: 'string', Params: 'any' },
    dataOut: { Error: 'string', Code: 'string' },
  }),
  CollapsedNode: makeSpec({
    kind: 'CollapsedNode',
    execOut: [],
    errorOut: ['Fail'],
    dataIn: { SubgraphID: 'string', Label: 'string' },
    dataOut: { Error: 'string', Code: 'string' },
    defaults: { literal: { Label: 'Collapsed' } },
  }),
}

// Loop 是 RegionRunner, 不可绑定; 其余(除 marker)都可。
const BINDABLE = new Set(
  Object.keys(SPECS).filter(
    (k) => !['Loop', 'Subgraph', 'CollapsedNode', 'ManualOnly'].includes(k),
  ),
)

function makeSg(p: {
  nodes: GraphNode[]
  edges: GraphEdge[]
  outputPins?: { id: string; name: string; nodeID: string }[]
}): Subgraph {
  return {
    id: 'sg-test',
    rev: 1,
    label: 'T',
    graph: { id: 'g1', schemaVersion: 1, nodes: p.nodes, edges: p.edges },
    entry: { nodeID: 'in' },
    outputPins: p.outputPins ?? [{ id: 'decl-done', name: 'done', nodeID: 'out' }],
    createdAt: '',
  }
}

function ctx(extra?: Partial<ConvertCtx>): ConvertCtx {
  return {
    specFor: (k) => SPECS[k],
    bindable: BINDABLE,
    subgraphsById: new Map(),
    ...extra,
  }
}

function node(id: string, kind: string, config?: Record<string, any>, label?: string): GraphNode {
  return { id, kind, x: 0, y: 0, config, ...(label ? { label } : {}) }
}

function codeOf(r: ReturnType<typeof subgraphToScript>): string {
  if (!r.ok) throw new Error(`unexpected unsupported: ${JSON.stringify(r.unsupported)}`)
  return r.code
}

function reasonsOf(r: ReturnType<typeof subgraphToScript>): string[] {
  if (r.ok) throw new Error('expected unsupported')
  return r.unsupported.map((u) => u.reason)
}

// ---- 成功转换 ----

describe('subgraphToScript: 线性链', () => {
  it('exec 顺序 → 语句顺序, 默认值省略, 出口 → return 出口名', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('n1', 'KeyHoldStart', { literal: { VK: 'W' } }),
          node('n2', 'Sleep', { literal: { Duration: 3000, JitterPct: 0 } }),
        ],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Done', to: 'n2.In' },
          { from: 'n2.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(
      ['KeyHoldStart({ VK: "W" })', 'Sleep({ Duration: 3000 })', 'return "done"'].join('\n'),
    )
  })

  it('用户改过名的节点带行尾注释', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [node('n1', 'KeyHoldStart', { literal: { VK: 'W' } }, '按住前进')],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(['KeyHoldStart({ VK: "W" }) // 按住前进', 'return "done"'].join('\n'))
  })
})

describe('subgraphToScript: 分支与数据', () => {
  it('多出口节点 → if/else if(按 spec 出口顺序), 未接线出口无分支', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('n1', 'CheckTemplate', { literal: { Template: 'g-1' } }),
          node('n2', 'ClickAt', { literal: { XRatio: 0.5 } }),
        ],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Found', to: 'n2.In' },
          { from: 'n2.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(
      [
        'const r1 = CheckTemplate({ Template: "g-1" })',
        'if (r1.exit === Exit.Found) {',
        '  ClickAt({ XRatio: 0.5 })',
        '  return "done"',
        '}',
      ].join('\n'),
    )
  })

  it('双出口都接线 → else if + 多出口 return 各自出口名', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [node('n1', 'CheckTemplate', { literal: { Template: 'g-1' } })],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Found', to: 'out.In' },
          { from: 'n1.NotFound', to: 'out2.In' },
        ],
        outputPins: [
          { id: 'decl-ok', name: 'ok', nodeID: 'out' },
          { id: 'decl-miss', name: 'miss', nodeID: 'out2' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(
      [
        'const r1 = CheckTemplate({ Template: "g-1" })',
        'if (r1.exit === Exit.Found) {',
        '  return "ok"',
        '} else if (r1.exit === Exit.NotFound) {',
        '  return "miss"',
        '}',
      ].join('\n'),
    )
  })

  it('data 边: exec 祖先产出 → rN.字段引用; 纯函数 → 内联', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('n1', 'CheckTemplate', { literal: { Template: 'g-1' } }),
          node('n2', 'ClickAt'),
          node('n3', 'Sleep'),
          node('now1', 'Now'),
        ],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Found', to: 'n2.In' },
          { from: 'n1.Point', to: 'n2.Point' },
          { from: 'n2.Done', to: 'n3.In' },
          { from: 'now1.Ms', to: 'n3.Duration' },
          { from: 'n3.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(
      [
        'const r1 = CheckTemplate({ Template: "g-1" })',
        'if (r1.exit === Exit.Found) {',
        '  ClickAt({ Point: r1.Point })',
        '  Sleep({ Duration: Now({}) })',
        '  return "done"',
        '}',
      ].join('\n'),
    )
  })

  it('GetParam → params.get(名)', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [node('n1', 'Sleep'), node('p1', 'GetParam', { literal: { ParamName: 'ms' } })],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'p1.Value', to: 'n1.Duration' },
          { from: 'n1.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(['Sleep({ Duration: params.get("ms") })', 'return "done"'].join('\n'))
  })

  it('Expr pure data node → inline JS expression with wired dynamic inputs', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('n1', 'Sleep'),
          node('expr1', 'Expr', {
            literal: { Expression: 'base + 1' },
            Inputs: [{ Name: 'base', Type: 'number' }],
          }),
          node('now1', 'Now'),
        ],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'expr1.Result', to: 'n1.Duration' },
          { from: 'now1.Ms', to: 'expr1.base' },
          { from: 'n1.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(['Sleep({ Duration: (Now({}) + 1) })', 'return "done"'].join('\n'))
  })

  it('capture pin 是普通字面量, 透传进参数对象', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [node('n1', 'CheckTemplate', { literal: { Template: 'g-1', CapFound: 'found' } })],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Found', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toContain('CheckTemplate({ Template: "g-1", CapFound: "found" })')
  })

  it('嵌套 Subgraph/CollapsedNode → Subgraph() 调用, 出口 decl ID 映射成 Name', () => {
    // 双出口 callee → 必须按 exit 分支; 单出口的线性化另测。
    const callee: Subgraph = makeSg({ nodes: [], edges: [] })
    callee.id = 'sg-2'
    callee.outputPins = [
      { id: 'decl-x', name: 'finished', nodeID: 'o' },
      { id: 'decl-y', name: 'failed', nodeID: 'o2' },
    ]
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('c1', 'CollapsedNode', { SubgraphID: 'sg-2', literal: { Label: 'Collapsed' } }),
        ],
        edges: [
          { from: 'in.Done', to: 'c1.In' },
          { from: 'c1.decl-x', to: 'out.In' },
        ],
      }),
      ctx({ subgraphsById: new Map([['sg-2', callee]]) }),
    )
    expect(codeOf(r)).toBe(
      [
        'const r1 = Subgraph({ SubgraphID: "sg-2" })',
        'if (r1.exit === "finished") {',
        '  return "done"',
        '}',
      ].join('\n'),
    )
  })

  it('单出口 callee 跑干也返回唯一出口(subgraph_call.go:66) → 线性语句不分支', () => {
    const callee: Subgraph = makeSg({ nodes: [], edges: [] })
    callee.id = 'sg-3'
    callee.outputPins = [{ id: 'decl-x', name: 'finished', nodeID: 'o' }]
    const r = subgraphToScript(
      makeSg({
        nodes: [node('c1', 'Subgraph', { SubgraphID: 'sg-3' })],
        edges: [
          { from: 'in.Done', to: 'c1.In' },
          { from: 'c1.decl-x', to: 'out.In' },
        ],
      }),
      ctx({ subgraphsById: new Map([['sg-3', callee]]) }),
    )
    expect(codeOf(r)).toBe(['Subgraph({ SubgraphID: "sg-3" })', 'return "done"'].join('\n'))
  })
})

// ---- 拒转 ----

describe('subgraphToScript: 拒转', () => {
  it('非 bindable 节点与非 Expr 动态输入节点 → unsupported, 且攒全所有不支持项', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [node('m1', 'ManualOnly'), node('d1', 'DynamicOnly')],
        edges: [
          { from: 'in.Done', to: 'm1.In' },
          { from: 'm1.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    const reasons = reasonsOf(r)
    expect(reasons).toContain('subgraphScript.reason.not_bindable')
    expect(reasons).toContain('subgraphScript.reason.dynamic_inputs')
    expect(reasons.length).toBe(2)
  })

  it('exec 汇合(两分支进同一节点) → duplicated tail in each JS branch', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('n1', 'CheckTemplate', { literal: { Template: 'g' } }),
          node('n2', 'Sleep', { literal: { Duration: 1 } }),
        ],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Found', to: 'n2.In' },
          { from: 'n1.NotFound', to: 'n2.In' },
          { from: 'n2.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(
      [
        'const r1 = CheckTemplate({ Template: "g" })',
        'if (r1.exit === Exit.Found) {',
        '  Sleep({ Duration: 1 })',
        '  return "done"',
        '} else if (r1.exit === Exit.NotFound) {',
        '  Sleep({ Duration: 1 })',
        '  return "done"',
        '}',
      ].join('\n'),
    )
  })

  it('exec 成环 → unsupported', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('n1', 'KeyHoldStart', { literal: { VK: 'W' } }),
          node('n2', 'Sleep', { literal: { Duration: 1 } }),
        ],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Done', to: 'n2.In' },
          { from: 'n2.Done', to: 'n1.In' },
        ],
      }),
      ctx(),
    )
    expect(reasonsOf(r)).toContain('subgraphScript.reason.cycle')
  })

  it('跨分支 data 引用(数据源不是执行祖先) → unsupported', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('n1', 'CheckTemplate', { literal: { Template: 'g' } }),
          node('n2', 'ClickAt'),
          node('n3', 'CheckTemplate', { literal: { Template: 'h' } }),
        ],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Found', to: 'n2.In' },
          { from: 'n1.NotFound', to: 'n3.In' },
          { from: 'n3.Point', to: 'n2.Point' },
          { from: 'n2.Done', to: 'out.In' },
          { from: 'n3.Found', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(reasonsOf(r)).toContain('subgraphScript.reason.cross_branch_data')
  })

  it('Fail 出口接线 → unsupported(提示手写 try/catch)', () => {
    const callee: Subgraph = makeSg({ nodes: [], edges: [] })
    callee.id = 'sg-2'
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('c1', 'Subgraph', { SubgraphID: 'sg-2' }),
          node('n2', 'Sleep', { literal: { Duration: 1 } }),
        ],
        edges: [
          { from: 'in.Done', to: 'c1.In' },
          { from: 'c1.Fail', to: 'n2.In' },
          { from: 'n2.Done', to: 'out.In' },
        ],
      }),
      ctx({ subgraphsById: new Map([['sg-2', callee]]) }),
    )
    expect(reasonsOf(r)).toContain('subgraphScript.reason.fail_edge')
  })

  it('多输出纯函数被消费 → unsupported', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [node('n1', 'Sleep'), node('t1', 'TwoOut')],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 't1.A', to: 'n1.Duration' },
          { from: 'n1.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(reasonsOf(r)).toContain('subgraphScript.reason.multi_out_pure')
  })

  it('disabled 节点 → unsupported(运行时 passthrough 语义转不动)', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [{ ...node('n1', 'KeyHoldStart', { literal: { VK: 'W' } }), disabled: true }],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(reasonsOf(r)).toContain('subgraphScript.reason.disabled')
  })

  it('同一 exec 出口扇出到多个目标 → unsupported', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('n1', 'KeyHoldStart', { literal: { VK: 'W' } }),
          node('n2', 'Sleep', { literal: { Duration: 1 } }),
          node('n3', 'Sleep', { literal: { Duration: 2 } }),
        ],
        edges: [
          { from: 'in.Done', to: 'n1.In' },
          { from: 'n1.Done', to: 'n2.In' },
          { from: 'n1.Done', to: 'n3.In' },
          { from: 'n2.Done', to: 'out.In' },
          { from: 'n3.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(reasonsOf(r)).toContain('subgraphScript.reason.fan_out')
  })

  it('嵌套子图引用的 callee 不存在 → unsupported', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [node('c1', 'Subgraph', { SubgraphID: 'sg-miss' })],
        edges: [{ from: 'in.Done', to: 'c1.In' }],
      }),
      ctx(),
    )
    expect(reasonsOf(r)).toContain('subgraphScript.reason.callee_missing')
  })
})

describe('subgraphToScript: Loop 与控制转移', () => {
  it('Loop count → for loop, Body becomes block and Done continues after the loop', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('loop1', 'Loop', { literal: { Mode: 'count', Count: 3 } }),
          node('sleepBody', 'Sleep', { literal: { Duration: 100 } }),
          node('sleepAfter', 'Sleep', { literal: { Duration: 200 } }),
        ],
        edges: [
          { from: 'in.Done', to: 'loop1.In' },
          { from: 'loop1.Body', to: 'sleepBody.In' },
          { from: 'loop1.Done', to: 'sleepAfter.In' },
          { from: 'sleepAfter.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(
      [
        'for (let i1 = 0; i1 < 3; i1++) {',
        '  Sleep({ Duration: 100 })',
        '}',
        'Sleep({ Duration: 200 })',
        'return "done"',
      ].join('\n'),
    )
  })

  it('Loop forever → while true', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('loop1', 'Loop', { literal: { Mode: 'forever' } }),
          node('sleepBody', 'Sleep', { literal: { Duration: 100 } }),
        ],
        edges: [
          { from: 'in.Done', to: 'loop1.In' },
          { from: 'loop1.Body', to: 'sleepBody.In' },
          { from: 'loop1.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(
      ['while (true) {', '  Sleep({ Duration: 100 })', '}', 'return "done"'].join('\n'),
    )
  })

  it('Break and Continue nodes become JS control statements', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('loop1', 'Loop', { literal: { Mode: 'count', Count: 3 } }),
          node('check1', 'CheckTemplate', { literal: { Template: 'g-1' } }),
          node('break1', 'Break'),
          node('continue1', 'Continue'),
        ],
        edges: [
          { from: 'in.Done', to: 'loop1.In' },
          { from: 'loop1.Body', to: 'check1.In' },
          { from: 'check1.Found', to: 'break1.In' },
          { from: 'check1.NotFound', to: 'continue1.In' },
          { from: 'loop1.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(
      [
        'for (let i1 = 0; i1 < 3; i1++) {',
        '  const r2 = CheckTemplate({ Template: "g-1" })',
        '  if (r2.exit === Exit.Found) {',
        '    break',
        '  } else if (r2.exit === Exit.NotFound) {',
        '    continue',
        '  }',
        '}',
        'return "done"',
      ].join('\n'),
    )
  })

  it('pure data reused inside and after Loop stays in valid JS scope', () => {
    const r = subgraphToScript(
      makeSg({
        nodes: [
          node('loop1', 'Loop', { literal: { Mode: 'count', Count: 3 } }),
          node('sleepBody', 'Sleep'),
          node('sleepAfter', 'Sleep'),
          node('now1', 'Now'),
        ],
        edges: [
          { from: 'in.Done', to: 'loop1.In' },
          { from: 'loop1.Body', to: 'sleepBody.In' },
          { from: 'now1.Ms', to: 'sleepBody.Duration' },
          { from: 'loop1.Done', to: 'sleepAfter.In' },
          { from: 'now1.Ms', to: 'sleepAfter.Duration' },
          { from: 'sleepAfter.Done', to: 'out.In' },
        ],
      }),
      ctx(),
    )
    expect(codeOf(r)).toBe(
      [
        'for (let i1 = 0; i1 < 3; i1++) {',
        '  Sleep({ Duration: Now({}) })',
        '}',
        'Sleep({ Duration: Now({}) })',
        'return "done"',
      ].join('\n'),
    )
  })
})
