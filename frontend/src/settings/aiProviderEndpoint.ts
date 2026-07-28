export type AIProviderEndpointIssue = 'invalid' | 'https_required' | 'local_http_ack_required'

export function aiProviderEndpointIssue(
  rawEndpoint: string,
  allowLocalHTTP: boolean,
): AIProviderEndpointIssue | undefined {
  const raw = rawEndpoint.trim()
  if (!raw) return undefined
  if (raw.includes('\\')) return 'invalid'

  let endpoint: URL
  try {
    endpoint = new URL(raw)
  } catch {
    return 'invalid'
  }

  if (
    endpoint.username ||
    endpoint.password ||
    endpoint.search ||
    endpoint.hash ||
    !endpoint.hostname
  ) {
    return 'invalid'
  }

  if (endpoint.protocol === 'http:') {
    if (!isLoopbackHost(endpoint.hostname)) return 'https_required'
    if (!allowLocalHTTP) return 'local_http_ack_required'
  } else if (endpoint.protocol !== 'https:') {
    return 'https_required'
  }

  return undefined
}

function isLoopbackHost(rawHost: string): boolean {
  const host = rawHost.replace(/^\[|\]$/g, '').toLocaleLowerCase()
  if (host === 'localhost' || host === '::1') return true

  const octets = host.split('.').map(Number)
  return (
    octets.length === 4 &&
    octets[0] === 127 &&
    octets.every((octet) => Number.isInteger(octet) && octet >= 0 && octet <= 255)
  )
}
