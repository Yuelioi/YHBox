// frontend/src/composables/editor/useNodeGroupColor.ts
// Editor v2 B/C polish — shared group color + per-node icon color helpers.
// Used by NodeExplorerModal + InlineContextMenu for consistent visual mental
// model: category label color = dominant node color in that group, item icon
// color = node's own visual.bg-derived hue (matches what user sees on canvas).

import { getSpec } from '@/components/containers/nodeRegistry/registry'
import type { NodeKindSpec } from '@/components/containers/nodeRegistry/index'

/**
 * Map nodeRegistry group → its dominant Tailwind color name (e.g. 'amber', 'blue').
 * Derived from actual visual.bg fields per spec file:
 *   variables: amber (SetVar/GetVar/IncVar/GetSys/GetParam all amber)
 *   purefunc:  amber (math + comparison + logic — all amber)
 *   control:   blue (Loop/If/Switch/Parallel/Race dominant; Start/Stop/Break/Continue jelly mix)
 *   detect:    violet (CheckTemplate/WaitTemplate/Screenshot violet; DetectColor/ROIColorScan cyan; pick violet majority)
 *   input:     orange (KeyPress/ClickAt/KeyHold/MouseHold all orange)
 *   system:    cyan (Try/Throw/Subgraph cyan; OnEvent/PlayClip pink; cyan majority)
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
