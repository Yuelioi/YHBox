import { spawn } from 'node:child_process'
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createRequire } from 'node:module'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const writeMode = process.argv.includes('--write')
const temporary = await mkdtemp(resolve(tmpdir(), 'yotta-contracts-'))
const temporaryMetadata = resolve(temporary, 'metadata.json')
const temporarySchema = resolve(temporary, 'workflow-source.schema.json')
const temporaryDiagnostic = resolve(temporary, 'diagnostic.schema.json')
const temporaryWorkflowAuthoring = resolve(temporary, 'authoring-patch.schema.json')
const temporaryNodeSchema = resolve(temporary, 'node-contract.schema.json')
const temporaryAuthoringSchema = resolve(temporary, 'authoring-projection.schema.json')
const temporaryBuiltinCatalog = resolve(temporary, 'builtin-catalog.json')
const temporaryBuiltinAuthoring = resolve(temporary, 'builtin-authoring.json')
const temporaryBuiltinDocs = resolve(temporary, 'builtin-nodes.md')
const require = createRequire(resolve(root, 'frontend/package.json'))
const { compile } = require('json-schema-to-typescript')

try {
  await run('go', ['run', './cmd/yotta-contracts', '-contract', 'metadata', '-output', temporaryMetadata])
  const metadata = JSON.parse(await readFile(temporaryMetadata, 'utf8'))
  const outputRoot = resolve(root, 'contracts/workflow', metadata.workflow.pathVersion)
  const nodeOutputRoot = resolve(root, 'contracts/node', metadata.node.pathVersion)
  const workflowCurrentRoot = resolve(root, 'contracts/workflow/current')
  const nodeCurrentRoot = resolve(root, 'contracts/node/current')
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
    'workflow-authoring',
    '-output',
    temporaryWorkflowAuthoring,
  ])
  await run('go', [
    'run',
    './cmd/yotta-contracts',
    '-contract',
    'node',
    '-output',
    temporaryNodeSchema,
  ])
  await run('go', [
    'run',
    './cmd/yotta-contracts',
    '-contract',
    'authoring',
    '-output',
    temporaryAuthoringSchema,
  ])
  for (const [contract, output] of [
    ['builtin-catalog', temporaryBuiltinCatalog],
    ['builtin-authoring', temporaryBuiltinAuthoring],
    ['builtin-docs', temporaryBuiltinDocs],
  ]) {
    await run('go', ['run', './cmd/yotta-contracts', '-contract', contract, '-output', output])
  }
  const schemaText = await readFile(temporarySchema, 'utf8')
  const diagnosticSchemaText = await readFile(temporaryDiagnostic, 'utf8')
  const workflowAuthoringSchemaText = await readFile(temporaryWorkflowAuthoring, 'utf8')
  const nodeSchemaText = await readFile(temporaryNodeSchema, 'utf8')
  const authoringSchemaText = await readFile(temporaryAuthoringSchema, 'utf8')
  const builtinCatalogText = await readFile(temporaryBuiltinCatalog, 'utf8')
  const builtinAuthoringText = await readFile(temporaryBuiltinAuthoring, 'utf8')
  const builtinDocsText = await readFile(temporaryBuiltinDocs, 'utf8')
  const schema = JSON.parse(schemaText)
  const diagnosticSchema = JSON.parse(diagnosticSchemaText)
  const workflowAuthoringSchema = JSON.parse(workflowAuthoringSchemaText)
  const nodeSchema = JSON.parse(nodeSchemaText)
  const authoringSchema = JSON.parse(authoringSchemaText)
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
  const workflowAuthoringTypes = await compile(workflowAuthoringSchema, 'WorkflowPatch', {
    ...options,
    bannerComment: '/* Generated from current Workflow Authoring Patch Go types. Do not edit. */',
  })
  const nodeTypes = await compile(nodeSchema, 'NodeContract', {
    ...options,
    bannerComment: '/* Generated from current Node Contract Go types. Do not edit. */',
  })
  const authoringTypes = await compile(authoringSchema, 'NodeAuthoringProjection', {
    ...options,
    bannerComment: '/* Generated from current Node Authoring Projection Go types. Do not edit. */',
  })
  const files = new Map([
    ['workflow-source.schema.json', schemaText],
    ['diagnostic.schema.json', diagnosticSchemaText],
    ['workflow-source.ts', sourceTypes],
    ['diagnostic.ts', diagnosticTypes],
    ['authoring-patch.schema.json', workflowAuthoringSchemaText],
    ['authoring-patch.ts', workflowAuthoringTypes],
  ])
  const nodeContractFiles = new Map([
    ['node-contract.schema.json', nodeSchemaText],
    ['node-contract.ts', nodeTypes],
    ['authoring-projection.schema.json', authoringSchemaText],
    ['authoring-projection.ts', authoringTypes],
  ])
  const nodeSnapshotFiles = new Map([
    ['builtin-catalog.json', builtinCatalogText],
    ['builtin-authoring.json', builtinAuthoringText],
    ['builtin-nodes.md', builtinDocsText],
  ])
  const nodeFiles = new Map([...nodeContractFiles, ...nodeSnapshotFiles])
  const workflowCurrentFiles = new Map([
    ['workflow-source.ts', `export * from '../${metadata.workflow.pathVersion}/workflow-source'\n`],
    ['diagnostic.ts', `export * from '../${metadata.workflow.pathVersion}/diagnostic'\n`],
    ['authoring-patch.ts', `export * from '../${metadata.workflow.pathVersion}/authoring-patch'\n`],
  ])
  const nodeCurrentFiles = new Map([
    ['node-contract.ts', `export * from '../${metadata.node.pathVersion}/node-contract'\n`],
    ['authoring-projection.ts', `export * from '../${metadata.node.pathVersion}/authoring-projection'\n`],
    ['builtin-authoring.json', builtinAuthoringText],
    ['builtin-authoring.ts', `import document from '../${metadata.node.pathVersion}/builtin-authoring.json'\n\nexport default document\n`],
  ])

  if (writeMode) {
    await mkdir(outputRoot, { recursive: true })
    for (const [name, content] of files) {
      await writeVersionedFile(outputRoot, name, content, 'Workflow')
    }
    await mkdir(nodeOutputRoot, { recursive: true })
    for (const [name, content] of nodeContractFiles) {
      await writeVersionedFile(nodeOutputRoot, name, content, 'Node Contract')
    }
    for (const [name, content] of nodeSnapshotFiles) {
      await writeFile(resolve(nodeOutputRoot, name), content)
    }
    await mkdir(workflowCurrentRoot, { recursive: true })
    for (const [name, content] of workflowCurrentFiles) await writeFile(resolve(workflowCurrentRoot, name), content)
    await mkdir(nodeCurrentRoot, { recursive: true })
    for (const [name, content] of nodeCurrentFiles) await writeFile(resolve(nodeCurrentRoot, name), content)
    console.log(`updated Workflow ${metadata.workflow.pathVersion} and Node Contract ${metadata.node.pathVersion} artifacts`)
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
    for (const [name, generated] of workflowCurrentFiles) {
      let tracked
      try { tracked = await readFile(resolve(workflowCurrentRoot, name), 'utf8') }
      catch (error) { fail(`missing Workflow current/${name}: ${error.message}\nRun task contracts:update.`) }
      if (tracked !== generated) fail(`Workflow current/${name} differs from generated alias. Run task contracts:update.`)
    }
    for (const [name, generated] of nodeCurrentFiles) {
      let tracked
      try { tracked = await readFile(resolve(nodeCurrentRoot, name), 'utf8') }
      catch (error) { fail(`missing Node current/${name}: ${error.message}\nRun task contracts:update.`) }
      if (tracked !== generated) fail(`Node current/${name} differs from generated alias. Run task contracts:update.`)
    }
    console.log(`Workflow ${metadata.workflow.pathVersion} contracts OK: ${files.size} files; Node Contract ${metadata.node.pathVersion}: ${nodeFiles.size} files`)
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

async function writeVersionedFile(outputRoot, name, content, domain) {
  const path = resolve(outputRoot, name)
  try {
    const existing = await readFile(path, 'utf8')
    if (existing !== content) {
      fail(`${domain} artifact ${name} is immutable at this version. Bump the owning contract version before updating it.`)
    }
  } catch (error) {
    if (error?.code !== 'ENOENT') throw error
    await writeFile(path, content)
  }
}
