import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/SettingsAI.vue'), 'utf8')

describe('SettingsAI provider endpoint installation', () => {
  it('persists the provider address unchanged and restores explicit official defaults', () => {
    expect(source).toContain('v-model="profile.endpoint"')
    expect(source).toContain('endpoint: profile.endpoint.trim()')
    expect(source).toContain('allowLocalHttp: profile.allowLocalHttp')
    expect(source).toContain('restoreProviderEndpoint(profile)')
    expect(source).toContain('https://api.openai.com/v1/responses')
    expect(source).toContain('https://api.anthropic.com/v1/messages')
  })

  it('requires explicit risk acknowledgement for loopback HTTP and offers Chat Completions explicitly', () => {
    expect(source).toContain("startsWith('http://')")
    expect(source).toContain('setLocalHTTP(index, value)')
    expect(source).toContain('openai-chat-completions')
    expect(source).not.toContain('openai-compatible')
  })

  it('preserves manually entered token limits and exposes an unlimited setting', () => {
    expect(source).toContain(':step-snapping="false"')
    expect(source).toContain('setUnlimitedOutputTokens(index, value)')
    expect(source).toContain('settingsAI.profiles.max_tokens_unlimited')
  })

  it('renders field remarks below labels as compact secondary text', () => {
    expect(source).toContain("description: 'mt-1 text-[11px] leading-4 font-normal text-dimmed'")
    expect(source).toContain('compactDescriptionUI')
    expect(source).not.toContain(':hint="t(\'settingsAI.profiles.')
  })

  it('describes a successful model test without exposing provider finish codes', () => {
    expect(source).toContain("t('settingsAI.profiles.test_ok'")
    expect(source).not.toContain('finish: results[profile.slot]!.finish')
  })
})
