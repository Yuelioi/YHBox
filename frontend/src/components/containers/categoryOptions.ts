export function uniqueCategoryOptions(
  ...groups: Array<Iterable<string | undefined | null>>
): string[] {
  const categories = new Set<string>()
  for (const group of groups) {
    for (const item of group) {
      const category = item?.trim()
      if (category) categories.add(category)
    }
  }
  return [...categories]
}

export function addCreatedCategory(
  existing: string[],
  item: string,
): { categories: string[]; value: string } {
  const category = item.trim()
  if (!category) return { categories: existing, value: '' }
  if (existing.includes(category)) return { categories: existing, value: category }
  return { categories: [...existing, category], value: category }
}
