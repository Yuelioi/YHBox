import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const taskfile = readFileSync(new URL('../build/Taskfile.yml', import.meta.url), 'utf8')
const windowsTaskfile = readFileSync(new URL('../build/windows/Taskfile.yml', import.meta.url), 'utf8')
const frontendTask = taskfile.match(/  build:frontend:\r?\n([\s\S]*?)\r?\n\r?\n  frontend:vendor:/)?.[1] ?? ''
const sources = frontendTask.match(/    sources:\r?\n([\s\S]*?)    generates:/)?.[1] ?? ''

test('frontend build incrementality tracks source inputs without consuming dist outputs', () => {
  assert.match(sources, /- "src\/\*\*\/\*"/)
  assert.match(sources, /- "public\/\*\*\/\*"/)
  assert.doesNotMatch(sources, /- "\*\*\/\*"/)
  assert.doesNotMatch(sources, /dist\/\*\*\/\*/)
})

test('Windows build cleanup does not load user PowerShell profiles or use a wildcard target', () => {
  assert.doesNotMatch(windowsTaskfile, /powershell\s+Remove-item/i)
  assert.match(windowsTaskfile, /powershell -NoProfile -Command "Remove-Item -LiteralPath/)
  assert.doesNotMatch(windowsTaskfile, /Remove-Item[^\r\n]*\*\.syso/i)
})
