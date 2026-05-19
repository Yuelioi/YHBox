// D2 parity helper test: dumps frontend registry to a known file so the
// backend Go parity test can read + diff against its own ExportJSON.
//
// The path is chosen so the same file can be read from Go without a build
// step: `<repo>/internal/services/container/nodekind/testdata/fe-registry.json`.
//
// Skipping behavior: if SKIP_PARITY_DUMP env is set (e.g. CI step that runs
// vitest in parallel and doesn't want the file written), the test no-ops.
import { describe, it, expect } from 'vitest'
import { writeFileSync, mkdirSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import './specs' // populate registry
import { exportJSON } from './export'
import { allKinds } from './registry'

describe('D2 parity dump', () => {
  it('writes frontend registry dump to testdata for Go parity test', () => {
    if (process.env.SKIP_PARITY_DUMP) {
      return
    }
    const json = exportJSON()
    // Sanity: registry populated.
    expect(allKinds().length).toBeGreaterThanOrEqual(60)
    expect(json).toContain('"kind": "Sleep"')
    expect(json).toContain('"kind": "GetVar"')

    // Path is relative to the vitest cwd (frontend/). Reach into repo root then
    // into Go testdata directory. Mkdir -p to handle first-run case.
    const repoRoot = resolve(__dirname, '..', '..', '..', '..', '..')
    const outPath = resolve(
      repoRoot,
      'internal/services/container/nodekind/testdata/fe-registry.json',
    )
    mkdirSync(dirname(outPath), { recursive: true })
    writeFileSync(outPath, json + '\n', 'utf8')
  })
})
