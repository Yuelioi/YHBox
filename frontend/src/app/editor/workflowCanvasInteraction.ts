export const WORKFLOW_CANVAS_INTERACTION = {
  selectionKeyCode: true,
  multiSelectionKeyCode: 'Control',
  panActivationKeyCode: 'Space',
  panOnDrag: [0, 1] as number[],
  selectNodesOnDrag: true,
}

export function mergeMarqueeSelection(
  selectionBeforeDrag: ReadonlySet<string>,
  marqueeSelection: ReadonlySet<string>,
): Set<string> {
  return new Set([...selectionBeforeDrag, ...marqueeSelection])
}
