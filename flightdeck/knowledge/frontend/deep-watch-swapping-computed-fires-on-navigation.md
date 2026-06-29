# ⚠ deep watch 一个会换引用的 computed: 导航换引用也会触发 (误判成编辑)

SUMMARY: deep watch 一个会换引用的 computed, 导航换引用也触发, 会被误判成编辑.
READ WHEN: 写 deep watch 监听一个「会切换指向不同对象」的 computed 前 (activeGraph / 当前选中项 之类); 撞「只是切换/导航、没编辑, 却被标 dirty / 触发了 onChange」

---


## Signature
- symptom: 只是进子图、什么都没改, 编辑器就变「未保存」
- error_type: —  (Vue 响应式误触发)
- where: `useContainerDraft` dirty watcher — `watch(() => activeGraph.value, cb, { deep: true })`
- trigger: `activeGraph` 是 computed, 进子图时它指向的对象换了引用 → deep watch 把"换引用"也当编辑

## 症状/复现

进任意子图(不做任何编辑) → 编辑器状态立刻变「未保存」。出主图不会(回调对 path 空提前 return)。

## 根因

`activeGraph` 是随 `editorPath` 切换指向的 computed。`watch(getter, cb, {deep})` 会因**两件事**触发:
① 当前对象**深层内容被改**(真编辑); ② getter 返回的**对象引用变了**(导航切层级 / 子图被 replace)。
进子图属于 ②, 此时 `path.length>0` → 直接 `dirty=true`, 哪怕零编辑。保存后 `replaceSubgraph` 换引用同样会误标。

## 修法

判别两种触发: **深层 mutate(真编辑)→ `new === old`(同一对象被改); 换引用(导航/replace)→ `new !== old`**。
只在 `new === old` 时标脏:

```js
watch(() => activeGraph.value, (g, prev) => {
  if (path.length === 0) return
  if (g !== prev) return   // 切层级/换引用, 非编辑
  dirty.value = true; touchSubgraph(...)
}, { deep: true })
```

**通用教训**: deep watch 一个可能返回不同对象引用的 computed 时, 必须区分"引用变(导航)"vs"内容变(编辑)",
否则任何切换/replace 都会误触发回调。这是 Vue deep watch 的已知语义(深层 mutate 时 oldVal===newVal)。

## Cases
- 2026-06-13 首次: 进子图就误标未保存。修: cb 加 `if (g !== prev) return` 判别。
