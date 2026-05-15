<template>
  <div class="px-8 py-6 space-y-6">
    <!-- Enable/Disable + state badge -->
    <div class="flex items-center justify-between gap-4">
      <div
        class="inline-flex items-center gap-2 px-3 py-1.5 rounded-md bg-default border border-default text-xs font-medium"
      >
        <span
          class="size-1.5 rounded-full"
          :class="enabled ? 'bg-primary animate-pulse' : 'bg-accented'"
        ></span>
        <span :class="enabled ? 'text-primary' : 'text-muted'">
          {{ enabled ? `已启用 ${currentMods}+1~6` : '已关闭' }}
        </span>
      </div>

      <UButton v-if="!enabled" color="primary" icon="i-tabler-bolt" @click="onEnable"
        >启用热键</UButton
      >
      <UButton v-else color="error" variant="soft" icon="i-tabler-bolt-off" @click="onDisable"
        >关闭热键</UButton
      >
    </div>

    <!-- Mods selector -->
    <div class="rounded-xl bg-default border border-default p-5">
      <div class="flex items-center justify-between gap-6">
        <div>
          <h3 class="text-sm font-medium text-highlighted">修饰键组合</h3>
          <p class="text-xs text-dimmed mt-1">
            配合数字键 1 ~ 6 切换上阵队伍。建议避开其它应用占用的组合。
          </p>
        </div>
        <USelect
          v-model="modsLabel"
          :items="modOptions"
          class="w-44"
          @update:model-value="onModsChange"
        />
      </div>
    </div>

    <!-- Info card -->
    <div class="rounded-xl bg-default/50 border border-default/60 p-5">
      <div class="flex items-start gap-3">
        <UIcon name="i-tabler-info-circle" class="size-4 text-dimmed mt-0.5 shrink-0" />
        <div class="text-xs text-muted leading-relaxed">
          <p>全局热键。游戏在前台且主界面（无弹窗占用 L 键）时按热键切队，约 3 秒完成。</p>
          <p>开发原因: 小吱刷钱/跑图/粉爪/副本 队伍都不一样, 一直切换太烦人!</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useToast } from '@nuxt/ui/composables'
import { useBattleStore } from '@/stores/battle'

const battleStore = useBattleStore()
const toast = useToast()

const enabled = computed(() => battleStore.status.enabled)
const currentMods = computed(() => battleStore.status.mods)

const modOptions = ['Ctrl', 'Ctrl+Shift', 'Ctrl+Alt', 'Shift+Alt', 'Ctrl+Shift+Alt']
const modsLabel = ref<string>(battleStore.status.mods)

// Sync UI when store changes externally
watch(
  () => battleStore.status.mods,
  (v) => {
    if (v !== modsLabel.value) modsLabel.value = v
  },
)

async function onEnable() {
  const r = await battleStore.enable()
  if (r === undefined) return
  toast.add({
    title: '已启用队伍切换热键',
    color: 'success',
    icon: 'i-tabler-bolt',
  })
}

async function onDisable() {
  const r = await battleStore.disable()
  if (r === undefined) return
  toast.add({
    title: '已关闭队伍切换热键',
    color: 'neutral',
    icon: 'i-tabler-bolt-off',
  })
}

async function onModsChange(v: string) {
  const r = await battleStore.setMods(v)
  if (r === undefined) {
    // rollback UI to server's notion
    modsLabel.value = battleStore.status.mods
    return
  }
  toast.add({
    title: `修饰键已切换为 ${v}`,
    color: 'neutral',
    icon: 'i-tabler-check',
  })
}
</script>
