// Unified drag-drop abstraction. VarsPanel / FavoritesPanel / RecentPanel /
// Promote-to-Var (C) all funnel through here. One MIME, one payload schema,
// one modifier-key handling. Spec: editor-v2-vars-panel-design.md §4.6.

import type { VariableRef } from '@/lib/variableRef'

export const MIME = 'application/x-yhfish'

export type EditorDragPayload =
  | { type: 'var'; ref: VariableRef; modifier: 'none' | 'alt' }
  | { type: 'node-spec'; kind: string }
  | { type: 'library-subgraph'; id: string }

/**
 * Call from dragstart handler. Auto-detects Alt modifier for `var` payload.
 * Other payload types ignore modifier.
 */
export function startEditorDrag(payload: EditorDragPayload, e: DragEvent): void {
  if (!e.dataTransfer) return
  const enriched: EditorDragPayload = payload.type === 'var'
    ? { ...payload, modifier: e.altKey ? 'alt' : 'none' }
    : payload
  e.dataTransfer.setData(MIME, JSON.stringify(enriched))
  e.dataTransfer.effectAllowed = 'copy'
}

/**
 * Call from drop / dragover. Returns null if not a YHFish payload (lets
 * existing handlers like vue-flow continue).
 */
export function readDragPayload(e: DragEvent): EditorDragPayload | null {
  if (!e.dataTransfer) return null
  const raw = e.dataTransfer.getData(MIME)
  if (!raw) return null
  try {
    return JSON.parse(raw) as EditorDragPayload
  } catch {
    return null
  }
}
