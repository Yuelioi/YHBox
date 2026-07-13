<template>
  <!-- 宽工作区中参与布局；空间不足时覆盖画布，避免与 Inspector 一起把画布挤没。 -->
  <div
    class="dock-shell"
    :class="{ 'dock-shell--overlay': overlay }"
    :data-layout="overlay ? 'overlay' : 'docked'"
  >
    <aside
      :style="{ width: width + 'px' }"
      class="dock-panel border-r border-default overflow-hidden flex flex-col bg-default"
    >
      <slot />
    </aside>
    <SplitHandle
      v-if="!overlay"
      :model-value="width"
      :min="min"
      :max="max"
      @update:model-value="setWidth"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SplitHandle from '@/components/common/SplitHandle.vue'
import { useSplitpane } from '@/composables/useSplitpane'

const { wide, overlay = false } = defineProps<{ wide: boolean; overlay?: boolean }>()

// 窄模式 (节点库/变量/Snippets 列表) 与 宽模式 (资产: 缩略图网格 / 列表) 各自一套宽度,
// 各自持久化、互不挤压: 资产宽态拖宽不会缩窄列表态, 反之亦然.
// 宽态 min 450: 详情已改为按需 modal、不再常占右栏, 整宽全给网格/列表 → 450 足够
// (旧 660 是当时右栏吃 320px 列表只剩 ~280 才挤; 用户 2026-06-15 验).
const narrow = useSplitpane('editor.dock.narrow', { default: 300, min: 240, max: 480 })
const widePane = useSplitpane('editor.dock.wide', { default: 520, min: 450, max: 900 })

const active = computed(() => (wide ? widePane : narrow))
const width = computed(() => active.value.width.value)
const min = computed(() => (wide ? 450 : 240))
const max = computed(() => (wide ? 900 : 480))
function setWidth(v: number) {
  active.value.setWidth(v)
}
</script>

<style scoped>
.dock-shell {
  display: flex;
  min-height: 0;
  flex: none;
}

.dock-panel {
  min-height: 0;
  flex: none;
}

.dock-shell--overlay {
  position: absolute;
  inset-block: 0;
  left: 44px;
  z-index: 35;
  max-width: calc(100% - 76px);
  box-shadow: 12px 0 32px color-mix(in oklch, var(--ui-bg) 72%, transparent);
}

.dock-shell--overlay .dock-panel {
  max-width: 100%;
}
</style>
