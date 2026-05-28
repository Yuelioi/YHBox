// 共享 group color + per-node icon color helpers.
// 用于 NodeExplorerModal + InlineContextMenu: 分类 label 用该 group 主色, item icon
// 用节点自己 visual.bg 派生色 — 保证跟用户在画布上看到的颜色一致.

import { useI18n } from 'vue-i18n'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import type { NodeGroup, NodeKindSpec } from '@/components/containers/nodeRegistry/index'

/**
 * 10 个 canonical NodeGroup, 跟 nodeRegistry/index.ts NodeGroup 类型同步.
 * 各模态框 (NodeExplorerModal / InlineContextMenu) 共享, 不再各处 hardcode.
 */
export const ALL_NODE_GROUPS: NodeGroup[] = [
  'control',
  'variables',
  'purefunc',
  'detect',
  'input',
  'system',
  'io',
  'stopwatch',
  'mock',
  'test',
]

/**
 * group → 该 group 主色 (Tailwind tone). 由该 group 各 spec.visual.bg 取众数派生.
 * key 'misc' 是 fallback string, 不在 NodeGroup 类型里 (那 10 个都有真色).
 */
const GROUP_TONE: Record<string, string> = {
  control: 'blue',
  variables: 'amber',
  purefunc: 'amber',
  detect: 'violet',
  input: 'orange',
  system: 'cyan',
  io: 'emerald',
  stopwatch: 'rose',
  mock: 'zinc',
  test: 'zinc',
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

// group label key 映射到 i18n key (nodeGroup.* + 一个 system_subgraph)
const GROUP_I18N_KEY: Record<string, string> = {
  control: 'nodeGroup.control',
  variables: 'nodeGroup.variables',
  purefunc: 'nodeGroup.purefunc',
  detect: 'nodeGroup.detect',
  input: 'nodeGroup.input',
  system: 'nodeGroup.system_subgraph',
  io: 'nodeGroup.io',
  stopwatch: 'nodeGroup.stopwatch',
  mock: 'nodeGroup.mock',
  test: 'nodeGroup.test',
  misc: 'nodeGroup.other',
}

export function groupLabelZh(group: string): string {
  const { t } = useI18n()
  const key = GROUP_I18N_KEY[group]
  if (!key) return group
  return t(key)
}
