import { describe, expect, it } from 'vitest'
import { parseWorkspaceResource, serializeWorkspaceResource } from './resourceDrag'

describe('workspace resource drag payload', () => {
  it('round-trips a bounded asset selection', () => {
    expect(parseWorkspaceResource(serializeWorkspaceResource('template-1'))).toBe('template-1')
  })

  it('rejects unrelated and malformed drop data', () => {
    expect(parseWorkspaceResource('{"kind":"window"}')).toBeNull()
    expect(parseWorkspaceResource('{"guid":"../asset"}')).toBeNull()
    expect(parseWorkspaceResource('not json')).toBeNull()
  })
})
