export const EDITOR_SPACIOUS_WIDTH = 1500
export const EDITOR_COMPACT_WIDTH = 1040

export type EditorWorkspaceLayout = 'spacious' | 'focused' | 'compact'

/**
 * Content-driven editor layout. A zero width is the pre-measurement state, so
 * keep the stable spacious layout until ResizeObserver reports a real value.
 */
export function resolveEditorWorkspaceLayout(width: number): EditorWorkspaceLayout {
  if (!Number.isFinite(width) || width <= 0 || width >= EDITOR_SPACIOUS_WIDTH) return 'spacious'
  if (width >= EDITOR_COMPACT_WIDTH) return 'focused'
  return 'compact'
}
