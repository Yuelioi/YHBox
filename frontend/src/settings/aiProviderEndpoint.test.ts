import { describe, expect, it } from 'vitest'
import { aiProviderEndpointIssue } from './aiProviderEndpoint'

describe('aiProviderEndpointIssue', () => {
  it('accepts provider root URLs without inventing a request path', () => {
    expect(aiProviderEndpointIssue('https://api.deepseek.com', false)).toBeUndefined()
    expect(aiProviderEndpointIssue('https://api.deepseek.com/v1/responses', false)).toBeUndefined()
  })

  it('requires HTTPS for remote endpoints', () => {
    expect(aiProviderEndpointIssue('http://api.example.com/v1/responses', false)).toBe(
      'https_required',
    )
  })

  it('requires explicit acknowledgement for loopback HTTP', () => {
    expect(aiProviderEndpointIssue('http://127.0.0.1:8080/v1/responses', false)).toBe(
      'local_http_ack_required',
    )
    expect(aiProviderEndpointIssue('http://127.0.0.1:8080/v1/responses', true)).toBeUndefined()
  })

  it('rejects credentials, query parameters, and fragments', () => {
    expect(aiProviderEndpointIssue('https://user:pass@api.example.com/v1/responses', false)).toBe(
      'invalid',
    )
    expect(aiProviderEndpointIssue('https://api.example.com/v1/responses?debug=1', false)).toBe(
      'invalid',
    )
    expect(aiProviderEndpointIssue('https://api.example.com/v1/responses#debug', false)).toBe(
      'invalid',
    )
  })
})
