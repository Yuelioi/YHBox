import { describe, expect, it } from 'vitest'
import {
  EDITOR_COMPACT_WIDTH,
  EDITOR_SPACIOUS_WIDTH,
  resolveEditorWorkspaceLayout,
} from './workspaceLayout'

describe('resolveEditorWorkspaceLayout', () => {
  it('keeps the stable spacious layout before the workspace is measured', () => {
    expect(resolveEditorWorkspaceLayout(0)).toBe('spacious')
  })

  it('overlays the dock when simultaneous panels would starve the canvas', () => {
    expect(resolveEditorWorkspaceLayout(EDITOR_SPACIOUS_WIDTH - 1)).toBe('focused')
    expect(resolveEditorWorkspaceLayout(1366)).toBe('focused')
  })

  it('reserves compact mode for genuinely constrained workspaces', () => {
    expect(resolveEditorWorkspaceLayout(EDITOR_COMPACT_WIDTH - 1)).toBe('compact')
    expect(resolveEditorWorkspaceLayout(EDITOR_SPACIOUS_WIDTH)).toBe('spacious')
  })
})
