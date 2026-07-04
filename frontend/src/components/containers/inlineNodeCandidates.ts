import type { NodeKindSpec, PinType } from './nodeRegistry'
import { pinTypeCompat } from './nodeRegistry'

export interface InlineNodeCandidateContext {
  side: 'input' | 'output'
  pinType?: PinType
  isExec?: boolean
}

export function filterInlineNodeCandidates(
  specs: NodeKindSpec[],
  ctx?: InlineNodeCandidateContext,
): NodeKindSpec[] {
  const eligible = specs.filter((s) => !s.excludeFromPalette)
  if (!ctx) return eligible

  if (ctx.isExec) {
    return eligible.filter((s) => {
      if (s.isPureData || s.isVisualOnly) return false
      return ctx.side === 'output' ? s.execIn.length > 0 : s.execOut.length > 0
    })
  }

  if (!ctx.pinType) return eligible
  if (ctx.side === 'output') {
    return eligible.filter((s) =>
      Object.values(s.dataIn ?? {}).some((to) => isCandidateCompatible(ctx.pinType!, to)),
    )
  }

  return eligible.filter((s) =>
    Object.values(s.dataOut ?? {}).some((from) => isCandidateCompatible(from, ctx.pinType!)),
  )
}

function isCandidateCompatible(from: PinType, to: PinType): boolean {
  const compat = pinTypeCompat(from, to)
  return compat.allow && !compat.warn
}
