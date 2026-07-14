import { spawn } from 'node:child_process'
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createRequire } from 'node:module'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const outputRoot = resolve(root, 'contracts/workflow/v3')
const nodeOutputRoot = resolve(root, 'contracts/node/3.1')
const writeMode = process.argv.includes('--write')
const temporary = await mkdtemp(resolve(tmpdir(), 'yotta-contracts-'))
const temporarySchema = resolve(temporary, 'workflow-source.schema.json')
const temporaryDiagnostic = resolve(temporary, 'diagnostic.schema.json')
const temporaryNodeSchema = resolve(temporary, 'node-contract.schema.json')
const require = createRequire(resolve(root, 'frontend/package.json'))
const { compile } = require('json-schema-to-typescript')

try {
  await run('go', ['run', './cmd/yotta-contracts', '-output', temporarySchema])
  await run('go', [
    'run',
    './cmd/yotta-contracts',
    '-contract',
    'diagnostic',
    '-output',
    temporaryDiagnostic,
  ])
  await run('go', [
    'run',
    './cmd/yotta-contracts',
    '-contract',
    'node',
    '-output',
    temporaryNodeSchema,
  ])
  const schemaText = await readFile(temporarySchema, 'utf8')
  const diagnosticSchemaText = await readFile(temporaryDiagnostic, 'utf8')
  const nodeSchemaText = await readFile(temporaryNodeSchema, 'utf8')
  const schema = JSON.parse(schemaText)
  const diagnosticSchema = JSON.parse(diagnosticSchemaText)
  const nodeSchema = JSON.parse(nodeSchemaText)
  const options = {
    bannerComment: '/* Generated from WorkflowSource Go types. Do not edit. */',
    style: { singleQuote: true, semi: false },
    unknownAny: false,
  }
  const sourceTypes = await compile(schema, 'WorkflowSource', options)
  const diagnosticTypes = await compile(diagnosticSchema, 'Diagnostic', {
    ...options,
    bannerComment: '/* Generated from Diagnostic Go types. Do not edit. */',
  })
  const nodeTypes = await compile(nodeSchema, 'NodeContract', {
    ...options,
    bannerComment: '/* Generated from Node Contract 3.1 Go types. Do not edit. */',
  })
  const files = new Map([
    ['workflow-source.schema.json', schemaText],
    ['diagnostic.schema.json', diagnosticSchemaText],
    ['workflow-source.ts', sourceTypes],
    ['diagnostic.ts', diagnosticTypes],
  ])
  const nodeFiles = new Map([
    ['node-contract.schema.json', nodeSchemaText],
    ['node-contract.ts', nodeTypes],
  ])

  if (writeMode) {
    await mkdir(outputRoot, { recursive: true })
    for (const [name, content] of files) await writeFile(resolve(outputRoot, name), content)
    await mkdir(nodeOutputRoot, { recursive: true })
    for (const [name, content] of nodeFiles) await writeFile(resolve(nodeOutputRoot, name), content)
    console.log(`updated ${files.size} Workflow v3 and ${nodeFiles.size} Node Contract 3.1 files`)
  } else {
    for (const [name, generated] of files) {
      let tracked
      try { tracked = await readFile(resolve(outputRoot, name), 'utf8') }
      catch (error) { fail(`missing ${name}: ${error.message}\nRun task contracts:update.`) }
      if (tracked !== generated) fail(`${name} differs from generated contract. Run task contracts:update and review the diff.`)
    }
    for (const [name, generated] of nodeFiles) {
      let tracked
      try { tracked = await readFile(resolve(nodeOutputRoot, name), 'utf8') }
      catch (error) { fail(`missing ${name}: ${error.message}\nRun task contracts:update.`) }
      if (tracked !== generated) fail(`${name} differs from generated contract. Run task contracts:update and review the diff.`)
    }
    console.log(`Workflow v3 contracts OK: ${files.size} files; Node Contract 3.1: ${nodeFiles.size} files`)
  }
} finally {
  await rm(temporary, { recursive: true, force: true })
}

function run(command, args) {
  return new Promise((resolvePromise, reject) => {
    const child = spawn(command, args, { cwd: root, stdio: 'inherit', env: process.env })
    child.on('error', reject)
    child.on('close', (code) => code === 0 ? resolvePromise() : reject(new Error(`${command} exited with ${code}`)))
  })
}

function fail(message) {
  console.error(message)
  process.exitCode = 1
  throw new Error(message)
}
