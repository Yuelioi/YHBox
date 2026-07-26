export type CollapseSelectionErrorReason =
  | 'invalid'
  | 'incoming_error'
  | 'multiple_entry'
  | 'input_type'
  | 'output_type'
  | 'missing_boundary'
  | 'unknown'

const reasons = new Map<string, CollapseSelectionErrorReason>([
  ['selection is empty', 'invalid'],
  ['only executable nodes and graph calls can be collapsed', 'invalid'],
  ['collapse selection is invalid', 'invalid'],
  ['selection has an incoming error route', 'incoming_error'],
  ['selection has multiple execution entries', 'multiple_entry'],
  ['selection input type is unavailable', 'input_type'],
  ['selection output type is unavailable', 'output_type'],
  ['selection needs one execution entry and at least one signal exit', 'missing_boundary'],
  ['selection must have one execution entry and at least one signal exit', 'missing_boundary'],
])

export function collapseSelectionErrorReason(error: unknown): CollapseSelectionErrorReason {
  const message = error instanceof Error ? error.message : String(error)
  return reasons.get(message) ?? 'unknown'
}
