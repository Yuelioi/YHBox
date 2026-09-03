import { reactive } from 'vue'
import { describe, expect, it } from 'vitest'
import { editableAnnotationUpdate } from './annotationUpdate'

describe('editableAnnotationUpdate', () => {
  it('removes Vue proxies before an annotation enters the editor command boundary', () => {
    const annotation = reactive({
      id: 'note-a',
      text: '',
      position: { x: 10, y: 20 },
      size: { width: 260, height: 140 },
    })

    const update = editableAnnotationUpdate(annotation, { text: 'hello' })

    expect(() => structuredClone(update)).not.toThrow()
    expect(update).toEqual({
      id: 'note-a',
      text: 'hello',
      position: { x: 10, y: 20 },
      size: { width: 260, height: 140 },
    })
  })
})
