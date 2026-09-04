---
title: Updates and Backups
description: Upgrade alpha builds without losing local work
slug: updates-and-backups
order: 95
source_id: yotta-updates-backups
---

# Updates and Backups

Before updating: stop Runs and recording, wait for settings to save, export important workflows,
close Yotta completely, then back up the full profile directory.

Windows default profile:

```text
%LOCALAPPDATA%\Yotta\Yotta
```

Do not copy only a live database file; WAL data may still be active. Exit first and copy the complete
directory.

Extract each app version into a clean program directory instead of mixing old and new files. Keep the
old package until the new version opens settings, workflows, and resources correctly.

`4.0.0-alpha.N` is the public prerelease version. Windows numeric metadata uses the fourth segment,
so `4.0.0-alpha.2` maps to `4.0.0.2`.

Do not manually delete object storage, databases, migration files, or runtime locks. Shared content
may still be referenced by workflows.
