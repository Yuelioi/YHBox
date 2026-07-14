import { createApp, nextTick } from 'vue'
import { describe, expect, it } from 'vitest'
import CappedPreviewImage from './CappedPreviewImage.vue'

describe('CappedPreviewImage', () => {
  it('keeps source pixels at a 1:1 maximum by default', async () => {
    const el = document.createElement('div')
    document.body.appendChild(el)
    const app = createApp(CappedPreviewImage, { src: 'data:image/png;base64,preview' })

    try {
      app.mount(el)
      const image = el.querySelector('img')!
      Object.defineProperty(image, 'naturalWidth', { configurable: true, value: 80 })
      Object.defineProperty(image, 'naturalHeight', { configurable: true, value: 45 })
      image.dispatchEvent(new Event('load'))
      await nextTick()

      expect(image.style.maxWidth).toBe('80px')
      expect(image.style.maxHeight).toBe('45px')
    } finally {
      app.unmount()
      el.remove()
    }
  })

  it('honors a caller-provided upscale limit', async () => {
    const el = document.createElement('div')
    document.body.appendChild(el)
    const app = createApp(CappedPreviewImage, {
      src: 'data:image/png;base64,preview',
      maxUpscale: 1.5,
    })

    try {
      app.mount(el)
      const image = el.querySelector('img')!
      Object.defineProperty(image, 'naturalWidth', { configurable: true, value: 100 })
      Object.defineProperty(image, 'naturalHeight', { configurable: true, value: 60 })
      image.dispatchEvent(new Event('load'))
      await nextTick()

      expect(image.style.maxWidth).toBe('150px')
      expect(image.style.maxHeight).toBe('90px')
    } finally {
      app.unmount()
      el.remove()
    }
  })
})
