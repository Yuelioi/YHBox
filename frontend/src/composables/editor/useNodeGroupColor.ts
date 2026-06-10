// 共享 group color + per-node icon color helpers.
// 用于 NodeExplorerModal + InlineContextMenu: 分类 label 用该 group 主色, item icon
// 用节点自己 visual.bg 派生色 — 保证跟用户在画布上看到的颜色一致.

import { useI18n } from 'vue-i18n'
import { getSpec } from '@/components/containers/nodeRegistry/registry'
import type { NodeGroup, NodeKindSpec } from '@/components/containers/nodeRegistry/index'
import { groupVisual, resolvePalette } from '@/components/containers/visualRegistry'

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
  'event',
  'random',
  'list',
]

/** Tailwind text color class for a group label/chevron. 从视觉注册中心派生 (单一真源). */
export function groupLabelColor(group: string): string {
  return resolvePalette(groupVisual(group).color).text
}

/**
 * 节点 icon 颜色 = 其 group 主色 (visual 本就按 group 派生, 直接走注册中心, 不再正则抠 class).
 */
export function nodeIconColor(spec: NodeKindSpec | undefined): string {
  if (!spec) return resolvePalette('zinc').text
  return resolvePalette(groupVisual(spec.group ?? 'system').color).text
}

/** Resolve color from a kind string (convenience for templates). */
export function kindIconColor(kind: string): string {
  return nodeIconColor(getSpec(kind))
}

// group label key 映射到 i18n key (nodeGroup.*)
const GROUP_I18N_KEY: Record<string, string> = {
  control: 'nodeGroup.control',
  variables: 'nodeGroup.variables',
  purefunc: 'nodeGroup.purefunc',
  detect: 'nodeGroup.detect',
  input: 'nodeGroup.input',
  system: 'nodeGroup.system',
  io: 'nodeGroup.io',
  stopwatch: 'nodeGroup.stopwatch',
  mock: 'nodeGroup.mock',
  test: 'nodeGroup.test',
  event: 'nodeGroup.event',
  random: 'nodeGroup.random',
  list: 'nodeGroup.list',
  misc: 'nodeGroup.other',
}

export function groupLabelZh(group: string): string {
  const { t } = useI18n()
  const key = GROUP_I18N_KEY[group]
  if (!key) return group
  return t(key)
}
