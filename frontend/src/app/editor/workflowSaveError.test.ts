import { describe, expect, it } from 'vitest'
import { RPCError } from '@/lib/invoke'
import { describeWorkflowSaveError } from './workflowSaveError'

describe('workflow save errors', () => {
  it('turns a raw validation code into an actionable save message', () => {
    const failure = describeWorkflowSaveError(new Error('INVALID_FIELD'))

    expect(failure.kind).toBe('validation')
    expect(failure.message).toContain('节点参数或连线')
    expect(failure.message).not.toBe('INVALID_FIELD')
  })

  it('keeps an unknown internal code as supporting context instead of the whole message', () => {
    const failure = describeWorkflowSaveError(
      new RPCError({ code: 'PATCH_STORAGE_FAILED' }, 'workflow.applyPatch', 'op-1', null),
    )

    expect(failure.kind).toBe('unknown')
    expect(failure.message).toContain('保存时发生内部错误')
    expect(failure.message).toContain('PATCH_STORAGE_FAILED')
  })

  it('identifies revision conflicts without relying on the rendered banner text', () => {
    const failure = describeWorkflowSaveError(new Error('workflow source revision conflict'))

    expect(failure.kind).toBe('revision')
    expect(failure.message).toContain('重新加载')
  })

  it('uses structured patch context to explain and locate the invalid node field', () => {
    const failure = describeWorkflowSaveError(
      new RPCError(
        {
          code: 'INVALID_CONFIG_VALUE',
          category: 'validation',
          message: 'value does not satisfy the node contract',
          details: { commandIndex: 0 },
        },
        'workflow.applyPatch',
        'op-1',
        null,
      ),
      [
        {
          kind: 'set-config',
          setConfig: {
            graphId: 'main',
            nodeId: 'delay',
            fieldId: 'duration-milliseconds',
            value: 5.3,
          },
        },
      ],
    )

    expect(failure.kind).toBe('validation')
    expect(failure.message).toContain('duration-milliseconds')
    expect(failure.message).toContain('定位')
    expect(failure.target).toEqual({
      graphId: 'main',
      nodeId: 'delay',
      fieldId: 'duration-milliseconds',
      portId: undefined,
    })
  })
})
