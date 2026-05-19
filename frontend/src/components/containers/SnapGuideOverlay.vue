<!-- frontend/src/components/containers/SnapGuideOverlay.vue -->
<!-- Editor v2 C — drag-time alignment guides (PS smart guides).
     Spec: editor-v2-quick-actions-design.md §9.
     Renders inside the canvas wrapper (position:relative) as a pointer-events-none SVG overlay.
     Coordinates are in flow-canvas space; the parent wraps VueFlow which handles pan/zoom internally.
     NOTE: Because VueFlow transforms its inner viewport (not the wrapper), this SVG is rendered in
     screen pixel space (the wrapper div). To stay aligned across pan/zoom we use a large fixed
     viewBox and position lines in screen pixels (converted from flow coords via useVueFlow().flowToScreenCoordinate). -->
<template>
  <svg
    v-if="guides.length > 0"
    class="absolute inset-0 pointer-events-none"
    style="z-index: 5;"
    :width="width"
    :height="height"
    :viewBox="`0 0 ${width} ${height}`"
  >
    <line
      v-for="(g, idx) in guides"
      :key="idx"
      :x1="g.x1"
      :y1="g.y1"
      :x2="g.x2"
      :y2="g.y2"
      :stroke="g.axis === 'y' ? '#06b6d4' : '#d946ef'"
      stroke-width="1"
      stroke-dasharray="4 2"
      opacity="0.85"
    />
  </svg>
</template>

<script setup lang="ts">
export interface SnapGuide {
  axis: 'x' | 'y'
  x1: number
  y1: number
  x2: number
  y2: number
}

defineProps<{
  guides: SnapGuide[]
  width: number
  height: number
}>()
</script>
