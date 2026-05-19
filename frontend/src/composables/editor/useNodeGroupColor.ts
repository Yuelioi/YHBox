// 共享 group color + per-node icon color helpers.
// 用于 NodeExplorerModal + InlineContextMenu: 分类 label 用该 group 主色, item icon
// 用节点自己 visual.bg 派生色 — 保证跟用户在画布上看到的颜色一致.

import { getSpec } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec } from '@/components/containers/nodeRegistry/index'

/**
 * group → 该 group 主色 (Tailwind tone). 由该 group 各 spec.visual.bg 取众数派生:
 *   variables: amber (SetVar/GetVar/IncVar/GetSys/GetParam)
 *   purefunc:  amber (math + comparison + logic)
 *   control:   blue
 *   detect:    violet
 *   input:     orange
 *   system:    cyan
 *   misc:      zinc (fallback)
 */
const GROUP_TONE: Record<string, string> = {
  control: 'blue',
  variables: 'amber',
  purefunc: 'amber',
  detect: 'violet',
  input: 'orange',
  system: 'cyan',
  misc: 'zinc',
}

/** Tailwind text color class for a group label/chevron. */
export function groupLabelColor(group: string): string {
  const tone = GROUP_TONE[group] ?? GROUP_TONE.misc
  return `text-${tone}-400`
}

/**
 * Extract a Tailwind color name (e.g., 'amber') from a node spec's visual.bg.
 * Expects format like 'bg-amber-500/15'. Falls back to group dominant if parse fails.
 */
export function nodeIconColor(spec: NodeKindSpec | undefined): string {
  if (!spec) return 'text-zinc-400'
  const bg = spec.visual?.bg ?? ''
  // Match 'bg-<tone>-<shade>/<opacity>' OR 'bg-<tone>-<shade>'
  const m = bg.match(/^bg-([a-z]+)-\d+/)
  if (m) return `text-${m[1]}-400`
  return groupLabelColor(spec.group ?? 'misc')
}

/** Resolve color from a kind string (convenience for templates). */
export function kindIconColor(kind: string): string {
  return nodeIconColor(getSpec(kind))
}

export const GROUP_LABELS_ZH: Record<string, string> = {
  control: '控制流',
  variables: '变量',
  purefunc: '运算',
  detect: '检测',
  input: '输入',
  system: '系统/子图',
  misc: '其它',
}

export function groupLabelZh(group: string): string {
  return GROUP_LABELS_ZH[group] ?? group
}
