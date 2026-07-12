<!--
BaseModal 共享 modal 外壳 (纯黑底, 内容平铺 — 复刻 BaseModal 之前各 modal 的原版观感)。
- header: [icon] 标题 …… [✕]  (border-b)
- body:   默认 slot, 内容直接平铺纯黑底 (px-5 py-4 统一 padding, 无浅色面板/无内嵌框)
- footer: #footer slot 按钮行 (border-t); 无 slot 不渲染
层次靠内容自身元素 (列表/卡片用 bg-elevated 自己抬升), 外壳只给纯黑底 + 统一结构/留白。
open 走 v-model, 跟 useDialogOpen() 正交。复杂内部布局由调用方塞 slot。
-->
<template>
  <UModal
    :open="open"
    :dismissible="dismissible"
    :ui="{ content: contentClasses }"
    @update:open="(v: boolean) => emit('update:open', v)"
  >
    <template #content>
      <div
        class="flex flex-col bg-default"
        :class="tall ? 'h-[92vh] max-h-[92vh]' : 'max-h-[85vh]'"
      >
        <header class="flex items-center gap-2.5 px-5 py-3.5 border-b border-default shrink-0">
          <UIcon v-if="icon" :name="icon" :class="['size-5 shrink-0', iconColorClass]" />
          <h3 class="text-sm font-medium text-highlighted flex-1 min-w-0 truncate">{{ title }}</h3>
          <slot name="header-extra" />
          <UButton
            v-if="showClose"
            icon="i-tabler-x"
            variant="ghost"
            color="neutral"
            size="xs"
            class="shrink-0 -mr-1"
            :aria-label="t('common.close')"
            @click="emit('update:open', false)"
          />
        </header>

        <div class="flex-1 min-h-0 overflow-y-auto px-5 py-4">
          <slot />
        </div>

        <footer
          v-if="$slots.footer"
          class="flex items-center justify-end gap-2 px-5 py-3.5 border-t border-default shrink-0"
        >
          <slot name="footer" />
        </footer>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

type ModalSize = 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | '5xl' | '6xl' | '7xl'
type IconColor = 'primary' | 'error' | 'warning' | 'success' | 'info'

const props = withDefaults(
  defineProps<{
    open: boolean
    title?: string
    icon?: string
    iconColor?: IconColor
    size?: ModalSize
    showClose?: boolean
    /** 占满高度档 (92vh, 编辑器全屏用); 缺省内容自适应、上限 85vh。 */
    tall?: boolean
    /** 追加到 UModal content 的 class (如全屏宽度覆盖)。 */
    contentClass?: string
    /** false = 不因点外部/Esc 关闭 (只能点 ✕/按钮关) — 代码编辑器 modal 用, 防补全浮层(挂 body, 算"外部")误关 + 防误触丢稿。 */
    dismissible?: boolean
  }>(),
  { title: '', iconColor: 'primary', size: 'md', showClose: true, dismissible: true },
)
const emit = defineEmits<{ 'update:open': [v: boolean] }>()
const { t } = useI18n()

// 字面量映射 (动态 text-${x} 会被 Tailwind purge).
const ICON_COLOR_MAP: Record<IconColor, string> = {
  primary: 'text-primary',
  error: 'text-error',
  warning: 'text-warning',
  success: 'text-success',
  info: 'text-info',
}
const iconColorClass = computed(() => ICON_COLOR_MAP[props.iconColor])

const SIZE_MAP: Record<ModalSize, string> = {
  sm: 'sm:max-w-sm',
  md: 'sm:max-w-md',
  lg: 'sm:max-w-lg',
  xl: 'sm:max-w-xl',
  '2xl': 'sm:max-w-2xl',
  '3xl': 'sm:max-w-3xl',
  '4xl': 'sm:max-w-4xl',
  '5xl': 'sm:max-w-5xl',
  '6xl': 'sm:max-w-6xl',
  '7xl': 'sm:max-w-7xl',
}
const contentClasses = computed(() =>
  [SIZE_MAP[props.size], props.contentClass].filter(Boolean).join(' '),
)
</script>
