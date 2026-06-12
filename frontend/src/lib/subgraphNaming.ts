// 「<base> N」序号命名 — 折叠/裸建子图默认名 (主流节点编辑器式递增, 弃时间戳)。
function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function nextSubgraphName(existingLabels: string[], base: string): string {
  const re = new RegExp(`^${escapeRegExp(base)} (\\d+)$`)
  let max = 0
  for (const label of existingLabels) {
    const m = re.exec(label.trim())
    if (m) max = Math.max(max, Number(m[1]))
  }
  return `${base} ${max + 1}`
}
