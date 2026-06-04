import { describe, it, expect } from 'vitest'
import { estimateNodeSize } from './elkGraph'

describe('estimateNodeSize', () => {
  it('未知 kind 回退默认 220x90', () => {
    expect(estimateNodeSize('Nope', {})).toEqual({ width: 220, height: 90 })
  })
  it('CommentBox 用 cfg 宽度', () => {
    expect(estimateNodeSize('CommentBox', { width: 600 }).width).toBe(600)
  })
  it('Switch 高度随 cases 数增长', () => {
    const few = estimateNodeSize('Switch', { cases: ['a'] }).height
    const many = estimateNodeSize('Switch', { cases: ['a', 'b', 'c', 'd'] }).height
    expect(many).toBeGreaterThan(few)
  })
})
