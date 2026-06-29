import { describe, expect, it } from 'vitest'
import { nextSubgraphName } from './subgraphNaming'

describe('nextSubgraphName', () => {
  it('空库从 1 起', () => {
    expect(nextSubgraphName([], '子图')).toBe('子图 1')
  })
  it('取最大序号 +1, 中间空洞不补', () => {
    expect(nextSubgraphName(['子图 1', '子图 5', '上钩处理'], '子图')).toBe('子图 6')
  })
  it('只匹配「base 数字」整名, 相似前缀不算', () => {
    expect(nextSubgraphName(['子图 3x', '子图abc', '我的子图 9'], '子图')).toBe('子图 1')
  })
  it('base 含正则特殊字符不炸', () => {
    expect(nextSubgraphName(['s.g (1) 2'], 's.g (1)')).toBe('s.g (1) 3')
  })
})
