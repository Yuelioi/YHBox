<template>
  <div class="settings-page">
    <div class="flex items-start gap-3 rounded-xl border border-error/30 bg-error/5 p-4">
      <UIcon
        name="i-tabler-device-desktop-cog"
        class="mt-0.5 size-5 shrink-0 text-error"
        aria-hidden="true"
      />
      <div class="min-w-0">
        <p class="text-sm font-medium text-default">{{ t('settingsAutomation.security.title') }}</p>
        <p class="mt-1 text-xs leading-relaxed text-dimmed">
          {{ t('settingsAutomation.security.hint') }}
        </p>
      </div>
    </div>

    <SettingsSection
      :title="t('settingsAutomation.targets.title')"
      :description="t('settingsAutomation.targets.hint')"
      icon="i-tabler-pointer-cog"
    >
      <template #badge>
        <UBadge size="xs" color="neutral" variant="subtle">{{ draft.length }}</UBadge>
      </template>
      <template #actions>
        <div class="flex flex-wrap gap-2">
          <UButton
            v-if="hasMissingConsent"
            size="sm"
            color="warning"
            variant="soft"
            icon="i-tabler-shield-check"
            :loading="bulkConsentBusy"
            @click="grantAllConsents"
          >
            {{ t('settingsAutomation.bulk.grant') }}
          </UButton>
          <UButton
            v-if="hasGrantedConsent"
            size="sm"
            color="neutral"
            variant="soft"
            icon="i-tabler-shield-off"
            :loading="bulkConsentBusy"
            @click="revokeAllConsents"
          >
            {{ t('settingsAutomation.bulk.revoke') }}
          </UButton>
          <UButton
            size="sm"
            color="primary"
            variant="soft"
            icon="i-tabler-brand-windows"
            :disabled="!desktopTargetType"
            @click="addTarget('desktop-window')"
          >
            {{ t('settingsAutomation.targets.add_windows') }}
          </UButton>
          <UButton
            size="sm"
            color="primary"
            variant="soft"
            icon="i-tabler-brand-android"
            :disabled="!androidTargetType"
            @click="addTarget('android-device')"
          >
            {{ t('settingsAutomation.targets.add_android') }}
          </UButton>
          <UButton
            size="sm"
            color="primary"
            variant="soft"
            icon="i-tabler-brand-chrome"
            :disabled="!browserTargetType"
            @click="addTarget('browser-page')"
          >
            {{ t('settingsAutomation.targets.add_browser') }}
          </UButton>
        </div>
      </template>

      <div
        v-if="captureFeedback"
        class="flex flex-wrap items-center gap-3 rounded-lg border px-3 py-2 text-xs"
        :class="
          captureFeedback.tone === 'success'
            ? 'border-success/30 bg-success/10 text-success'
            : captureFeedback.tone === 'warning'
              ? 'border-warning/30 bg-warning/10 text-warning'
              : 'border-error/30 bg-error/10 text-error'
        "
        :role="captureFeedback.tone === 'error' ? 'alert' : 'status'"
      >
        <span class="min-w-0 flex-1">{{ captureFeedback.message }}</span>
        <UButton
          v-if="captureFeedback.elevationSuggested"
          size="xs"
          color="warning"
          variant="soft"
          icon="i-tabler-shield-lock"
          @click="restartElevated"
        >
          {{ t('settingsAutomation.capture.restart_elevated') }}
        </UButton>
      </div>

      <div v-if="draft.length" class="space-y-3">
        <article v-for="(target, index) in draft" :key="target.slot" class="ai-profile">
          <button
            type="button"
            class="flex min-h-11 min-w-0 flex-1 cursor-pointer items-center gap-3 text-left"
            :aria-expanded="expandedSlot === target.slot"
            :aria-controls="`automation-target-${target.slot}`"
            @click="toggleExpanded(target.slot)"
          >
            <span
              class="flex size-9 shrink-0 items-center justify-center rounded-lg border border-default bg-elevated"
            >
              <UIcon name="i-tabler-pointer-cog" class="size-4 text-toned" aria-hidden="true" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-2">
                <span class="truncate text-sm font-medium text-default">{{ target.label }}</span>
                <UBadge
                  :color="target.workflowConsent ? 'success' : 'warning'"
                  size="xs"
                  variant="subtle"
                  :icon="
                    target.workflowConsent ? 'i-tabler-shield-check' : 'i-tabler-shield-exclamation'
                  "
                >
                  {{
                    t(
                      target.workflowConsent
                        ? 'settingsAutomation.targets.workflow_allowed'
                        : 'settingsAutomation.targets.consent_required',
                    )
                  }}
                </UBadge>
                <UBadge v-if="isDesktop(target)" size="xs" color="neutral" variant="subtle">
                  {{ t(`settingsAutomation.backend.${target.inputBackend}`) }}
                </UBadge>
                <UBadge v-if="isDesktop(target)" size="xs" color="neutral" variant="subtle">
                  {{ t(`settingsAutomation.captureBackend.${target.captureBackend}`) }}
                </UBadge>
                <UBadge size="xs" color="neutral" variant="outline">
                  {{ target.targetKind }} · {{ target.adapterKind }}
                </UBadge>
              </span>
              <span class="mt-1 block truncate text-xs text-dimmed">
                {{ targetSummary(target) }} · <code>{{ target.slot }}</code>
              </span>
            </span>
            <UIcon
              name="i-tabler-chevron-down"
              class="size-4 shrink-0 text-dimmed transition-transform"
              :class="expandedSlot === target.slot ? 'rotate-180' : ''"
              aria-hidden="true"
            />
          </button>

          <div
            v-if="expandedSlot === target.slot"
            :id="`automation-target-${target.slot}`"
            class="ai-profile__details"
          >
            <div data-testid="automation-target-core-fields" class="space-y-4">
              <UFormField :label="t('settingsAutomation.targets.name_label')" required>
                <UInput v-model.trim="target.label" size="sm" @change="commit" />
              </UFormField>
              <UFormField
                :label="t('settingsAutomation.targets.slot_label')"
                :hint="t('settingsAutomation.targets.slot_hint')"
              >
                <UInput :model-value="target.slot" size="sm" disabled class="font-mono" />
              </UFormField>
              <UFormField
                v-if="isDesktop(target)"
                :label="t('settingsAutomation.targets.application_label')"
                :hint="t('settingsAutomation.targets.application_hint')"
                required
              >
                <USelect
                  :model-value="target.applicationSlot"
                  :items="applicationItems"
                  size="sm"
                  @update:model-value="(value: string) => setApplication(index, value)"
                />
              </UFormField>
              <UFormField
                v-if="isDesktop(target)"
                :label="t('settingsAutomation.targets.backend_label')"
                :hint="t('settingsAutomation.targets.backend_hint')"
                required
              >
                <USelect
                  :model-value="target.inputBackend"
                  :items="backendItems"
                  size="sm"
                  @update:model-value="(value: InputBackend) => setBackend(index, value)"
                />
              </UFormField>
              <UFormField
                v-if="isDesktop(target)"
                :label="t('settingsAutomation.targets.capture_backend_label')"
                :hint="t('settingsAutomation.targets.capture_backend_hint')"
                required
              >
                <USelect
                  :model-value="target.captureBackend"
                  :items="captureBackendItems"
                  size="sm"
                  @update:model-value="(value: CaptureBackend) => setCaptureBackend(index, value)"
                />
              </UFormField>
            </div>

            <template v-if="isDesktop(target)">
              <div
                class="flex flex-wrap items-center gap-3 rounded-lg border border-primary/25 bg-primary/5 p-3"
              >
                <UIcon name="i-tabler-focus-2" class="size-4 shrink-0 text-primary" />
                <p class="min-w-0 flex-1 text-xs leading-5 text-dimmed">
                  {{ t('settingsAutomation.capture.hint') }}
                </p>
                <UButton
                  v-if="capturingSlot !== target.slot"
                  size="sm"
                  color="primary"
                  variant="soft"
                  icon="i-tabler-focus-2"
                  :disabled="Boolean(capturingSlot)"
                  @click="startCapture(target)"
                >
                  {{ t('settingsAutomation.capture.start') }}
                </UButton>
                <UButton
                  v-else
                  size="sm"
                  color="warning"
                  variant="soft"
                  icon="i-tabler-x"
                  @click="cancelCapture()"
                >
                  {{ t('settingsAutomation.capture.cancel') }}
                </UButton>
              </div>

              <div data-testid="automation-target-window-fields" class="space-y-4">
                <UFormField
                  :label="t('settingsAutomation.targets.window_title_label')"
                  :hint="t('settingsAutomation.targets.window_title_hint')"
                >
                  <UInput v-model="target.windowTitle" size="sm" @change="commit" />
                </UFormField>
                <UFormField
                  :label="t('settingsAutomation.targets.window_class_label')"
                  :hint="t('settingsAutomation.targets.window_class_hint')"
                >
                  <UInput
                    v-model="target.windowClass"
                    size="sm"
                    class="font-mono"
                    @change="commit"
                  />
                </UFormField>
              </div>

              <UFormField
                :label="t('settingsAutomation.targets.timeout_label')"
                :hint="t('settingsAutomation.targets.timeout_hint')"
              >
                <UInputNumber
                  v-model="target.resolveTimeoutMilliseconds"
                  :min="100"
                  :max="10000"
                  :step="100"
                  size="sm"
                  class="w-full sm:w-48"
                  @change="commit"
                />
              </UFormField>
            </template>

            <template v-else-if="isAndroid(target)">
              <div class="rounded-lg border border-primary/25 bg-primary/5 p-3">
                <div class="flex flex-wrap items-center gap-3">
                  <UIcon name="i-tabler-brand-android" class="size-4 shrink-0 text-primary" />
                  <p class="min-w-0 flex-1 text-xs leading-5 text-dimmed">
                    {{ t('settingsAutomation.android.discovery_hint') }}
                  </p>
                  <UButton
                    size="sm"
                    variant="soft"
                    icon="i-tabler-refresh"
                    :loading="adbLoading"
                    @click="refreshADBDevices"
                  >
                    {{ t('settingsAutomation.android.refresh') }}
                  </UButton>
                </div>
                <p v-if="adbError" class="mt-2 text-xs text-error" role="alert">{{ adbError }}</p>
              </div>
              <div class="grid gap-4 sm:grid-cols-2">
                <UFormField
                  :label="t('settingsAutomation.android.device_label')"
                  :hint="t('settingsAutomation.android.device_hint')"
                  required
                >
                  <USelect
                    :model-value="target.adbSerial"
                    :items="adbDeviceItems"
                    size="sm"
                    @update:model-value="(value: string) => setADBDevice(index, value)"
                  />
                </UFormField>
                <UFormField
                  :label="t('settingsAutomation.android.package_label')"
                  :hint="t('settingsAutomation.android.package_hint')"
                  required
                >
                  <UInput
                    v-model.trim="target.androidPackage"
                    size="sm"
                    class="font-mono"
                    @change="commit"
                  />
                </UFormField>
                <UFormField :label="t('settingsAutomation.android.identity_label')">
                  <UInput
                    :model-value="androidIdentity(target)"
                    size="sm"
                    disabled
                    class="font-mono"
                  />
                </UFormField>
                <UFormField :label="t('settingsAutomation.android.state_label')">
                  <div class="flex items-center gap-2">
                    <UBadge
                      :color="health[target.slot]?.ok ? 'success' : 'neutral'"
                      size="sm"
                      variant="subtle"
                    >
                      {{ health[target.slot]?.code ?? t('settingsAutomation.android.not_checked') }}
                    </UBadge>
                    <UButton
                      size="xs"
                      variant="ghost"
                      :disabled="!target.persisted"
                      :loading="healthLoading[target.slot]"
                      @click="checkHealth(target)"
                    >
                      {{ t('settingsAutomation.android.check_health') }}
                    </UButton>
                  </div>
                  <p v-if="health[target.slot]?.message" class="mt-1 text-xs text-dimmed">
                    {{ health[target.slot]?.message }}
                  </p>
                </UFormField>
              </div>
              <UFormField
                :label="t('settingsAutomation.targets.timeout_label')"
                :hint="t('settingsAutomation.targets.timeout_hint')"
              >
                <UInputNumber
                  v-model="target.resolveTimeoutMilliseconds"
                  :min="100"
                  :max="10000"
                  :step="100"
                  size="sm"
                  class="w-full sm:w-48"
                  @change="commit"
                />
              </UFormField>
            </template>

            <template v-else>
              <div class="rounded-lg border border-primary/25 bg-primary/5 p-3">
                <div class="flex flex-wrap items-center gap-3">
                  <UIcon name="i-tabler-brand-chrome" class="size-4 shrink-0 text-primary" />
                  <p class="min-w-0 flex-1 text-xs leading-5 text-dimmed">
                    {{ t('settingsAutomation.browser.discovery_hint') }}
                  </p>
                  <UButton
                    size="sm"
                    variant="soft"
                    icon="i-tabler-refresh"
                    :loading="browserLoading"
                    :disabled="!target.browserEndpoint?.trim()"
                    @click="refreshBrowserTargets(target)"
                  >
                    {{ t('settingsAutomation.browser.refresh') }}
                  </UButton>
                </div>
                <p v-if="browserError" class="mt-2 text-xs text-error" role="alert">
                  {{ browserError }}
                </p>
              </div>
              <div class="grid gap-4 sm:grid-cols-2">
                <UFormField
                  :label="t('settingsAutomation.browser.endpoint_label')"
                  :hint="t('settingsAutomation.browser.endpoint_hint')"
                  required
                >
                  <UInput
                    v-model.trim="target.browserEndpoint"
                    size="sm"
                    class="font-mono"
                    @change="browserEndpointChanged(target)"
                  />
                </UFormField>
                <UFormField
                  :label="t('settingsAutomation.browser.page_label')"
                  :hint="t('settingsAutomation.browser.page_hint')"
                  required
                >
                  <USelect
                    :model-value="target.browserTargetId"
                    :items="browserTargetItems"
                    size="sm"
                    @update:model-value="(value: string) => setBrowserTarget(index, value)"
                  />
                </UFormField>
                <UFormField :label="t('settingsAutomation.browser.url_label')">
                  <UInput
                    :model-value="target.browserUrl || '—'"
                    size="sm"
                    disabled
                    class="font-mono"
                  />
                </UFormField>
                <UFormField :label="t('settingsAutomation.browser.state_label')">
                  <div class="flex items-center gap-2">
                    <UBadge
                      :color="health[target.slot]?.ok ? 'success' : 'neutral'"
                      size="sm"
                      variant="subtle"
                    >
                      {{ health[target.slot]?.code ?? t('settingsAutomation.browser.not_checked') }}
                    </UBadge>
                    <UButton
                      size="xs"
                      variant="ghost"
                      :disabled="!target.persisted"
                      :loading="healthLoading[target.slot]"
                      @click="checkHealth(target)"
                    >
                      {{ t('settingsAutomation.browser.check_health') }}
                    </UButton>
                  </div>
                  <p v-if="health[target.slot]?.message" class="mt-1 text-xs text-dimmed">
                    {{ health[target.slot]?.message }}
                  </p>
                </UFormField>
              </div>
              <UFormField
                :label="t('settingsAutomation.targets.timeout_label')"
                :hint="t('settingsAutomation.targets.timeout_hint')"
              >
                <UInputNumber
                  v-model="target.resolveTimeoutMilliseconds"
                  :min="100"
                  :max="10000"
                  :step="100"
                  size="sm"
                  class="w-full sm:w-48"
                  @change="commit"
                />
              </UFormField>
            </template>

            <UFormField
              v-if="isDesktop(target)"
              :label="t('settingsAutomation.targets.mouse_counts_label')"
              :hint="t('settingsAutomation.targets.mouse_counts_hint')"
            >
              <UInputNumber
                v-model="target.mouseCounts360"
                :min="0"
                :max="10000000"
                :step="100"
                size="sm"
                class="w-full sm:w-48"
                @change="commit"
              />
            </UFormField>

            <div class="rounded-lg border border-warning/30 bg-warning/5 p-3">
              <div class="flex flex-wrap items-start gap-3">
                <UIcon
                  name="i-tabler-shield-exclamation"
                  class="mt-0.5 size-4 shrink-0 text-warning"
                  aria-hidden="true"
                />
                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <p class="text-xs font-medium text-default">
                      {{ t('settingsAutomation.consent.title') }}
                    </p>
                    <SettingsRestartBadge />
                  </div>
                  <p class="mt-1 text-xs leading-relaxed text-dimmed">
                    {{ t('settingsAutomation.consent.hint') }}
                  </p>
                </div>
                <UButton
                  v-if="target.workflowConsent"
                  size="sm"
                  variant="soft"
                  color="warning"
                  icon="i-tabler-shield-off"
                  :loading="busy[target.slot]"
                  @click="revoke(target)"
                >
                  {{ t('settingsAutomation.consent.revoke') }}
                </UButton>
                <UButton
                  v-else
                  size="sm"
                  variant="soft"
                  color="primary"
                  icon="i-tabler-shield-check"
                  :loading="busy[target.slot]"
                  :disabled="!target.persisted"
                  @click="grant(target)"
                >
                  {{ t('settingsAutomation.consent.grant') }}
                </UButton>
              </div>
            </div>

            <div class="flex items-center border-t border-default/60 pt-4">
              <UButton
                size="sm"
                variant="ghost"
                color="neutral"
                icon="i-tabler-copy"
                :disabled="!target.persisted"
                @click="duplicateTarget(target)"
              >
                {{ t('settingsAutomation.targets.duplicate') }}
              </UButton>
              <UButton
                class="ml-auto"
                size="sm"
                variant="ghost"
                color="error"
                icon="i-tabler-trash"
                @click="removeTarget(target)"
              >
                {{ t('settingsAutomation.targets.delete') }}
              </UButton>
            </div>
          </div>
        </article>
      </div>

      <div v-else class="settings-empty-state" role="status">
        <UIcon name="i-tabler-pointer-cog" class="size-6 text-dimmed" aria-hidden="true" />
        <p class="text-sm font-medium text-default">{{ t('settingsAutomation.targets.empty') }}</p>
        <p class="max-w-md text-center text-xs leading-relaxed text-dimmed">
          {{ t('settingsAutomation.targets.empty_hint') }}
        </p>
        <div class="flex flex-wrap justify-center gap-2">
          <UButton
            size="sm"
            color="primary"
            variant="soft"
            icon="i-tabler-brand-windows"
            :disabled="!desktopTargetType"
            @click="addTarget('desktop-window')"
          >
            {{ t('settingsAutomation.targets.add_windows') }}
          </UButton>
          <UButton
            size="sm"
            color="primary"
            variant="soft"
            icon="i-tabler-brand-android"
            :disabled="!androidTargetType"
            @click="addTarget('android-device')"
          >
            {{ t('settingsAutomation.targets.add_android') }}
          </UButton>
          <UButton
            size="sm"
            color="primary"
            variant="soft"
            icon="i-tabler-brand-chrome"
            :disabled="!browserTargetType"
            @click="addTarget('browser-page')"
          >
            {{ t('settingsAutomation.targets.add_browser') }}
          </UButton>
        </div>
      </div>
    </SettingsSection>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  backend,
  type AndroidDeviceDescriptor,
  type AutomationTargetHealth,
  type AutomationTargetTypeDescriptor,
  type BrowserTargetDescriptor,
  type InstalledAutomationTargetProfile,
} from '@/lib/backend'
import { useSettingsStore } from '@/stores/settings'
import { useConfirm } from '@/composables/useConfirm'
import SettingsRestartBadge from '@/components/settings/SettingsRestartBadge.vue'
import SettingsSection from '@/components/settings/SettingsSection.vue'
import { useWailsEvent } from '@/composables/useWailsEvent'
import { matchingInstalledApplications } from '@/settings/windowTargetCapture'

type InputBackend = InstalledAutomationTargetProfile['inputBackend']
type CaptureBackend = InstalledAutomationTargetProfile['captureBackend']
interface AutomationTargetDraft extends InstalledAutomationTargetProfile {
  persisted: boolean
}

const { t } = useI18n()
const { confirm } = useConfirm()
const store = useSettingsStore()
const applications = computed(() => store.data?.applications.profiles ?? [])
const targets = computed(() => store.data?.automation.targets ?? [])
const targetTypes = ref<AutomationTargetTypeDescriptor[]>([])
const desktopTargetType = computed(() =>
  targetTypes.value.find(
    (candidate) => candidate.profileKind === 'desktop-window' && candidate.hostAvailable,
  ),
)
const androidTargetType = computed(() =>
  targetTypes.value.find(
    (candidate) => candidate.profileKind === 'android-device' && candidate.hostAvailable,
  ),
)
const browserTargetType = computed(() =>
  targetTypes.value.find(
    (candidate) => candidate.profileKind === 'browser-page' && candidate.hostAvailable,
  ),
)
const draft = ref<AutomationTargetDraft[]>([])
const expandedSlot = ref('')
const busy = reactive<Record<string, boolean>>({})
const capturingSlot = ref('')
const captureID = ref('')
const captureFeedback = ref<{
  tone: 'success' | 'warning' | 'error'
  message: string
  elevationSuggested?: boolean
} | null>(null)
const elevated = ref(false)
const bulkConsentBusy = ref(false)
const adbDevices = ref<AndroidDeviceDescriptor[]>([])
const adbLoading = ref(false)
const adbError = ref('')
const browserTargets = ref<BrowserTargetDescriptor[]>([])
const browserLoading = ref(false)
const browserError = ref('')
const health = reactive<Record<string, AutomationTargetHealth | undefined>>({})
const healthLoading = reactive<Record<string, boolean>>({})
let captureTimer: ReturnType<typeof setTimeout> | undefined
const applicationItems = computed(() =>
  applications.value.map((application) => ({ label: application.label, value: application.slot })),
)
const hasMissingConsent = computed(
  () =>
    applications.value.some((application) => !application.workflowConsent) ||
    targets.value.some((target) => !target.workflowConsent),
)
const hasGrantedConsent = computed(
  () =>
    applications.value.some((application) => Boolean(application.workflowConsent)) ||
    targets.value.some((target) => Boolean(target.workflowConsent)),
)
const backendItems = computed(() =>
  (desktopTargetType.value?.inputBackends ?? []).map((value) => ({
    label: t(`settingsAutomation.backend.${value}`),
    value,
  })),
)
const captureBackendItems = computed(() =>
  (desktopTargetType.value?.captureBackends ?? []).map((value) => ({
    label: t(`settingsAutomation.captureBackend.${value}`),
    value,
  })),
)
const adbDeviceItems = computed(() =>
  adbDevices.value.map((device) => ({
    label: `${device.model || device.serial} · ${device.serial} · ${device.state}`,
    value: device.serial,
    disabled: device.state !== 'device' || !device.product || !device.model || !device.device,
  })),
)
const browserTargetItems = computed(() =>
  browserTargets.value.map((target) => ({
    label: `${target.title || target.url || target.id} · ${target.id}`,
    value: target.id,
  })),
)

onMounted(async () => {
  const [types, isElevated] = await Promise.all([
    backend.automation.listTargetTypes(),
    backend.tools.isElevated(),
  ])
  targetTypes.value = types ?? []
  elevated.value = isElevated
})

watch(
  targets,
  () => {
    draft.value = targets.value.map((target) => ({ ...target, persisted: true }))
  },
  { immediate: true },
)

useWailsEvent<unknown>('win32windowtarget:captured', (raw) => {
  const payload = Array.isArray(raw) ? raw[0] : raw
  void acceptCapture(payload)
})

onBeforeUnmount(() => {
  void cancelCapture(true)
})

function toggleExpanded(slot: string) {
  expandedSlot.value = expandedSlot.value === slot ? '' : slot
}
function applicationLabel(slot: string) {
  return applications.value.find((application) => application.slot === slot)?.label ?? slot
}
function isDesktop(target: AutomationTargetDraft): boolean {
  return target.targetKind === 'desktop-window' && target.adapterKind === 'win32'
}
function isAndroid(target: AutomationTargetDraft): boolean {
  return target.targetKind === 'android-device' && target.adapterKind === 'android-adb'
}
function isBrowser(target: AutomationTargetDraft): boolean {
  return target.targetKind === 'browser-cdp' && target.adapterKind === 'browser-cdp'
}
function targetSummary(target: AutomationTargetDraft): string {
  if (isDesktop(target)) return applicationLabel(target.applicationSlot)
  if (isBrowser(target))
    return `${target.browserTitle || t('settingsAutomation.browser.unselected')} · ${target.browserUrl || '—'}`
  return `${target.adbModel || t('settingsAutomation.android.unselected')} · ${target.adbSerial || '—'}`
}
function androidIdentity(target: AutomationTargetDraft): string {
  if (!target.adbSerial) return '—'
  return `${target.adbProduct || '?'} / ${target.adbModel || '?'} / ${target.adbDevice || '?'}`
}
function uniqueSlot(base: string): string {
  const taken = new Set([
    ...draft.value.map((target) => target.slot),
    ...(store.data?.ai.profiles ?? []).map((profile) => profile.slot),
    ...(store.data?.network.httpOrigins ?? []).map((origin) => origin.slot),
    ...(store.data?.applications.profiles ?? []).map((application) => application.slot),
  ])
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) if (!taken.has(`${base}-${index}`)) return `${base}-${index}`
}
function uniqueLabel(base: string): string {
  const taken = new Set(draft.value.map((target) => target.label))
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) if (!taken.has(`${base} ${index}`)) return `${base} ${index}`
}
function fileName(path: string): string {
  return path.split(/[\\/]/).pop() || path
}
function slug(value: string): string {
  return (
    value
      .toLocaleLowerCase()
      .replace(/\.exe$/i, '')
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-|-$/g, '') || 'application'
  )
}
function uniqueApplicationLabel(base: string): string {
  const taken = new Set(applications.value.map((application) => application.label))
  if (!taken.has(base)) return base
  for (let index = 2; ; index++) if (!taken.has(`${base} ${index}`)) return `${base} ${index}`
}
async function addTarget(profileKind: 'desktop-window' | 'android-device' | 'browser-page') {
  const type =
    profileKind === 'desktop-window'
      ? desktopTargetType.value
      : profileKind === 'android-device'
        ? androidTargetType.value
        : browserTargetType.value
  if (!type) return
  const desktop = profileKind === 'desktop-window'
  const browser = profileKind === 'browser-page'
  const slot = uniqueSlot(desktop ? 'window-target' : browser ? 'browser-target' : 'android-target')
  draft.value.push({
    slot,
    label: uniqueLabel(
      t(
        desktop
          ? 'settingsAutomation.targets.new_blank_label'
          : browser
            ? 'settingsAutomation.browser.new_blank_label'
            : 'settingsAutomation.android.new_blank_label',
      ),
    ),
    targetKind: type.targetKind as InstalledAutomationTargetProfile['targetKind'],
    adapterKind: type.adapterKind as InstalledAutomationTargetProfile['adapterKind'],
    applicationSlot: '',
    windowTitle: '',
    windowClass: '',
    inputBackend: (type.inputBackends[0] ?? '') as InputBackend,
    captureBackend: (type.captureBackends[0] ?? '') as CaptureBackend,
    mouseCounts360: 0,
    resolveTimeoutMilliseconds: 3000,
    adbSerial: '',
    adbProduct: '',
    adbModel: '',
    adbDevice: '',
    androidPackage: '',
    browserEndpoint: 'http://127.0.0.1:9222',
    browserTargetId: '',
    browserWebSocketUrl: '',
    browserTitle: '',
    browserUrl: '',
    persisted: false,
  })
  expandedSlot.value = slot
}
async function duplicateTarget(source: AutomationTargetDraft) {
  if (!source.persisted) return
  const slot = uniqueSlot(`${source.slot}-copy`)
  const duplicate: AutomationTargetDraft = {
    ...source,
    slot,
    label: uniqueLabel(t('settingsAutomation.targets.copy_label', { name: source.label })),
    persisted: false,
  }
  delete duplicate.workflowConsent
  draft.value.push(duplicate)
  expandedSlot.value = slot
  await commit()
}
function metadata(target: AutomationTargetDraft): InstalledAutomationTargetProfile {
  const common = {
    slot: target.slot,
    label: target.label.trim(),
    targetKind: target.targetKind,
    adapterKind: target.adapterKind,
    applicationSlot: target.applicationSlot,
    windowTitle: target.windowTitle.trim(),
    windowClass: target.windowClass.trim(),
    inputBackend: target.inputBackend,
    captureBackend: target.captureBackend,
    mouseCounts360: target.mouseCounts360,
    resolveTimeoutMilliseconds: target.resolveTimeoutMilliseconds,
    ...(target.workflowConsent ? { workflowConsent: target.workflowConsent } : {}),
  }
  if (isDesktop(target)) return common
  const adapterNeutral = {
    ...common,
    applicationSlot: '',
    windowTitle: '',
    windowClass: '',
    inputBackend: '' as InputBackend,
    captureBackend: '' as CaptureBackend,
    mouseCounts360: 0,
  }
  if (isBrowser(target))
    return {
      ...adapterNeutral,
      browserEndpoint: target.browserEndpoint?.trim(),
      browserTargetId: target.browserTargetId?.trim(),
      browserWebSocketUrl: target.browserWebSocketUrl?.trim(),
      browserTitle: target.browserTitle?.trim(),
      browserUrl: target.browserUrl?.trim(),
    }
  return {
    ...adapterNeutral,
    adbSerial: target.adbSerial?.trim(),
    adbProduct: target.adbProduct?.trim(),
    adbModel: target.adbModel?.trim(),
    adbDevice: target.adbDevice?.trim(),
    androidPackage: target.androidPackage?.trim(),
  }
}
async function commit(): Promise<boolean> {
  if (draft.value.some((target) => !targetComplete(target))) return false
  const ok = await store.patchAutomationTargets(draft.value.map(metadata))
  if (ok) for (const target of draft.value) target.persisted = true
  return ok
}
function targetComplete(target: AutomationTargetDraft): boolean {
  if (isBrowser(target)) {
    return Boolean(
      target.label.trim() &&
      target.browserEndpoint?.trim() &&
      target.browserTargetId?.trim() &&
      target.browserWebSocketUrl?.trim(),
    )
  }
  if (isAndroid(target)) {
    return Boolean(
      target.label.trim() &&
      target.adbSerial &&
      target.adbProduct &&
      target.adbModel &&
      target.adbDevice &&
      target.androidPackage?.trim(),
    )
  }
  return Boolean(
    target.label.trim() &&
    target.applicationSlot &&
    applications.value.some((application) => application.slot === target.applicationSlot) &&
    target.windowTitle.trim() &&
    target.windowClass.trim(),
  )
}

async function refreshBrowserTargets(target: AutomationTargetDraft): Promise<void> {
  browserLoading.value = true
  browserError.value = ''
  try {
    browserTargets.value =
      (await backend.automation.listBrowserTargets(target.browserEndpoint?.trim() || '')) ?? []
    if (browserTargets.value.length === 0)
      browserError.value = t('settingsAutomation.browser.none_found')
  } catch (error) {
    browserTargets.value = []
    browserError.value = errorText(error)
  } finally {
    browserLoading.value = false
  }
}

function browserEndpointChanged(target: AutomationTargetDraft): void {
  target.browserTargetId = ''
  target.browserWebSocketUrl = ''
  target.browserTitle = ''
  target.browserUrl = ''
  delete target.workflowConsent
  delete health[target.slot]
  browserTargets.value = []
}

async function setBrowserTarget(index: number, id: string): Promise<void> {
  const selected = browserTargets.value.find((target) => target.id === id)
  const target = draft.value[index]
  if (!selected || !target || !isBrowser(target)) return
  target.browserTargetId = selected.id
  target.browserWebSocketUrl = selected.webSocketDebuggerUrl
  target.browserTitle = selected.title
  target.browserUrl = selected.url
  delete target.workflowConsent
  delete health[target.slot]
  await commit()
}
async function refreshADBDevices(): Promise<void> {
  adbLoading.value = true
  adbError.value = ''
  try {
    adbDevices.value = (await backend.automation.listADBDevices()) ?? []
    if (adbDevices.value.length === 0) adbError.value = t('settingsAutomation.android.none_found')
  } catch (error) {
    adbError.value = errorText(error)
  } finally {
    adbLoading.value = false
  }
}
async function setADBDevice(index: number, serial: string): Promise<void> {
  const selected = adbDevices.value.find(
    (device) => device.serial === serial && device.state === 'device',
  )
  const target = draft.value[index]
  if (!selected || !target) return
  target.adbSerial = selected.serial
  target.adbProduct = selected.product
  target.adbModel = selected.model
  target.adbDevice = selected.device
  delete health[target.slot]
  await commit()
}
async function checkHealth(target: AutomationTargetDraft): Promise<void> {
  if (!(await commit())) return
  healthLoading[target.slot] = true
  try {
    health[target.slot] = await backend.automation.checkTargetHealth(target.slot)
  } catch (error) {
    health[target.slot] = { ok: false, code: 'unavailable', message: errorText(error) }
  } finally {
    healthLoading[target.slot] = false
  }
}
async function setApplication(index: number, value: string) {
  draft.value[index]!.applicationSlot = value
  await commit()
}
async function setBackend(index: number, value: InputBackend) {
  draft.value[index]!.inputBackend = value
  await commit()
}
async function setCaptureBackend(index: number, value: CaptureBackend) {
  draft.value[index]!.captureBackend = value
  await commit()
}
async function grant(target: AutomationTargetDraft) {
  if (!(await commit())) return
  busy[target.slot] = true
  try {
    await backend.automation.grantWorkflowConsent(target.slot)
  } finally {
    busy[target.slot] = false
  }
}
async function revoke(target: AutomationTargetDraft) {
  busy[target.slot] = true
  try {
    await backend.automation.revokeWorkflowConsent(target.slot)
  } finally {
    busy[target.slot] = false
  }
}
async function grantAllConsents(): Promise<void> {
  if (bulkConsentBusy.value) return
  const accepted = await confirm({
    title: t('settingsAutomation.bulk.grant_title'),
    description: t('settingsAutomation.bulk.grant_hint'),
    confirmText: t('settingsAutomation.bulk.grant'),
    cancelText: t('common.cancel'),
    color: 'warning',
  })
  if (accepted !== true) return
  bulkConsentBusy.value = true
  try {
    await backend.automation.grantAllWorkflowConsents()
    await store.load()
    captureFeedback.value = {
      tone: 'success',
      message: t('settingsAutomation.bulk.granted'),
    }
  } catch (error) {
    captureFeedback.value = { tone: 'error', message: errorText(error) }
  } finally {
    bulkConsentBusy.value = false
  }
}
async function revokeAllConsents(): Promise<void> {
  if (bulkConsentBusy.value) return
  const accepted = await confirm({
    title: t('settingsAutomation.bulk.revoke_title'),
    description: t('settingsAutomation.bulk.revoke_hint'),
    confirmText: t('settingsAutomation.bulk.revoke'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (accepted !== true) return
  bulkConsentBusy.value = true
  try {
    await backend.automation.revokeAllWorkflowConsents()
    await store.load()
    captureFeedback.value = {
      tone: 'warning',
      message: t('settingsAutomation.bulk.revoked'),
    }
  } catch (error) {
    captureFeedback.value = { tone: 'error', message: errorText(error) }
  } finally {
    bulkConsentBusy.value = false
  }
}
async function removeTarget(target: AutomationTargetDraft) {
  if (capturingSlot.value === target.slot) await cancelCapture(true)
  if (!target.persisted) {
    draft.value = draft.value.filter((candidate) => candidate.slot !== target.slot)
    return
  }
  const accepted = await confirm({
    title: t('settingsAutomation.confirm.delete_title', { name: target.label }),
    description: t('settingsAutomation.confirm.delete_hint'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    color: 'error',
  })
  if (accepted === true) {
    await store.patchAutomationTargets(
      targets.value.filter((candidate) => candidate.slot !== target.slot),
    )
  }
}

async function startCapture(target: AutomationTargetDraft): Promise<void> {
  if (capturingSlot.value) return
  captureFeedback.value = null
  capturingSlot.value = target.slot
  try {
    const id = await backend.tools.startWin32WindowTargetCapture()
    if (!id) throw new Error(t('settingsAutomation.capture.start_failed'))
    captureID.value = id
    captureTimer = setTimeout(() => void timeoutCapture(), 30_000)
  } catch (error) {
    resetCapture()
    captureFeedback.value = { tone: 'error', message: errorText(error) }
  }
}

async function cancelCapture(silent = false): Promise<void> {
  const id = captureID.value
  resetCapture()
  if (id) {
    try {
      await backend.tools.cancelWin32WindowTargetCapture(id)
    } catch (error) {
      if (!silent) captureFeedback.value = { tone: 'error', message: errorText(error) }
      return
    }
  }
  if (!silent) {
    captureFeedback.value = {
      tone: 'warning',
      message: t('settingsAutomation.capture.cancelled'),
    }
  }
}

async function timeoutCapture(): Promise<void> {
  await cancelCapture(true)
  captureFeedback.value = {
    tone: 'warning',
    message: t(
      elevated.value
        ? 'settingsAutomation.capture.timeout_elevated'
        : 'settingsAutomation.capture.timeout_uac',
    ),
    elevationSuggested: !elevated.value,
  }
}

async function restartElevated(): Promise<void> {
  const accepted = await confirm({
    title: t('settingsAutomation.capture.restart_elevated_title'),
    description: t('settingsAutomation.capture.restart_elevated_hint'),
    confirmText: t('settingsAutomation.capture.restart_elevated'),
    cancelText: t('common.cancel'),
    color: 'warning',
  })
  if (accepted !== true) return
  try {
    await backend.tools.restartElevated()
  } catch (error) {
    captureFeedback.value = { tone: 'error', message: errorText(error) }
  }
}

async function acceptCapture(raw: unknown): Promise<void> {
  const slot = capturingSlot.value
  if (!slot || typeof raw !== 'object' || raw === null) return
  const payload = raw as Record<string, unknown>
  resetCapture()
  if (typeof payload.error === 'string' && payload.error) {
    captureFeedback.value = { tone: 'error', message: payload.error }
    return
  }
  if (
    typeof payload.executable !== 'string' ||
    typeof payload.title !== 'string' ||
    typeof payload.class !== 'string' ||
    !payload.executable ||
    !payload.title ||
    !payload.class
  ) {
    captureFeedback.value = {
      tone: 'error',
      message: t('settingsAutomation.capture.incomplete'),
    }
    return
  }
  try {
    const inspection = await backend.applications.inspectExecutable(payload.executable)
    if (!inspection) throw new Error(t('settingsAutomation.capture.inspect_failed'))
    const matches = matchingInstalledApplications(applications.value, inspection)
    if (matches.length === 0) {
      const executableName = fileName(inspection.executable).replace(/\.exe$/i, '')
      const accepted = await confirm({
        title: t('settingsAutomation.capture.install_title', { name: executableName }),
        description: t('settingsAutomation.capture.install_hint', {
          path: inspection.executable,
        }),
        confirmText: t('settingsAutomation.capture.install_confirm'),
        cancelText: t('common.cancel'),
        color: 'warning',
      })
      if (accepted !== true) {
        captureFeedback.value = {
          tone: 'warning',
          message: t('settingsAutomation.capture.install_cancelled'),
        }
        return
      }
      const target = draft.value.find((candidate) => candidate.slot === slot)
      if (!target) return
      const application = {
        slot: uniqueSlot(slug(executableName)),
        label: uniqueApplicationLabel(executableName),
        executable: inspection.executable,
        executableDigest: inspection.digest,
        arguments: [],
      }
      target.applicationSlot = application.slot
      target.windowTitle = payload.title
      target.windowClass = payload.class
      delete target.workflowConsent
      const ok = await store.patch({
        applications: { profiles: [...applications.value, application] },
        automation: { targets: draft.value.map(metadata) },
      })
      if (!ok) throw new Error(t('settingsAutomation.capture.save_failed'))
      for (const candidate of draft.value) candidate.persisted = true
      captureFeedback.value = {
        tone: 'success',
        message: t('settingsAutomation.capture.installed_and_completed', { name: target.label }),
      }
      return
    }
    if (matches.length > 1) throw new Error(t('settingsAutomation.capture.application_ambiguous'))
    const target = draft.value.find((candidate) => candidate.slot === slot)
    if (!target) return
    target.applicationSlot = matches[0]!.slot
    target.windowTitle = payload.title
    target.windowClass = payload.class
    if (!(await commit())) throw new Error(t('settingsAutomation.capture.save_failed'))
    captureFeedback.value = {
      tone: 'success',
      message: t('settingsAutomation.capture.completed', { name: target.label }),
    }
  } catch (error) {
    captureFeedback.value = { tone: 'error', message: errorText(error) }
  }
}

function resetCapture(): void {
  if (captureTimer) clearTimeout(captureTimer)
  captureTimer = undefined
  captureID.value = ''
  capturingSlot.value = ''
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
</script>
