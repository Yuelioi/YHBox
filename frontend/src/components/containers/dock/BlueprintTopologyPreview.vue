<template>
  <svg
    viewBox="0 0 240 112"
    class="h-full w-full"
    role="img"
    :aria-label="t('assetBrowser.blueprintPreview', { n: graph.nodes.length })"
  >
    <defs>
      <pattern :id="patternId" width="12" height="12" patternUnits="userSpaceOnUse">
        <circle cx="1" cy="1" r="0.65" fill="var(--ui-border)" opacity="0.45" />
      </pattern>
    </defs>
    <rect width="240" height="112" :fill="`url(#${patternId})`" />
    <g v-if="layoutNodes.length">
      <line
        v-for="(edge, index) in layoutEdges"
        :key="index"
        :x1="edge.from.x"
        :y1="edge.from.y"
        :x2="edge.to.x"
        :y2="edge.to.y"
        stroke="var(--ui-border-accented)"
        stroke-width="1.25"
        opacity="0.72"
      />
      <g v-for="node in layoutNodes" :key="node.id">
        <rect
          :x="node.x - 14"
          :y="node.y - 7"
          width="28"
          height="14"
          rx="3"
          fill="var(--ui-bg-elevated)"
          stroke="var(--ui-border-accented)"
        />
        <rect
          :x="node.x - 14"
          :y="node.y - 7"
          width="28"
          height="3"
          rx="2"
          fill="var(--ui-primary)"
          opacity="0.72"
        />
      </g>
    </g>
    <g v-else>
      <rect x="89" y="45" width="62" height="22" rx="5" fill="var(--ui-bg-elevated)" />
      <text x="120" y="60" text-anchor="middle" fill="var(--ui-text-dimmed)" font-size="9">
        {{ t('assetBrowser.emptyBlueprint') }}
      </text>
    </g>
  </svg>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Graph } from '@/lib/backend'

const { graph } = defineProps<{ graph: Graph }>()
const { t } = useI18n()
const patternId = `blueprint-grid-${useId().replaceAll(':', '')}`

const layoutNodes = computed(() => {
  const nodes = graph.nodes.slice(0, 24)
  if (!nodes.length) return []
  const xs = nodes.map((node) => node.x ?? 0)
  const ys = nodes.map((node) => node.y ?? 0)
  const minX = Math.min(...xs)
  const maxX = Math.max(...xs)
  const minY = Math.min(...ys)
  const maxY = Math.max(...ys)
  const spanX = Math.max(1, maxX - minX)
  const spanY = Math.max(1, maxY - minY)
  return nodes.map((node, index) => ({
    id: node.id,
    x: maxX === minX ? 32 + (index % 6) * 34 : 24 + ((node.x - minX) / spanX) * 192,
    y: maxY === minY ? 28 + Math.floor(index / 6) * 24 : 18 + ((node.y - minY) / spanY) * 76,
  }))
})

const layoutEdges = computed(() => {
  const byId = new Map(layoutNodes.value.map((node) => [node.id, node]))
  return graph.edges
    .slice(0, 36)
    .map((edge) => ({
      from: byId.get(edge.from.split('.')[0]!),
      to: byId.get(edge.to.split('.')[0]!),
    }))
    .filter(
      (
        edge,
      ): edge is {
        from: (typeof layoutNodes.value)[number]
        to: (typeof layoutNodes.value)[number]
      } => !!edge.from && !!edge.to,
    )
})
</script>
