import { backend, type AssetRecord, type AssetSummary } from '@/lib/backend'
import type { WorkflowResource } from '../../../../contracts/workflow/current/workflow-source'

export async function snapshotGlobalAsset(asset: AssetSummary): Promise<WorkflowResource> {
  return snapshotAsset(asset)
}

export async function snapshotGlobalAssetByID(guid: string): Promise<WorkflowResource> {
  const record = await backend.assets.get(guid)
  if (record.kind !== 'template' && record.kind !== 'macro' && record.kind !== 'clip')
    throw new Error(`Global Asset ${guid} has an unsupported kind`)
  const asset: AssetSummary = {
    guid: record.guid,
    kind: record.kind,
    name: record.name,
    description: record.description,
    category: record.category,
    tags: record.tags,
    variantCount: record.variants?.length ?? 0,
    variants: (record.variants ?? []).map((variant) => ({
      resolution: [Number(variant.resolution[0] ?? 0), Number(variant.resolution[1] ?? 0)],
      blob: { ...variant.blob },
    })),
    blob: record.blob ? { ...record.blob } : undefined,
    createdAt: record.createdAt,
  }
  return snapshotAsset(asset, record)
}

async function snapshotAsset(
  asset: AssetSummary,
  loadedRecord?: AssetRecord,
): Promise<WorkflowResource> {
  const presentation = {
    id: `asset-${asset.guid}`,
    name: asset.name,
    description: asset.description?.trim() || undefined,
    category: asset.category?.trim() || undefined,
    tags: uniqueStrings(asset.tags ?? []),
  }
  if (asset.kind === 'template') {
    const record = loadedRecord ?? (await backend.assets.get(asset.guid))
    const variants = (record.variants ?? []).map((variant, index) => {
      const resolution: [number, number] = [
        Number(variant.resolution[0] ?? 0),
        Number(variant.resolution[1] ?? 0),
      ]
      const bbox: [number, number, number, number] =
        variant.bbox.length === 4
          ? [
              Number(variant.bbox[0]),
              Number(variant.bbox[1]),
              Number(variant.bbox[2]),
              Number(variant.bbox[3]),
            ]
          : [0, 0, resolution[0], resolution[1]]
      return {
        id: `variant-${resolution[0]}x${resolution[1]}-${index + 1}`,
        resolution,
        bbox,
        blob: { ...variant.blob },
      }
    })
    variants.sort((left, right) => left.id.localeCompare(right.id))
    if (!variants.length) throw new Error('Global Asset has no portable image variants')
    return {
      ...presentation,
      kind: 'image',
      image: { variants: variants as [(typeof variants)[0], ...typeof variants] },
    }
  }
  if (asset.kind === 'macro') {
    const macro = await backend.macros.get(asset.guid)
    if (!macro) throw new Error('Global Asset has no portable Macro payload')
    const analysis = await backend.macros.analyze(macro.document)
    return {
      ...presentation,
      kind: 'macro',
      macro: {
        blob: { ...macro.blob },
        baseResolution: [...macro.document.baseResolution],
        actionCount: macro.document.actions.length,
        durationUs: analysis.durationUs,
      },
    }
  }
  const clip = await backend.clips.summary(asset.guid)
  const mouseMode =
    clip.meta.mouseMode === 'relative' || clip.meta.mouseMode === 'absolute'
      ? clip.meta.mouseMode
      : 'mixed'
  return {
    ...presentation,
    kind: 'input-clip',
    inputClip: {
      blob: { ...clip.blob },
      durationUs: clip.durationUs,
      eventCount: clip.eventCount,
      recordingMode: clip.meta.recordingMode,
      mouseMode,
      baseResolution: [...clip.meta.baseResolution],
      mouseCounts360: clip.meta.mouseCounts360,
      stopHotkeyVk: clip.meta.stopHotkeyVK,
    },
  }
}

function uniqueStrings(values: string[]): string[] {
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))].sort()
}
