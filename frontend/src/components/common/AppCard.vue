<!-- 卡片外壳: V4 raised 表面 (顶光 + 高光 + 柔投影)。padding 档可选; hover=true 时悬停加强浮起。
     内容走默认 slot。颜色/文字由内容自身决定, 本组件只给外壳。 -->
<template>
  <div
    class="rounded-xl raised-surface"
    :class="[
      paddingClass,
      hover ? 'transition-shadow duration-150 hover:shadow-[0_6px_20px_rgba(0,0,0,0.45)]' : '',
    ]"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    /** padding 档: panel=p-4 (紧凑面板) / section=p-6 (卡片) / none=无 */
    padding?: 'panel' | 'section' | 'none'
    /** 悬停加强浮起 (列表卡片用) */
    hover?: boolean
  }>(),
  { padding: 'section' },
)

const PAD: Record<NonNullable<typeof props.padding>, string> = {
  panel: 'p-4',
  section: 'p-6',
  none: '',
}
const paddingClass = computed(() => PAD[props.padding])
</script>
