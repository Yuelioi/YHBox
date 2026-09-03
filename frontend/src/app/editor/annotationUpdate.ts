import type { Annotation } from '../../../../contracts/workflow/current/workflow-source'

export function editableAnnotationUpdate(
  annotation: Annotation,
  patch: Partial<Pick<Annotation, 'text' | 'position' | 'size' | 'color'>>,
): Annotation {
  const color = patch.color ?? annotation.color
  return {
    id: annotation.id,
    text: patch.text ?? annotation.text,
    position: { ...(patch.position ?? annotation.position) },
    size: { ...(patch.size ?? annotation.size) },
    ...(color === undefined ? {} : { color }),
  }
}
