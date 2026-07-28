import { createPinia, setActivePinia } from 'pinia'
import ui from '@nuxt/ui/vue-plugin'
import { createApp, nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  updateSettings: vi.fn<(patch: unknown) => Promise<void>>(async () => undefined),
  secretStatus: vi.fn<(slots: string[]) => Promise<Record<string, boolean>>>(async () => ({})),
  setAPIKey: vi.fn<(slot: string, apiKey: string) => Promise<void>>(async () => undefined),
  testProfile: vi.fn(async () => ({
    ok: false,
    provider: '',
    requestedModel: '',
    resolvedModel: '',
    finish: '',
    failureClass: 'not-found',
    httpStatus: 404,
    error: 'AI provider failure: not-found',
  })),
}))

vi.mock('@/lib/backend', () => ({
  backend: {
    settings: {
      update: mocks.updateSettings,
    },
    ai: {
      secretStatus: mocks.secretStatus,
      setAPIKey: mocks.setAPIKey,
      deleteAPIKey: vi.fn(async () => undefined),
      testProfile: mocks.testProfile,
    },
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => ({ add: vi.fn() }),
}))

vi.mock('@/composables/useConfirm', () => ({
  useConfirm: () => ({ confirm: vi.fn(async () => true) }),
}))

vi.mock('@/components/common/AdaptiveSelect.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      setup: () => () => h('div'),
    }),
  }
})

import SettingsAI from './SettingsAI.vue'
import { useSettingsStore, type Settings } from '@/stores/settings'

const mounted: Array<ReturnType<typeof createApp>> = []

beforeEach(() => {
  mocks.updateSettings.mockReset()
  mocks.updateSettings.mockResolvedValue(undefined)
  mocks.secretStatus.mockReset()
  mocks.secretStatus.mockResolvedValue({})
  mocks.setAPIKey.mockReset()
  mocks.setAPIKey.mockResolvedValue(undefined)
  mocks.testProfile.mockReset()
  mocks.testProfile.mockResolvedValue({
    ok: false,
    provider: '',
    requestedModel: '',
    resolvedModel: '',
    finish: '',
    failureClass: 'not-found',
    httpStatus: 404,
    error: 'AI provider failure: not-found',
  })
})

afterEach(() => {
  for (const app of mounted.splice(0)) app.unmount()
  document.body.innerHTML = ''
})

describe('SettingsAI model draft', () => {
  it('explains a Responses 404 and points to Chat Completions', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const store = useSettingsStore()
    store.data = {
      ui: {},
      locale: 'zh',
      capture: {},
      ai: {
        profiles: [
          {
            slot: 'model',
            label: 'DeepSeek',
            provider: 'openai-responses',
            endpoint: 'https://api.deepseek.com',
            allowLocalHttp: false,
            model: 'deepseek-v4-flash',
            maxOutputTokens: 4096,
            capabilities: {
              structuredOutput: false,
              toolCalling: false,
              parallelTools: false,
              background: false,
              zeroRetention: false,
            },
            pricing: {
              inputMicrounitsPerMillion: 0,
              cacheReadMicrounitsPerMillion: 0,
              outputMicrounitsPerMillion: 0,
            },
            evaluation: 'unverified',
          },
        ],
      },
      network: { httpOrigins: [] },
      applications: { profiles: [] },
      automation: { targets: [] },
    } as unknown as Settings

    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(SettingsAI)
    app.use(pinia)
    app.use(ui)
    mounted.push(app)
    app.mount(root)
    await nextTick()

    const profileHeader = root.querySelector('.ai-profile > button') as HTMLButtonElement
    profileHeader.click()
    await nextTick()
    const test = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('settingsAI.profiles.test'),
    )
    test?.click()

    await vi.waitFor(() =>
      expect(root.textContent).toContain('settingsAI.test_errors.not_found_responses'),
    )
  })

  it('saves a provider root URL unchanged', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const store = useSettingsStore()
    store.data = {
      ui: {},
      locale: 'zh',
      capture: {},
      ai: { profiles: [] },
      network: { httpOrigins: [] },
      applications: { profiles: [] },
      automation: { targets: [] },
    } as unknown as Settings

    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(SettingsAI)
    app.use(pinia)
    app.use(ui)
    mounted.push(app)
    app.mount(root)
    await nextTick()

    const add = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('settingsAI.profiles.add'),
    )
    add?.click()
    await nextTick()

    const endpoint = root.querySelector('input[type="url"]') as HTMLInputElement
    endpoint.value = 'https://api.deepseek.com'
    endpoint.dispatchEvent(new Event('input', { bubbles: true }))

    const model = root.querySelector(
      'input[placeholder="settingsAI.profiles.model_placeholder"]',
    ) as HTMLInputElement
    model.value = 'deepseek-v4-flash'
    model.dispatchEvent(new Event('input', { bubbles: true }))
    model.dispatchEvent(new Event('change', { bubbles: true }))

    await vi.waitFor(() => expect(mocks.updateSettings).toHaveBeenCalledOnce())
    expect(mocks.updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        ai: expect.objectContaining({
          profiles: [
            expect.objectContaining({
              endpoint: 'https://api.deepseek.com',
              model: 'deepseek-v4-flash',
            }),
          ],
        }),
      }),
    )
    expect(store.saveState).toBe('saved')
  })

  it('keeps an incomplete new profile open when the API key field loses focus', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const store = useSettingsStore()
    store.data = {
      ui: {},
      locale: 'zh',
      capture: {},
      ai: { profiles: [] },
      network: { httpOrigins: [] },
      applications: { profiles: [] },
      automation: { targets: [] },
    } as unknown as Settings

    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(SettingsAI)
    app.use(pinia)
    app.use(ui)
    mounted.push(app)
    app.mount(root)
    await nextTick()

    const add = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('settingsAI.profiles.add'),
    )
    add?.click()
    await nextTick()

    const apiKey = root.querySelector(
      'input[placeholder="settingsAI.profiles.apikey_placeholder"]',
    ) as HTMLInputElement
    apiKey.value = 'secret-in-progress'
    apiKey.dispatchEvent(new Event('input', { bubbles: true }))
    apiKey.dispatchEvent(new Event('change', { bubbles: true }))
    await new Promise((resolve) => setTimeout(resolve, 0))
    await nextTick()

    expect(root.querySelector('.ai-profile__details')).toBeTruthy()
    expect(
      root.querySelector('input[placeholder="settingsAI.profiles.model_placeholder"]'),
    ).toBeTruthy()
    expect(mocks.updateSettings).not.toHaveBeenCalled()
    expect(mocks.setAPIKey).not.toHaveBeenCalled()

    const model = root.querySelector(
      'input[placeholder="settingsAI.profiles.model_placeholder"]',
    ) as HTMLInputElement
    model.value = 'gpt-test'
    model.dispatchEvent(new Event('input', { bubbles: true }))
    model.dispatchEvent(new Event('change', { bubbles: true }))

    await vi.waitFor(() => expect(mocks.updateSettings).toHaveBeenCalledOnce())
    await vi.waitFor(() =>
      expect(mocks.setAPIKey).toHaveBeenCalledWith('model', 'secret-in-progress'),
    )
    await nextTick()
    expect(root.querySelector('.ai-profile__details')).toBeTruthy()
  })

  it('keeps the API key draft focused when the preceding model change finishes saving', async () => {
    let finishUpdate: (() => void) | undefined
    mocks.updateSettings.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          finishUpdate = resolve
        }),
    )
    const pinia = createPinia()
    setActivePinia(pinia)
    const store = useSettingsStore()
    store.data = {
      ui: {},
      locale: 'zh',
      capture: {},
      ai: { profiles: [] },
      network: { httpOrigins: [] },
      applications: { profiles: [] },
      automation: { targets: [] },
    } as unknown as Settings

    const root = document.createElement('div')
    document.body.append(root)
    const app = createApp(SettingsAI)
    app.use(pinia)
    app.use(ui)
    mounted.push(app)
    app.mount(root)
    await nextTick()

    const add = [...root.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('settingsAI.profiles.add'),
    )
    expect(add).toBeTruthy()
    add?.click()
    await nextTick()

    const model = root.querySelector(
      'input[placeholder="settingsAI.profiles.model_placeholder"]',
    ) as HTMLInputElement
    expect(model).toBeTruthy()
    model.value = 'gpt-test'
    model.dispatchEvent(new Event('input', { bubbles: true }))
    model.dispatchEvent(new Event('change', { bubbles: true }))
    await vi.waitFor(() => expect(mocks.updateSettings).toHaveBeenCalledOnce())

    const apiKey = root.querySelector(
      'input[placeholder="settingsAI.profiles.apikey_placeholder"]',
    ) as HTMLInputElement
    expect(apiKey).toBeTruthy()
    apiKey.focus()
    apiKey.value = 'secret-in-progress'
    apiKey.dispatchEvent(new Event('input', { bubbles: true }))
    expect(document.activeElement).toBe(apiKey)
    expect(apiKey.value).toBe('secret-in-progress')

    finishUpdate?.()
    await vi.waitFor(() => expect(store.saveState).toBe('saved'))
    await nextTick()

    const currentAPIKey = root.querySelector(
      'input[placeholder="settingsAI.profiles.apikey_placeholder"]',
    ) as HTMLInputElement
    expect(root.querySelector('.ai-profile__details')).toBeTruthy()
    expect(document.activeElement).toBe(currentAPIKey)
    await vi.waitFor(() =>
      expect(mocks.setAPIKey).toHaveBeenCalledWith('model', 'secret-in-progress'),
    )
    await nextTick()
    expect(currentAPIKey.value).toBe('')
  })
})
