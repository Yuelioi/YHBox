<template>
  <div class="p-6 space-y-6 max-w-2xl">
    <header class="space-y-2">
      <h2 class="text-base font-medium text-highlighted">输入校准</h2>
      <p class="text-xs text-dimmed">
        鼠标硬件 DPI 影响"相对位移"类录制（视角转动）的跨电脑回放。 录制子图时会把本机 360° counts 写入
        <span class="text-toned">RecordingContext</span> 作为源；回放时按 target/source 比例缩放 MouseMoveRel。
      </p>
      <div class="rounded-md bg-elevated/30 border border-default/50 px-3 py-2 text-[11px] text-muted leading-relaxed">
        <UIcon name="i-tabler-info-circle" class="size-3.5 inline-block align-middle mr-1 text-amber-300/80" />
        <span class="text-toned">这里改的是什么</span>：本机值的"默认源"。改它影响：
        <ul class="list-disc pl-5 mt-1 space-y-0.5">
          <li>下次新建的 MouseCalibration 节点用这个值当默认</li>
          <li>"同步本机值到所有容器"按钮 + 节点 Inspector "FOREIGN" 警告里的"同步"按钮，把这个值写到所有容器</li>
        </ul>
        <span class="text-toned mt-1 block">这里改它<span class="text-warning">不会</span>自动改</span>已有容器里的 MouseCalibration 节点 — 它们各自持值（容器自包含）。要批量更新点上面"同步本机值到所有容器"，或进入容器手动改。
      </div>
    </header>

    <!-- 录制配置 -->
    <section class="rounded-md border border-default bg-elevated/40 p-4 space-y-3">
      <header>
        <h3 class="text-sm font-medium text-highlighted">录制配置</h3>
        <p class="text-[11px] text-dimmed mt-0.5">改动需重启 YHBox 生效 (启动期注入).</p>
      </header>

      <!-- 停录热键 -->
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <div class="text-sm text-default">停录热键</div>
          <div class="text-[11px] text-dimmed">
            游戏前台按下停止录制 (LL hook 拦截, 不透传游戏). 默认 F12.
          </div>
        </div>
        <div class="w-56 shrink-0">
          <HotkeyCaptureInput
            :model-value="settings?.ui.recordingStopHotkey ?? 'F12'"
            @update:model-value="(v: string) => patchRecord({ recordingStopHotkey: v })"
          />
        </div>
      </div>

      <!-- 鼠标语义 -->
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <div class="text-sm text-default">鼠标语义</div>
          <div class="text-[11px] text-dimmed leading-snug">
            relative (FPS): 录 RawDelta 给相机转向. absolute (UI/Slate): 录 screen px MouseMove 给
            click/hover.
          </div>
        </div>
        <USelect
          :model-value="settings?.ui.recordingMouseMode ?? 'relative'"
          :items="[
            { label: '相对 (FPS 相机)', value: 'relative' },
            { label: '绝对 (UI 点击)', value: 'absolute' },
          ]"
          class="w-56"
          @update:model-value="(v: string) => patchRecord({ recordingMouseMode: v })"
        />
      </div>
    </section>

    <section class="rounded-md border border-default bg-elevated/40 p-4 space-y-4">
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <h3 class="text-sm font-medium text-highlighted">本机 360° HID counts</h3>
          <p class="text-[11px] text-dimmed mt-0.5">原地转身 360° 鼠标硬件上报的累积 |dx|</p>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <UInputNumber
            v-model="manualCounts"
            :min="0"
            :max="999999"
            :step="100"
            class="w-32"
            @blur="onCommitManual"
          />
          <UButton
            size="xs"
            variant="ghost"
            color="neutral"
            icon="i-tabler-check"
            title="保存手填值"
            @click="onCommitManual"
          />
        </div>
      </div>

      <div class="flex items-center gap-2 flex-wrap">
        <UButton size="sm" color="primary" icon="i-tabler-target" @click="calibratorOpen = true">
          {{ (settings?.ui.mouseCounts360 ?? 0) > 0 ? '重新校准' : '开始校准' }}
        </UButton>
        <UButton
          size="sm"
          variant="soft"
          color="neutral"
          icon="i-tabler-pointer"
          @click="openMouseHUD"
        >
          打开鼠标 HUD
        </UButton>
        <UButton
          v-if="(settings?.ui.mouseCounts360 ?? 0) > 0"
          size="sm"
          variant="soft"
          color="primary"
          icon="i-tabler-refresh"
          @click="onSyncAll"
        >
          同步本机值到所有容器
        </UButton>
        <span class="ml-auto text-[11px] text-dimmed">
          也可以从其他电脑分享脚本附带的 counts，直接手填
        </span>
      </div>
    </section>

    <!-- 说明 -->
    <section
      class="rounded-md border border-default/60 bg-default/50 p-4 text-xs text-dimmed space-y-2"
    >
      <h4 class="text-xs uppercase tracking-wider text-toned">怎么用</h4>
      <ol class="list-decimal pl-5 space-y-1">
        <li>点「开始校准」打开对话框</li>
        <li>切到游戏，对准固定参照物，准备好</li>
        <li>
          按 <code class="bg-elevated/60 px-1 rounded text-toned">F8</code> 开始 3
          秒倒计时（不用回到本程序！）
        </li>
        <li>倒计时结束后开始累计 → 原地匀速转一整圈 360°</li>
        <li>转完再按一次 <code class="bg-elevated/60 px-1 rounded text-toned">F8</code> 停止</li>
        <li>切回程序点「保存」即可</li>
      </ol>
    </section>

    <!-- 校准 Modal -->
    <CalibratorModal v-model:open="calibratorOpen" @save="onCalibratorSaved" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { backend } from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import CalibratorModal from '@/components/calibration/CalibratorModal.vue'
import HotkeyCaptureInput from '@/components/hotkeys/HotkeyCaptureInput.vue'
import { useToast } from '@nuxt/ui/composables'
import { useConfirm } from '@/composables/useConfirm'

const { confirm } = useConfirm()

const settingsStore = useSettingsStore()
const settings = computed(() => settingsStore.data)
const toast = useToast()

const manualCounts = ref<number>(0)
watch(
  () => settings.value?.ui.mouseCounts360,
  (v) => {
    manualCounts.value = v ?? 0
  },
  { immediate: true },
)

async function onCommitManual() {
  const v = Number(manualCounts.value)
  if (!Number.isFinite(v) || v < 0) return
  const cur = settings.value?.ui.mouseCounts360 ?? 0
  if (v === cur) return
  await settingsStore.patch({ ui: { mouseCounts360: Math.floor(v) } })
}

// 录制配置 patch helper. settingsStore.patch 是 deep-merge 语义, 传 partial 即可.
async function patchRecord(patch: Record<string, any>) {
  if (!settings.value) return
  await settingsStore.patch({ ui: patch })
}

async function openMouseHUD() {
  await backend.tools.openMouseHUD()
}

// Plan B Task E.7: 独立"同步本机值到所有容器"按钮（不必先开校准 modal）
async function onSyncAll() {
  const cur = settings.value?.ui.mouseCounts360 ?? 0
  if (cur <= 0) {
    toast.add({ title: '本机 counts360 未设置', color: 'warning' })
    return
  }
  const yes = await confirm({
    title: '同步到所有容器？',
    description: `当前本机 counts360 = ${cur}。\n同步会覆盖所有本地容器主图 MouseCalibration 节点的值。`,
    confirmText: '同步',
    color: 'primary',
  })
  if (yes !== true) return
  const r = (await backend.containers.syncLocalMouseCalibration(cur)) as any
  if (r) {
    toast.add({
      title: `已同步 ${r.updated?.length ?? 0} 个容器`,
      description: r.skipped?.length ? `跳过 ${r.skipped.length} 个（无 MouseCalibration 节点）` : undefined,
      color: 'success',
    })
  }
}

const calibratorOpen = ref(false)

async function onCalibratorSaved(counts: number) {
  await settingsStore.patch({ ui: { mouseCounts360: counts } })
  const yes = await confirm({
    title: '同步到所有容器？',
    description: `新值：${counts}\n是否一键同步到所有本地容器？\n（推荐：替换所有容器主图 MouseCalibration 节点的 counts360）`,
    confirmText: '同步',
    cancelText: '不同步',
    color: 'primary',
  })
  if (yes === true) {
    const r = await backend.containers.syncLocalMouseCalibration(counts)
    if (r) {
      toast.add({
        title: `已同步 ${(r as any).updated?.length ?? 0} 个容器`,
        color: 'success',
      })
    }
  }
}
</script>
