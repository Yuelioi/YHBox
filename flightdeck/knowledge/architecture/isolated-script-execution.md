---
kind: note
summary: "Script 3.1 executes only in a one-shot, zero-ambient-authority worker; Windows requires LPAC/AppContainer plus atomic Job Object containment, and unsupported platforms fail closed."
activation: action
read_when: "modifying script nodes, guest runtimes, worker launchers, script ABI, script permissions, replay/retry, or cross-platform execution claims"
recheck_when: "the guest engine, Windows sandbox APIs, runner protocol, or supported-platform policy changes"
---
# Script execution is an isolated typed effect

Production code must never evaluate user scripts inside the Wails process. Each attempt launches one worker with a strict length-delimited protocol and an exact typed JSON input/output ABI. The guest receives no registry, node enumeration, variables, filesystem, network, process, window, secret, or arbitrary Go object bridge.

On Windows, a runnable worker requires LPAC/AppContainer and atomic Job Object association at process creation. Apply active-process=1, memory, CPU/time and kill-on-close limits; cancellation terminates the job. Failure to construct or verify containment returns `script.isolation_unavailable`.

Linux and macOS may compile the GUI and test platform-neutral protocol/engine code, but script execution must return `script.isolation_unavailable` until an equivalent launcher exists. There is no in-process or weaker subprocess fallback.

Only pure script computation may be replayed. Host effects must later use planned action → execute → durable receipt, and ambiguous worker death is never blindly retried. Journal stable codes, digests and counters; never persist source, input, output, credentials, or other sensitive payloads.
