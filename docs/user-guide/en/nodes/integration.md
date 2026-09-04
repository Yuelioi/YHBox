---
title: Files, Network, Scripts, and AI Nodes
description: Connect local data, HTTP services, isolated scripts, and models
slug: integration-nodes
order: 37
source_id: yotta-integration-nodes
---

# Files, Network, Scripts, and AI Nodes

File nodes read text, JSON, images, metadata, and save images within their configured boundary. Write
Log records bounded Run-scoped information; avoid noisy logging inside tight loops.

HTTP GET uses a configured network slot with base URL, timeout, and response limit. A common chain is:

```text
HTTP GET → Parse JSON → JSON Path → Compare → Branch
```

Handle timeout and server failure separately. Do not retry authentication or invalid configuration
without a change.

Execute Script runs JavaScript in an isolated one-shot worker with normalized JSON input/output and a
hard timeout. It must not bypass target, file, network, or input boundaries.

Generate Text and Extract Structured Data use an installed AI model slot. Treat model output as
untrusted data before passing it to applications, network calls, or automation effects.
