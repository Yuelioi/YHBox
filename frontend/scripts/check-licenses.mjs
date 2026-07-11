import { readFileSync } from 'node:fs'

const reportPath = process.argv[2]
if (!reportPath) throw new Error('usage: node check-licenses.mjs <pnpm-license-report.json>')

const report = JSON.parse(readFileSync(reportPath, 'utf8'))
const allowed = new Set([
  '0BSD',
  'Apache-2.0',
  'BlueOak-1.0.0',
  'BSD-2-Clause',
  'BSD-3-Clause',
  'CC0-1.0',
  'EPL-2.0',
  'ISC',
  'MIT',
  'MPL-2.0',
  'OFL-1.1',
  'Python-2.0',
])

const violations = Object.entries(report)
  .filter(([license]) => license !== 'Unknown' && !allowed.has(license))
  .flatMap(([license, packages]) => packages.map((pkg) => `${pkg.name}@${pkg.versions.join(',')} (${license})`))

// vaul-vue 0.4.1 omits license metadata from its npm tarball. Its upstream
// LICENSE is MIT: https://github.com/unovue/vaul-vue/blob/main/LICENSE
// Pin the exception to one version so any dependency update forces re-review.
for (const pkg of report.Unknown ?? []) {
  if (pkg.name !== 'vaul-vue' || pkg.versions.length !== 1 || pkg.versions[0] !== '0.4.1') {
    violations.push(`${pkg.name}@${pkg.versions.join(',')} (Unknown)`)
  }
}

if (violations.length) {
  throw new Error(`disallowed or unknown frontend licenses:\n${violations.join('\n')}`)
}

console.log('frontend production dependency licenses OK')
