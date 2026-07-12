import { readFile } from 'node:fs/promises'
import { gzipSync } from 'node:zlib'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const distRoot = resolve(frontendRoot, 'dist')
const manifestPath = resolve(distRoot, '.vite/manifest.json')
const budgetsPath = resolve(frontendRoot, 'bundle-budgets.json')

const [manifest, budgets] = await Promise.all([
  readJSON(manifestPath, 'Vite manifest'),
  readJSON(budgetsPath, 'bundle budgets'),
])

if (budgets.schemaVersion !== 1 || budgets.units !== 'decimal bytes') {
  fail(`unsupported bundle budget schema in ${budgetsPath}`)
}

const entryKeys = collectSynchronousJavaScript(manifest, budgets.entry.manifestKey)
const editorKeys = collectSynchronousJavaScript(manifest, budgets.editor.manifestKey)
for (const key of entryKeys) editorKeys.delete(key)

const entry = await measure('entry', entryKeys, manifest, budgets.gzipLevel)
const editor = await measure('editor', editorKeys, manifest, budgets.gzipLevel)
const iconsKey = uniqueKeyBySuffix(manifest, budgets.reportOnly.tablerIconsManifestSuffix)
const icons = await measure(
  'tabler-icons-report-only',
  new Set([iconsKey]),
  manifest,
  budgets.gzipLevel,
)

const checks = [
  {
    name: entry.name,
    actualGzipBytes: entry.gzipBytes,
    limitGzipBytes: budgets.entry.maxInitialGzipBytes,
    passed: entry.gzipBytes <= budgets.entry.maxInitialGzipBytes,
  },
  {
    name: editor.name,
    actualGzipBytes: editor.gzipBytes,
    limitGzipBytes: budgets.editor.maxInitialGzipBytes,
    targetGzipBytes: budgets.editor.targetInitialGzipBytes,
    passed: editor.gzipBytes <= budgets.editor.maxInitialGzipBytes,
  },
]

const report = {
  schemaVersion: budgets.schemaVersion,
  units: budgets.units,
  gzipLevel: budgets.gzipLevel,
  checks,
  measurements: [entry, editor, icons],
}

console.log(JSON.stringify(report, null, 2))

const failures = checks.filter((check) => !check.passed)
if (failures.length > 0) {
  for (const failure of failures) {
    console.error(
      `${failure.name} exceeds its gzip budget by ${failure.actualGzipBytes - failure.limitGzipBytes} bytes`,
    )
  }
  process.exit(1)
}

async function readJSON(path, label) {
  try {
    return JSON.parse(await readFile(path, 'utf8'))
  } catch (error) {
    fail(`cannot read ${label} at ${path}: ${error.message}`)
  }
}

function collectSynchronousJavaScript(manifest, rootKey) {
  if (!manifest[rootKey]) fail(`manifest key not found: ${rootKey}`)

  const result = new Set()
  const visited = new Set()
  const pending = [rootKey]
  while (pending.length > 0) {
    const key = pending.pop()
    if (visited.has(key)) continue
    visited.add(key)

    const item = manifest[key]
    if (!item) fail(`manifest import not found: ${key}`)
    if (item.file?.endsWith('.js')) result.add(key)
    for (const imported of item.imports ?? []) pending.push(imported)
  }
  return result
}

function uniqueKeyBySuffix(manifest, suffix) {
  const matches = Object.keys(manifest).filter((key) => key.replaceAll('\\', '/').endsWith(suffix))
  if (matches.length !== 1) {
    fail(`expected one manifest key ending in ${suffix}, found ${matches.length}`)
  }
  return matches[0]
}

async function measure(name, keys, manifest, gzipLevel) {
  const files = []
  for (const key of [...keys].sort()) {
    const file = manifest[key]?.file
    if (!file) fail(`manifest entry has no output file: ${key}`)
    const bytes = await readFile(resolve(distRoot, file))
    files.push({
      manifestKey: key,
      file,
      rawBytes: bytes.byteLength,
      gzipBytes: gzipSync(bytes, { level: gzipLevel }).byteLength,
    })
  }
  return {
    name,
    rawBytes: files.reduce((total, file) => total + file.rawBytes, 0),
    gzipBytes: files.reduce((total, file) => total + file.gzipBytes, 0),
    files,
  }
}

function fail(message) {
  console.error(message)
  process.exit(1)
}
