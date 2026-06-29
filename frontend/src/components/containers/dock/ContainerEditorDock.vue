<template>
  <!-- 停靠区: 黑底面板挤画布 (不盖). 多根 = aside + 右侧拖宽手柄, 与画布同处编辑器 flex 行. -->
  <aside
    :style="{ width: width + 'px' }"
    class="shrink-0 border-r border-default overflow-hidden flex flex-col bg-default"
  >
    <slot />
  </aside>
  <SplitHandle
    :model-value="width"
    :min="min"
    :max="max"
    @update:model-value="setWidth"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SplitHandle from '@/components/common/SplitHandle.vue'
import { useSplitpane } from '@/composables/useSplitpane'

const props = defineProps<{ wide: boolean }>()

// 窄模式 (节点库/变量/Snippets 列表) 与 宽模式 (资产: 缩略图网格 / 列表) 各自一套宽度,
// 各自持久化、互不挤压: 资产宽态拖宽不会缩窄列表态, 反之亦然.
// 宽态 min 450: 详情已改为按需 modal、不再常占右栏, 整宽全给网格/列表 → 450 足够
// (旧 660 是当时右栏吃 320px 列表只剩 ~280 才挤; 用户 2026-06-15 验).
const narrow = useSplitpane('editor.dock.narrow', { default: 300, min: 240, max: 480 })
const widePane = useSplitpane('editor.dock.wide', { default: 520, min: 450, max: 900 })

const active = computed(() => (props.wide ? widePane : narrow))
const width = computed(() => active.value.width.value)
const min = computed(() => (props.wide ? 450 : 240))
const max = computed(() => (props.wide ? 900 : 480))
function setWidth(v: number) {
  active.value.setWidth(v)
}
</script>
