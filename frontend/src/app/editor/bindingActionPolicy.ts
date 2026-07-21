export type BindingActionPolicy = {
  resetToDefault: boolean
  clear: boolean
}

export function bindingActionPolicy(input: {
  required: boolean
  hasDefault: boolean
  bound: boolean
}): BindingActionPolicy {
  if (!input.bound) return { resetToDefault: false, clear: false }
  if (input.hasDefault) return { resetToDefault: true, clear: false }
  return { resetToDefault: false, clear: !input.required }
}
