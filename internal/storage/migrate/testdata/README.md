# Development storage fixtures

`layout-1` is an immutable pre-4.0 development compatibility input. Tests copy
it as bytes and migrate the copy. Do not update these files with current
serializers when later layouts change.

Layout 1 predates the authoritative Content Catalog and Run Ledger databases,
so their absence is part of the fixture.

`invalid-legacy-run` freezes one valid legacy ownership marker and one malformed
record used to exercise quarantine and recovery surfaces.

These fixtures prove the current one-time owner-profile upgrade path. They do
not establish a public compatibility promise; the public floor starts at 4.0.
