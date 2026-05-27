// Container Editor — random ID helpers. backlog C3.
//
// 集中所有 graph node 临时 ID 生成. 各调用点之前散在 4+ 文件各搓一套
// `Math.random().toString(36).slice(2, N)`. 这里收口.

/** 通用随机短 ID — prefix + '_' + base36 后缀 (默认 6 字符). */
export function randID(prefix: string, len: number = 6): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 2 + len)}`
}

/** 按 kind 命名节点 ID — kind 小写做前缀. 'Subgraph' → 'subgraph_abc123'. */
export function newNodeID(kind: string): string {
  return randID(kind.toLowerCase())
}

/** 通用节点 ID — 不关心 kind 时用 (8 字符后缀, 跟老 genID 一致). */
export function genNodeID(): string {
  return randID('n', 8)
}
