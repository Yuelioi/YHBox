---
kind: note
summary: "Workflow HTTP access is an installed exact-origin capability; nodes never receive an ambient URL client."
activation: action
read_when: "modifying HTTP/network nodes, egress providers, SSRF controls, network settings, target slots, consent policy, response budgets, or network-related plugin imports"
recheck_when: "adding methods, credentials, request bodies/headers, binary/streaming responses, proxies, redirects, WebSockets, or live installation reload"
---
# Workflow HTTP is installed origin authority

HTTP GET binds capability `https://schemas.yotta.dev/capabilities/network/http-get/v1` to a logical slot whose production target is one immutable `httpegress.Profile`. The profile digest includes exact scheme/authority, private-network policy, timeout and response-byte budget. Workflow data supplies only an absolute relative-path form and a canonical string-list query object; it cannot select a scheme, host, port, proxy, redirect, credential, Cookie or request header.

Public profiles require HTTPS. DNS is resolved in the provider, every result is validated, and the transport dials a validated numeric address while preserving the installed hostname for HTTP/TLS. Public profiles reject loopback, private, link-local, multicast, unspecified, CGNAT and benchmark ranges. Explicit private-network profiles may reach loopback/private addresses but still reject link-local, multicast and unspecified destinations. Redirect following and environment proxies are disabled.

The provider enforces a total timeout, a maximum 256 KiB decoded UTF-8 body, and a bounded content type. Workflow output contains only status, body and content-type; all other headers remain outside workflow data. AdapterAction records status/byte counters and a domain-separated path digest, never URL/query/body/response text.

The capability is sensitive with ConsentOnce. Policy accepts only the exact installed provider artifact, ABI, target, resource kind and consent digest. Editing any profile semantic revokes persisted consent; AI and HTTP installations cannot share a target slot. Installations are startup snapshots, with no live or ambient fallback.

Authentication, request bodies, custom headers, binary responses, streaming and WebSockets require separate operations/contracts, credential or data-carrier design, budgets, consent identity and conformance tests. They must not widen HTTP GET through config flags.
