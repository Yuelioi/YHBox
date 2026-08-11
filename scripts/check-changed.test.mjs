import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

import { planChecks } from './check-changed.mjs'
import { affectedGoPackages } from './check-go-changed.mjs'

test('routes only checks relevant to changed files', () => {
  assert.deepEqual(planChecks(['docs/architecture/README.md']), ['check:docs'])
  assert.deepEqual(planChecks(['frontend/src/views/WorkflowEditorView.vue']), [
    'check:frontend:quick',
  ])
  assert.deepEqual(planChecks(['internal/workflow/compiler/compiler.go']), ['check:go:changed'])
  assert.deepEqual(planChecks(['internal/nodes/extended.go']), [
    'contracts:check',
    'nodes:compatibility:check',
    'check:go:changed',
  ])
	assert.deepEqual(planChecks(['internal/services/schedule/model.go']), [
	  'versions:compatibility:check',
	  'check:bindings',
	  'check:go:changed',
	])
  assert.deepEqual(planChecks(['native/capture_wgc/src/lib.rs']), ['check:rust'])
  assert.deepEqual(planChecks(['.github/workflows/ci.yml']), ['check:supply-chain:actions'])
  assert.deepEqual(planChecks(['pkg/platform/hires_timer_windows_test.go']), [
    'check:go:changed',
    'check:windows:timer',
  ])
})

test('routes documentation and its checker through the stable Markdown gate', () => {
  assert.deepEqual(planChecks(['README.md']), ['check:docs'])
  assert.deepEqual(planChecks(['scripts/check-docs.mjs']), ['check:docs'])
  assert.deepEqual(planChecks(['Taskfile.yml']), ['check:changed:self', 'check:docs'])
})

test('mixed Go and frontend work does not pull in unrelated Rust or supply-chain checks', () => {
  const checks = planChecks([
    'internal/workflow/compiler/compiler.go',
    'frontend/src/views/WorkflowEditorView.vue',
  ])
  assert.deepEqual(checks, ['check:go:changed', 'check:frontend:quick'])
  assert.ok(!checks.includes('check:rust'))
  assert.ok(!checks.some((check) => check.startsWith('check:supply-chain')))
})

test('Go routing includes reverse dependencies but excludes unrelated packages', () => {
  const root = process.cwd()
  const packages = [
    { ImportPath: 'example/compiler', Dir: `${root}/internal/compiler`, Imports: [] },
    { ImportPath: 'example/application', Dir: `${root}/internal/application`, Imports: ['example/compiler'] },
    { ImportPath: 'example/service', Dir: `${root}/internal/service`, TestImports: ['example/application'] },
    { ImportPath: 'example/unrelated', Dir: `${root}/internal/unrelated`, Imports: [] },
  ]
  assert.deepEqual(affectedGoPackages(['internal/compiler/compiler.go'], packages, root), [
    'example/application',
    'example/compiler',
    'example/service',
  ])
})

test('keeps full validation explicit for CI and packaging', () => {
  const taskfile = readFileSync(new URL('../Taskfile.yml', import.meta.url), 'utf8')
  const ci = readFileSync(new URL('../.github/workflows/ci.yml', import.meta.url), 'utf8')
  const checkBody = taskfile.match(/^  check:\r?\n([\s\S]*?)(?=^  \S)/m)?.[1] ?? ''
  const fullBody = taskfile.match(/^  check:full:\r?\n([\s\S]*?)(?=^  \S)/m)?.[1] ?? ''
  const packageBody = taskfile.match(/^  package:\r?\n([\s\S]*?)(?=^  \S)/m)?.[1] ?? ''

  assert.match(checkBody, /scripts\/check-changed\.mjs/)
  assert.doesNotMatch(checkBody, /check:supply-chain|check:go|check:frontend/)
  assert.match(fullBody, /check:supply-chain/)
  assert.match(fullBody, /check:docs/)
  assert.match(fullBody, /check:go:full/)
  assert.match(fullBody, /check:windows:timer/)
  assert.match(fullBody, /check:frontend:full/)
  assert.match(packageBody, /check:full/)
  assert.match(ci, /task check:full/)
})
