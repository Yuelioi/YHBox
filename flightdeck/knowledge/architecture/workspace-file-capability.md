---
kind: note
summary: "Workflow file effects use only the operation-scoped workspace-files capability and exact managed target; bounded reads and staged writes never accept an ambient or absolute host filesystem path."
activation: action
read_when: "modifying file nodes, workspace imports, filesystem providers, path validation, Host Profile targets, file permission UX, or file-related plugin imports"
recheck_when: "workspace storage layout, filesystem provider ABI, write capability, import/export flow, or supported-platform path policy changes"
---
# Workflow files are target-bound, not ambient paths

Node Contract 3.1 file nodes bind capability `https://schemas.yotta.dev/capabilities/filesystem/workspace/v1` to target slot `workspace-files` and scope `{"root":"workflow-files"}`. The production Host Profile pins that slot to the built-in `yotta.workspace-files` provider, its exact artifact digest, ABI, resource kind and Yotta-managed `workspace-3.1/files` root. Policy must reject any changed provider, target, kind, credential or artifact. Each node requests only its exact operation subset; declaring the capability does not grant every file operation.

Workflow data supplies only a relative path. The provider rejects empty, absolute, volume-qualified and traversal paths, resolves existing links or the destination parent, and verifies the object remains below the managed root. Read/stat/range and append/cancel/commit requests are strict tagged payloads with per-file and per-chunk budgets. A writer stages a regular file in the destination directory; commit uses durable no-clobber publish or atomic replacement, while cancel/drop removes the staging file. Existing directories and symlinks are never replaceable.

Text decoding and one-document JSON parsing happen in built-in adapters. `Load workspace image` validates a bounded PNG and commits an Image BlobRef; `Save image to workspace` reads an exact Image BlobRef and streams it to the managed root. Outputs are sealed against File Metadata/String/JSON/Image Data Types. Arbitrary import/export paths remain application commands outside Workflow Source.

The action journal stores stable outcome codes, byte counters and a domain-separated path digest. It never stores the path or file contents. There is no fallback to `YOTTA_DATA_DIR`, process working directory, arbitrary host path, legacy File object or direct `os.ReadFile` in a node adapter.

Adding a new file effect requires a new explicit provider operation and node requirement with its own quota and failure contract. It must not widen an existing node requirement, expose a host path, follow links, or bypass the staged writer.

