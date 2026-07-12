import { readFile, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { ESLint } from 'eslint'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const baselinePath = resolve(frontendRoot, 'lint-baseline.json')
const writeMode = process.argv.includes('--write')
const baseline = JSON.parse(await readFile(baselinePath, 'utf8'))

if (baseline.schemaVersion !== 1 || typeof baseline.rules !== 'object') {
  fail(`unsupported ESLint baseline schema in ${baselinePath}`)
}

const eslint = new ESLint({
  cwd: frontendRoot,
  cache: true,
  cacheLocation: resolve(frontendRoot, '.eslintcache'),
})
const results = await eslint.lintFiles(['.'])
const counts = Object.fromEntries(Object.keys(baseline.rules).map((rule) => [rule, 0]))
const unexpected = []

for (const result of results) {
  const messages = []
  for (const message of result.messages) {
    if (message.ruleId && message.ruleId in counts) counts[message.ruleId] += 1
    else messages.push(message)
  }
  if (messages.length > 0) unexpected.push({ ...result, messages })
}

if (unexpected.length > 0) {
  const formatter = await eslint.loadFormatter('stylish')
  console.error(await formatter.format(unexpected))
  fail('ESLint found violations outside the tracked debt baseline')
}

if (writeMode) {
  const next = `${JSON.stringify({ ...baseline, rules: counts }, null, 2)}\n`
  await writeFile(baselinePath, next)
  console.log(`updated ESLint debt baseline: ${JSON.stringify(counts)}`)
  process.exit(0)
}

const drift = Object.entries(counts).filter(([rule, count]) => count !== baseline.rules[rule])
if (drift.length > 0) {
  for (const [rule, count] of drift) {
    console.error(`${rule}: expected ${baseline.rules[rule]}, found ${count}`)
  }
  fail(
    'ESLint debt baseline changed; remove regressions or run pnpm lint:baseline:update after review',
  )
}

console.log(`ESLint OK; tracked debt unchanged: ${JSON.stringify(counts)}`)

function fail(message) {
  console.error(message)
  process.exit(1)
}
