# Bind capabilities to both Program and Run

Yotta separates compile-time Capability Requirements from runtime authorization: the Compiler freezes an attributed least-privilege Capability Plan into the Program, while the host policy issues a short-lived opaque Run Grant bound to that exact Program/plan, principal, provider, target, operations and scope. We reject ambient `ServiceBundle` access and bearer-token passthrough because a Program declaration cannot authorize itself, while authorizing only at dispatch loses the reviewable permission delta and enables confused-deputy reuse.
