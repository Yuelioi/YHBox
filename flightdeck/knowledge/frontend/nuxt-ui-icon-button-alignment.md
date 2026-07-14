---
kind: trap
summary: "Nuxt UI UButton 默认只做交叉轴 items-center；固定宽高的 icon-only 按钮必须依赖全局 justify-center 基线，否则图标贴左。"
activation: symptom
read_when: "新增或修改 icon-only UButton；看到按钮背景正确但图标水平偏左/不居中；给 UButton 强制 size-*、w-*、h-* 或 p-0 前。"
recheck_when: "升级 @nuxt/ui；修改 vite.config.ts 的 ui.button 主题；全局扫描图标按钮对齐时。"
---
# Nuxt UI 固定尺寸图标按钮居中

## 症状与根因

Nuxt UI v4 生成的 Button base 有 `inline-flex items-center`，但没有 `justify-center`。普通按钮宽度由图标、文字和 padding 撑开，看起来不会暴露问题；一旦页面强制 `size-7` / `size-8`、去掉 padding，主轴出现剩余空间，图标按默认 `justify-start` 靠左。

这不是 Iconify 图标自身 viewBox 偏移，也不是单个页面 CSS。给每个按钮临时补 `justify-center` 只能修当前实例，下一处固定尺寸按钮仍会复发。

## 项目级基线

`frontend/vite.config.ts` 的 Nuxt UI 主题必须保留：

```ts
button: {
  slots: { base: 'justify-center' },
}
```

页面规则：

- icon-only 固定尺寸用 `class="size-7 p-0"`，不要为尺寸单独覆盖 `:ui.base`。
- 带文字的普通按钮自然居中。
- 菜单行、导航行等需要左对齐时，显式写 `justify-start`；Tailwind class 冲突合并会覆盖全局 `justify-center`。
- icon-only 按钮必须有 `aria-label`，通常同时给 `title` 提供鼠标提示。

## 防回归与扫描

- `frontend/src/test/iconButtonAlignment.spec.ts` 锁定全局主题基线；删除居中规则会直接红测。
- 扫固定尺寸 Nuxt UI 图标按钮：

```powershell
rg -n "size-[0-9]+ p-0|w-[0-9]+.*p-0|h-[0-9]+.*p-0" frontend/src -g "*.vue"
```

- 扫到局部 `justify-center` 不必机械删除：光学微调或非 UButton 原生控件可以保留；先确认它是否依赖固定宽高。
