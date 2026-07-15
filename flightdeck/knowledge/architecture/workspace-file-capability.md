---
kind: note
summary: "Workflow file effects use only the read-only workspace-files capability and exact managed target; 3.1 never accepts an ambient or absolute host filesystem path."
activation: action
read_when: "modifying file nodes, workspace imports, filesystem providers, path validation, Host Profile targets, file permission UX, or file-related plugin imports"
recheck_when: "workspace storage layout, filesystem provider ABI, write capability, import/export flow, or supported-platform path policy changes"
---
# Workflow files are target-bound, not ambient paths

Node Contract 3.1 file nodes bind capability `https://schemas.yotta.dev/capabilities/filesystem/read/v1` to target slot `workspace-files` and scope `{"root":"workflow-files"}`. The production Host Profile pins that slot to the built-in `yotta.workspace-files` provider, its exact artifact digest, ABI, resource kind and Yotta-managed `workspace-3.1/files` root. Policy must reject any changed provider, target, kind, credential or artifact.

Workflow data supplies only a relative path. The provider rejects empty, absolute, volume-qualified and traversal paths, resolves existing links, and verifies the resolved object remains below the managed root. Read and stat requests are strict tagged payloads with bounded bytes; text decoding and one-document JSON parsing happen in the built-in adapter, and outputs are sealed against File Metadata/String/JSON Data Types.

The action journal stores stable outcome codes, byte counters and a domain-separated path digest. It never stores the path or file contents. There is no fallback to `YOTTA_DATA_DIR`, process working directory, arbitrary host path, legacy File object or direct `os.ReadFile` in a node adapter.

A future write/import capability must be a separate capability and provider operation with its own atomicity, quota, consent and target policy. It must not widen this read-only contract.
