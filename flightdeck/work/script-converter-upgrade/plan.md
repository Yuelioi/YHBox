# Script Converter Upgrade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让子图转脚本支持常见复杂控制流，优先覆盖 `Expr`、`Loop`、`Break`、`Continue`。

**Architecture:** 继续保持 `frontend/src/lib/subgraphToScript.ts` 为纯函数转换器。转换器先做图边分类和结构校验，再把 exec 链生成 JS；本轮把 `Loop` 从“不可绑定”特例改成结构节点特例，把 `Expr` 从动态输入拒转改成纯表达式内联。

**Tech Stack:** Vue frontend / TypeScript / Vitest / CodeMirror 预览复用现有链路。

---

### Task 1: 固定转换行为

**Files:**
- Modify: `frontend/src/lib/subgraphToScript.test.ts`

- [x] **Step 1: Add failing tests for Expr and Loop conversion**

Add tests that assert:

```ts
it('Expr pure data node inlines its JS expression with wired inputs', () => {
  const r = subgraphToScript(
    makeSg({
      nodes: [
        node('sleep', 'Sleep'),
        node('expr', 'Expr', {
          literal: { Expression: 'base + 1' },
          Inputs: [{ Name: 'base', Type: 'number' }],
        }),
        node('now', 'Now'),
      ],
      edges: [
        { from: 'in.Done', to: 'sleep.In' },
        { from: 'expr.Result', to: 'sleep.Duration' },
        { from: 'now.Ms', to: 'expr.base' },
        { from: 'sleep.Done', to: 'out.In' },
      ],
    }),
    ctx(),
  )
  expect(codeOf(r)).toBe(['Sleep({ Duration: (Now({}) + 1) })', 'return "done"'].join('\n'))
})
```

Add tests for `Loop` count/forever and body control:

```ts
expect(codeOf(r)).toContain('for (let i1 = 0; i1 < 3; i1++) {')
expect(codeOf(r)).toContain('while (true) {')
expect(codeOf(r)).toContain('break')
expect(codeOf(r)).toContain('continue')
```

- [x] **Step 2: Run tests to verify RED**

Run:

```powershell
pnpm --dir frontend test -- subgraphToScript
```

Expected: FAIL because `Expr` still returns `dynamic_inputs` and `Loop` still returns `not_bindable`.

### Task 2: Implement converter support

**Files:**
- Modify: `frontend/src/lib/subgraphToScript.ts`

- [x] **Step 1: Treat structural nodes explicitly**

Keep `Subgraph` / `CollapsedNode` special handling. Add structural handling for `Loop`, `Break`, and `Continue` before generic bindable checks:

- `Loop` is allowed even though it is not script-bindable.
- `Break` and `Continue` are allowed and emit control-transfer statements.
- `dynamicInputs` rejection skips `Expr`.

- [x] **Step 2: Render Expr as JS expression**

For `Expr`, read `literal.Expression` first, then `literal.Expr`, then empty string. Replace bare dynamic input names with rendered upstream values using word-boundary replacement. Wrap the result in parentheses.

- [x] **Step 3: Render Loop regions**

When emitting a `Loop` node:

- Read `Mode` and `Count` from literal/config.
- `count` renders `for (let iN = 0; iN < Count; iN++) { ... }`.
- `forever` renders `while (true) { ... }`.
- Emit body from `Loop.Body`.
- Continue chain after `Loop.Done`.

- [x] **Step 4: Render Break and Continue**

Emit `break` and `continue` as terminal statements for the current chain.

### Task 3: Verification and docs

**Files:**
- Modify: `flightdeck/work/script-converter-upgrade/index.md`
- Modify: `flightdeck/cockpit.md`

- [x] **Step 1: Run focused frontend tests**

```powershell
pnpm --dir frontend test -- subgraphToScript
```

- [x] **Step 2: Run typecheck**

```powershell
pnpm --dir frontend typecheck
```

- [x] **Step 3: Run diff whitespace check**

```powershell
git diff --check
```

- [x] **Step 4: Update Flightdeck status and commit**

Update the work index and cockpit with the implemented scope, then commit tracked changes.
