---
name: snippet-drawer-debug-discipline
description: 撞 bug 别假设 + 改 + 看用户验. 加 console.log 把 setup props 打出来 比改 3 轮代码再 reload 快得多. 这次违反 CLAUDE.md 头号铁律 — 在脑补上反复 fix
when_to_read: 撞 frontend bug 反复 fix 不好 / 想 reload 第 N 次让用户试 / 改了又改还在猜 root cause
applies_to: [debug-discipline, frontend, vue, methodology]
last_updated: 2026-05-26
status: active
---

# 撞 bug 先 dig source/log, 别反复 fix 让用户验

## 教训

撞 frontend bug 反复 fix 不好 = **没找 root cause**, 一直在猜. 停手. 加 1 行 `console.log` 把关键状态打出来 + 让用户贴 log, 比再改 3 轮重新 build 让用户 reload 试快 10x.

CLAUDE.md 头号铁律: "撞 bug 先验自己脑补". 这次违反.

## 为什么

每轮 "改 → build → 用户 reload → 用户试 → 用户报还不行" 的开销:
- 我: ~5 分钟改 + build (5.27s × 多次错版本)
- 用户: 切窗口 / 操作 / 截图 / 回复 — 上下文切换巨大
- 信息密度低: 用户只能说 "还不行", 我从这句话推不出 root cause

而打 log 一轮:
- 我: 加 1 行 log + build
- 用户: 同样 reload + 操作 + 贴 console
- 信息密度高: log 直接告诉我**实际状态**, root cause 立现

3 轮 fail-fix = 15 分钟 + 用户疲劳. 1 轮 log = 5 分钟 + 直接定位.

## 怎么 apply

撞 frontend bug:
1. **第 1 次 fix 之前**就开始想 log. 不是 fail 后再加 — fail 前加.
2. 撞 "状态没传到 / value 永远 X / event 没触发", 第 1 反应: 把那个值 / 那个 prop / 那个 event handler 在两端各 log 一次. 不是写 "修复"
3. 第 1 次 fix 失败 → 立刻停 fix → 加 log → 让用户跑一次再判断. 不接着改第 2 个猜测.
4. log 看到了再判断要不要改 + 改什么. **没 log 不动手第 2 次 fix.**

## 反模式

- ❌ "我觉得是 reactive 不 sync, 改成 ref" → build → 让用户验 → 还不行 → "那应该是 mount 时序, 加 v-if" → build → 让用户验 → ...
- ✅ "几个 fix 都没 work — log setup 时 props 是啥" → 1 行 console.log → 用户贴 → 立刻看到缺 `editingID` → 5 秒定位 = prop 大小写

## Case 1 — 2026-05-26 SaveSnippetDrawer 编辑 prefill 永远空

用户右键 → 编辑 snippet, drawer 打开 form 全空. 我猜了 3 轮:

| 轮 | 假设 | fix | 验证结果 |
|---|---|---|---|
| 1 | reactive form + 直接赋值不 sync NuxtUI v-model | 换 7 个独立 ref | 还不行 |
| 2 | mount 时机 store 没 load | 加 v-if 等 storeLoaded | 还不行 |
| 3 | drawer 复用不 remount | 加 :key 强制 remount | 还不行 |

用户终于受不了 ("我感觉你这部分应该写的有问题"), 我才加 console.log. 用户 1 次贴出 setup props (只含 `open/sourceKind/sourceConfig`, 缺 `editingID` 完全).

立刻定位 = Vue prop 大小写 [[vue-prop-camelize-asymmetry]] (`editingID` 定义, `:editing-id` 转 `editingId` 不匹配, prop undefined). 5 秒诊断, 1 行 replace_all 修.

**如果第 1 轮 fail 后就 log**: 节省 2 轮 fix + 用户 2 次操作 + 我 2 次 build. **如果用户报 bug 第一时间就 log**: 节省全部 3 轮.

教训嵌套: 头号铁律强 sanction "撞 bug 先验脑补", 我下意识把它理解成 "verify against 源码" — 但同样适用 "verify against runtime 状态". log = runtime 版的"读源码".
