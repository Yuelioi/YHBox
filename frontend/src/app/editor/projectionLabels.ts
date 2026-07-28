interface ProjectionLabelSource {
  id: string
  titleKey?: string
}

type Translate = (key: string) => string
type HasTranslation = (key: string) => boolean

export function projectionLabel(
  source: ProjectionLabelSource,
  t: Translate,
  te: HasTranslation,
): string {
  if (source.titleKey && te(source.titleKey)) return t(source.titleKey)

  const commonKey = `workflow.node.port.${source.id}`
  return te(commonKey) ? t(commonKey) : source.id
}

export function projectionLabelTitle(label: string, id: string): string {
  return label === id ? id : `${label} · ${id}`
}
