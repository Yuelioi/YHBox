import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useExprFusion } from './useExprFusion'

// Minimal graph factory: two Expr nodes A.value → B.<targetPin>, plus A's own data inputs.
function makeGraph() {
  return {
    nodes: [
      {
        id: 'A',
        kind: 'Expr',
        // 表达式存 literal.Expression, 动态输入声明存 config.Inputs[] (PascalCase Name/Type).
        config: {
          Inputs: [{ Name: 'x', Type: 'number' }],
          literal: { Expression: 'x * 2', x: 5 },
        },
      },
      {
        id: 'B',
        kind: 'Expr',
        config: {
          // B already has its own input named `x` → forces A's `x` to be renamed.
          Inputs: [
            { Name: 'in', Type: 'number' },
            { Name: 'x', Type: 'number' },
          ],
          literal: { Expression: 'in + x', in: 99, x: 7 },
        },
      },
    ],
    edges: [{ from: 'A.value', to: 'B.in' }],
  }
}

describe('useExprFusion literal migration', () => {
  it("carries A's renamed-input literal to B and drops the consumed targetPin literal", () => {
    const g = ref(makeGraph())
    const { fuse } = useExprFusion({ activeGraph: g, syncFlowFromDraft: () => {} })

    expect(fuse('A', 'B', 'in')).toBe(true)

    const b = g.value.nodes.find((n: any) => n.id === 'B')! as any
    const inputNames = (b.config.Inputs as any[]).map((i) => i.Name)

    // A's `x` collided with B's `x` → renamed to `x1`; both present.
    expect(inputNames).toEqual(['x', 'x1'])

    // A is gone.
    expect(g.value.nodes.find((n: any) => n.id === 'A')).toBeUndefined()

    // targetPin literal removed; A's literal carried under the renamed key; B's own kept.
    expect('in' in b.config.literal).toBe(false)
    expect(b.config.literal.x1).toBe(5)
    expect(b.config.literal.x).toBe(7)

    // targetPin `in` in B's expression replaced by the parenthesized, renamed A expr.
    expect(b.config.literal.Expression).toBe('(x1 * 2) + x')
  })
})
