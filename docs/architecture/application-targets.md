# Application targets

Desktop application automation is a Configured Target. Settings stores a slot, label, absolute executable path and argument list. There is no application capability, authorized profile, executable inspection, SHA-256 identity, consent, grant, credential binding or TTL.

The composition root places the configured provider in the same immutable target snapshot used by Network and Automation Targets. Every Run resolves the slot and directly invokes the selected operation through `internal/targetruntime`. Editing the path or arguments affects later Runs without requiring re-authorization or restarting Yotta.

`Launch configured application` starts the configured path with the configured argv, executable directory as working directory, and the complete inherited environment. It accepts any absolute executable path supported by the host; it does not restrict extensions, shells, script hosts, Yotta itself, proxies or environment variables.

`Terminate configured application` enumerates processes and terminates those whose current executable resolves to the same OS file as the configured path. The node returns `terminated-count`; the match is the meaning of the terminate operation, not an identity pin or admission check. Launch and terminate keep their typed node ports and operation payloads so graph execution remains deterministic.

Windows provides the current implementation. Linux and macOS expose authoring contracts but do not configure this provider until their platform implementations exist. Plugin/Process Node isolation is a separate runtime concern and does not participate in Application Target invocation.
