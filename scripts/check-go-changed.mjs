import { spawnSync } from 'node:child_process'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { changedFiles } from './check-changed.mjs'

function run(command, args, options = {}) {
  const result = spawnSync(command, args, { stdio: 'inherit', ...options })
  if (result.error) throw result.error
  if (result.status !== 0) process.exit(result.status ?? 1)
}

function packageInventory() {
  const format = '{{.ImportPath}}\t{{.Dir}}\t{{join .Imports ","}}\t{{join .TestImports ","}}\t{{join .XTestImports ","}}'
  const result = spawnSync('go', ['list', '-f', format, './...'], { encoding: 'utf8' })
  if (result.status !== 0) {
    throw new Error(`go list failed: ${(result.stderr || result.stdout).trim()}`)
  }
  return result.stdout
    .split(/\r?\n/)
    .filter(Boolean)
    .map((line) => {
      const [ImportPath, Dir, imports = '', testImports = '', xTestImports = ''] = line.split('\t')
      const splitImports = (value) => (value ? value.split(',') : [])
      return {
        ImportPath,
        Dir,
        Imports: splitImports(imports),
        TestImports: splitImports(testImports),
        XTestImports: splitImports(xTestImports),
      }
    })
}

export function affectedGoPackages(files, packages, root = process.cwd()) {
  if (files.some((path) => path === 'go.mod' || path === 'go.sum')) {
    return packages.map((pkg) => pkg.ImportPath).sort()
  }
  const changedDirectories = new Set(
    files.filter((path) => path.endsWith('.go')).map((path) => resolve(root, dirname(path)).toLowerCase()),
  )
  const selected = new Set(
    packages
      .filter((pkg) => changedDirectories.has(resolve(pkg.Dir).toLowerCase()))
      .map((pkg) => pkg.ImportPath),
  )
  let grew = true
  while (grew) {
    grew = false
    for (const pkg of packages) {
      if (selected.has(pkg.ImportPath)) continue
      const imports = [...(pkg.Imports ?? []), ...(pkg.TestImports ?? []), ...(pkg.XTestImports ?? [])]
      if (imports.some((dependency) => selected.has(dependency))) {
        selected.add(pkg.ImportPath)
        grew = true
      }
    }
  }
  return [...selected].sort()
}

function main() {
  const files = changedFiles()
  const packages = affectedGoPackages(files, packageInventory())
  if (packages.length === 0) {
    console.log('No changed Go package was found.')
    return
  }
  console.log(`Affected Go packages (${packages.length}):\n${packages.join('\n')}`)
  run('go', ['test', '-count=1', ...packages], { env: { ...process.env, GOTOOLCHAIN: 'local', GOFLAGS: '-mod=readonly' } })
  run('go', ['vet', ...packages], { env: { ...process.env, GOTOOLCHAIN: 'local', GOFLAGS: '-mod=readonly' } })
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) main()
