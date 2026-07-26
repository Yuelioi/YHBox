import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const viteConfig = readFileSync(join(process.cwd(), 'vite.config.ts'), 'utf8')

describe('Nuxt UI button alignment baseline', () => {
  it('centers button contents globally so fixed-size icon buttons cannot drift', () => {
    expect(viteConfig).toMatch(/button:\s*{[\s\S]*?slots:\s*{\s*base:\s*'justify-center'\s*}/)
  })
})
