// frontend/src/composables/editor/useAssetPicker.spec.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('useAssetPicker', () => {
  beforeEach(() => {
    vi.resetModules()
  })

  it('初始 request 为 null', async () => {
    const { useAssetPicker } = await import('./useAssetPicker')
    expect(useAssetPicker().request.value).toBeNull()
  })

  it('requestTemplatePick 写入 pin + selected 副本', async () => {
    const { useAssetPicker } = await import('./useAssetPicker')
    const { request, requestTemplatePick } = useAssetPicker()
    const src = ['a', 'b']
    requestTemplatePick('Templates', src)
    expect(request.value).toEqual({ pin: 'Templates', selected: ['a', 'b'] })
    src.push('c') // 副本: 不被外部 mutate 影响
    expect(request.value!.selected).toEqual(['a', 'b'])
  })

  it('updateSelection 仅在有 request 时改 selected', async () => {
    const { useAssetPicker } = await import('./useAssetPicker')
    const { request, requestTemplatePick, updateSelection } = useAssetPicker()
    updateSelection(['x']) // 无 request → no-op
    expect(request.value).toBeNull()
    requestTemplatePick('Templates', [])
    updateSelection(['x', 'y'])
    expect(request.value).toEqual({ pin: 'Templates', selected: ['x', 'y'] })
  })

  it('cancel 清空 request', async () => {
    const { useAssetPicker } = await import('./useAssetPicker')
    const { request, requestTemplatePick, cancel } = useAssetPicker()
    requestTemplatePick('Templates', ['a'])
    cancel()
    expect(request.value).toBeNull()
  })
})
