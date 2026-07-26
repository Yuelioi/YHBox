# ⚠ Vue prop 名禁连续大写 — kebab/camel 转换不对称
## 教训

Vue prop 名**禁连续大写字母**. 用 `editingId` / `userUrl` / `nodeHtml` (单大写), **不要** `editingID` / `userURL` / `nodeHTML` (连续大写). 否则 `:editing-id="..."` 这种正常 template 写法 camelize 后得不到 prop 名, prop 值永远 undefined.

## 为什么

Vue 内部转换不是 invertible:
- `camelize('editing-id')` → 把每个 `-X` 替换成 `X.toUpperCase()` → `editingId` (单大写 d)
- `hyphenate('editingID')` → 在大写字母前插 `-` 并转小写 → `editing-i-d` (注意中间出现的 `i-d`!)

定义 prop `editingID`, template 写 `:editing-id="..."`:
- Vue 把 attr `editing-id` camelize → `editingId`
- 在 props 里找 `editingId` → 找不到 (定义的是 `editingID`)
- prop 拿不到值 → 永远 undefined
- 静默, 无 warning (因为 `editingId` 在 fallthrough attrs 里看也找不到 prop 声明, 但 attr 还在 $attrs 上)

若 template 强行写 `:editingID="..."` (camelCase 直传, 没 hyphen), Vue 在 DOM template 里 case-fold 成 `editingid` → 还是不匹配. **camelCase prop 名只在 single-file component 的 `<script>` 内 ref 时安全, template 传值必死.**

## 怎么 apply

写 prop 前问: 名字含连续大写吗? 有则改单大写.
- ❌ `editingID` `userURL` `nodeHTML` `apiURL` `videoID`
- ✅ `editingId` `userUrl` `nodeHtml` `apiUrl` `videoId`

**用 TS 帮自己**: 把所有 prop 名串过一遍 lower-then-camelize, 看转回来跟原名一致吗:
```ts
const camelize = (s: string) => s.replace(/-(\w)/g, (_, c) => c.toUpperCase())
const hyphenate = (s: string) => s.replace(/([A-Z])/g, '-$1').toLowerCase()
camelize(hyphenate('editingID')) === 'editingID' // false → 'editingId'
camelize(hyphenate('editingId')) === 'editingId' // true ✓
```

不一致 = 这个名字必坑.

## 撞 bug 时的诊断 checklist

子组件 prop 永远 undefined, parent 明明传了:
1. 名字含连续大写? → 99% 是这个
2. console.log setup props: `JSON.stringify(props)` 看键名 (注意 undefined 被 JSON.stringify 吞, 用 `Object.keys(props)`)
3. parent template attr 写 kebab 还是 camel? 改一种试

## Case 1 — 2026-05-26 SaveSnippetDrawer 编辑 prefill 永远空

[snippet-drawer-debug-discipline.md](snippet-drawer-debug-discipline.md) 多次 fix 都没 work, 最终 console.log 看到 setup props 只含 `{open, sourceKind, sourceConfig}` — 缺 `editingID`. 因为我 prop 定义 `editingID`, template 写 `:editing-id="..."`, camelize → `editingId` (单 d) 跟 `editingID` 不匹配, prop undefined → `getById(undefined)` → `s = undefined` → prefill skip → form 全空.

Fix `d49a8e5`: 全 rename `editingID` → `editingId` (replace_all 2 个文件). template `:editing-id` camelize 后正好匹配, prop 拿到值, prefill 正常.

**前置 3 次失败 fix** 都是在猜:
- 改 reactive form + Object.assign (无效, 因为根本进不到 prefill 分支)
- 拆成 7 个独立 ref (无效, 同上)
- 加 v-if + :key 强 remount (无效, 同上)

每次都白改 build + 让用户验. 真正 root cause 是 prop 名. 教训见 [snippet-drawer-debug-discipline.md](snippet-drawer-debug-discipline.md).
