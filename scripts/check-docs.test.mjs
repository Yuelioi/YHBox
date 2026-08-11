import assert from 'node:assert/strict'
import test from 'node:test'

import { isExternalTarget, localLinkTargets, taskNames } from './check-docs.mjs'

test('extracts local Markdown links without images or external special cases changing the target', () => {
  assert.deepEqual(
    localLinkTargets('[guide](docs/README.md) ![shot](preview.png) [web](https://example.com/a)'),
    ['docs/README.md', 'preview.png', 'https://example.com/a'],
  )
  assert.equal(isExternalTarget('https://example.com/a'), true)
  assert.equal(isExternalTarget('mailto:security@example.com'), true)
  assert.equal(isExternalTarget('../README.md'), false)
})

test('extracts exact Task commands from prose and code blocks', () => {
  assert.deepEqual(taskNames('Run `task check` then:\n```powershell\ntask windows:smoke:automation\n```'), [
    'check',
    'windows:smoke:automation',
  ])
})
