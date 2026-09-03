import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { mkdtempSync, mkdirSync, statSync, utimesSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { isFrontendBuildCurrent } from './frontend-build-current.mjs'

const taskfile = readFileSync(new URL('../build/Taskfile.yml', import.meta.url), 'utf8')
const windowsTaskfile = readFileSync(new URL('../build/windows/Taskfile.yml', import.meta.url), 'utf8')
const frontendTask = taskfile.match(/  build:frontend:\r?\n([\s\S]*?)\r?\n\r?\n  frontend:vendor:/)?.[1] ?? ''
const sources = frontendTask.match(/    sources:\r?\n([\s\S]*?)    generates:/)?.[1] ?? ''

test('frontend build incrementality tracks source inputs without consuming dist outputs', () => {
  assert.match(sources, /- "src\/\*\*\/\*"/)
  assert.match(sources, /- "public\/\*\*\/\*"/)
  assert.doesNotMatch(sources, /- "\*\*\/\*"/)
  assert.doesNotMatch(sources, /dist\/\*\*\/\*/)
  assert.match(frontendTask, /frontend-build-current\.mjs/)
  assert.match(frontendTask, /dev\{\{else\}\}production/)
})

test('frontend build status becomes stale when a source is newer than the mode marker', () => {
  const root = mkdtempSync(join(tmpdir(), 'yotta-frontend-build-'))
  mkdirSync(join(root, 'src'), { recursive: true })
  mkdirSync(join(root, 'dist'), { recursive: true })
  writeFileSync(join(root, 'src', 'app.ts'), 'old')
  writeFileSync(join(root, 'dist', '.yotta-build-mode'), 'production')
  assert.equal(isFrontendBuildCurrent(root, 'production'), true)

  const markerTime = statSync(join(root, 'dist', '.yotta-build-mode')).mtimeMs
  writeFileSync(join(root, 'src', 'app.ts'), 'new')
  utimesSync(join(root, 'src', 'app.ts'), (markerTime + 2_000) / 1_000, (markerTime + 2_000) / 1_000)
  assert.equal(isFrontendBuildCurrent(root, 'production'), false)
  assert.equal(isFrontendBuildCurrent(root, 'dev'), false)
})

test('Windows build cleanup does not load user PowerShell profiles or use a wildcard target', () => {
  assert.doesNotMatch(windowsTaskfile, /powershell\s+Remove-item/i)
  assert.match(windowsTaskfile, /powershell -NoProfile -Command "Remove-Item -LiteralPath/)
  assert.doesNotMatch(windowsTaskfile, /Remove-Item[^\r\n]*\*\.syso/i)
})
