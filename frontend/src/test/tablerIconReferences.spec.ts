import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

interface IconifyCollection {
  icons: Record<string, unknown>
  aliases?: Record<string, unknown>
}

function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) return sourceFiles(path)
    return /\.(?:ts|vue)$/.test(entry.name) ? [path] : []
  })
}

describe('literal Tabler icon references', () => {
  it('bundles production icon references for synchronous first paint', () => {
    const config = readFileSync(join(process.cwd(), 'vite.config.ts'), 'utf8')

    expect(config).toContain("globInclude: ['src/**/*.{vue,ts}']")
    expect(config).toContain("globExclude: ['src/test/**', 'src/**/*.spec.ts', 'src/**/*.test.ts']")
    expect(config).toContain('icons: catalogIconNames')
  })

  it('only uses icons included in the installed collection', () => {
    const collection = JSON.parse(
      readFileSync(join(process.cwd(), 'node_modules/@iconify-json/tabler/icons.json'), 'utf8'),
    ) as IconifyCollection
    const available = new Set([
      ...Object.keys(collection.icons),
      ...Object.keys(collection.aliases ?? {}),
    ])
    const missing: string[] = []

    for (const file of sourceFiles(join(process.cwd(), 'src'))) {
      const source = readFileSync(file, 'utf8')
      for (const match of source.matchAll(/i-tabler-([a-z0-9-]+)/g)) {
        if (!available.has(match[1])) {
          missing.push(`${relative(process.cwd(), file)}: i-tabler-${match[1]}`)
        }
      }
    }

    expect(missing).toEqual([])
  })

  it('only publishes Catalog node icons included in the installed collection', () => {
    const collection = JSON.parse(
      readFileSync(join(process.cwd(), 'node_modules/@iconify-json/tabler/icons.json'), 'utf8'),
    ) as IconifyCollection
    const available = new Set([
      ...Object.keys(collection.icons),
      ...Object.keys(collection.aliases ?? {}),
    ])
    const authoring = JSON.parse(
      readFileSync(join(process.cwd(), '../contracts/node/current/builtin-authoring.json'), 'utf8'),
    ) as {
      body: { nodes: Array<{ nodeRef: { nodeTypeId: string }; icon?: string }> }
    }

    expect(
      authoring.body.nodes
        .filter((node) => node.icon && !available.has(node.icon))
        .map((node) => `${node.nodeRef.nodeTypeId}: ${node.icon}`),
    ).toEqual([])
  })
})
