---
version: 3.0
disabled_folders: ["archive"]    # the one structured toggle: listed folders never suggested / not flagged as orphans
---

## House rules

<!-- Deck-local flightdeck conventions + behavioral overrides. rules.md is mandatory — do not delete it.
     Defaults (override below): commit confirm-gated (asks Y/n); preflight/walkaround/emit-agents-md/status
     self-invoke; LANDING is manual (it archives + commits — opt in to auto); status auto-starts (idea→active)
     but does NOT auto-archive on done; git & emit inferred from .git / AGENTS.md presence; bundled scripts off.
     So out of the box nothing archives or commits without you. Tune everything via ### Autonomy overrides below.
     General project conventions belong in CLAUDE.md/AGENTS.md, not here. -->

### Project conventions

### Autonomy overrides
<!-- Omit a line = keep the default. To change a behavior, UNCOMMENT its phrase (one per line). -->

<!-- ── make me MORE autonomous (enabled) ── -->

landing: self-invoke
status: auto land
commit without asking
run scripts with uv

<!-- ── make me LESS autonomous (disabled) ── -->

<!-- ── environment (auto detect) ── -->