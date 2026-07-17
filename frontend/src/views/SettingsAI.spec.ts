import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(join(process.cwd(), 'src/views/SettingsAI.vue'), 'utf8')

describe('SettingsAI provider endpoint installation', () => {
  it('persists an exact provider-native endpoint and restores explicit official defaults', () => {
    expect(source).toContain('v-model="profile.endpoint"')
    expect(source).toContain('endpoint: profile.endpoint.trim()')
    expect(source).toContain('allowLocalHttp: profile.allowLocalHttp')
    expect(source).toContain('restoreProviderEndpoint(profile)')
    expect(source).toContain('https://api.openai.com/v1/responses')
    expect(source).toContain('https://api.anthropic.com/v1/messages')
  })

  it('requires explicit risk acknowledgement for loopback HTTP without adding protocol fallback', () => {
    expect(source).toContain("startsWith('http://')")
    expect(source).toContain('setLocalHTTP(index, value)')
    expect(source).not.toContain('chat-completions')
    expect(source).not.toContain('openai-compatible')
  })
})
