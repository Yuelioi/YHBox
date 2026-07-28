import { describe, expect, it } from 'vitest'
import { builtinNodeProjections } from './node'

const generateID = 'https://schemas.yotta.dev/nodes/ai/generate'
const extractID = 'https://schemas.yotta.dev/nodes/ai/extract'
const removedAgentID = 'https://schemas.yotta.dev/nodes/ai/agent'

describe('AI node authoring projection', () => {
  it('keeps only the two user-facing AI tasks', () => {
    const aiNodes = [...builtinNodeProjections.values()].filter((node) => node.category === 'ai')
    expect(aiNodes.map((node) => node.nodeRef.nodeTypeId).sort()).toEqual([extractID, generateID])
    expect(builtinNodeProjections.has(removedAgentID)).toBe(false)
  })

  it('uses multiline prompts and a friendly structured-field editor', () => {
    for (const nodeID of [generateID, extractID]) {
      const node = builtinNodeProjections.get(nodeID)
      const prompt = node?.dataInputs.find((port) => port.id === 'prompt')
      const image = node?.dataInputs.find((port) => port.id === 'image')
      expect(prompt?.editorAdapter).toBe('multiline-text')
      expect(image?.binding).toBe('optional')
    }

    const fields = builtinNodeProjections
      .get(extractID)
      ?.configFields.find((field) => field.id === 'fields')
    expect(fields?.editorAdapter).toBe('structured-output-fields')
    expect(fields?.control).toBe('list')
    expect(
      builtinNodeProjections.get(extractID)?.configFields.some((field) => field.id === 'schema'),
    ).toBe(false)
  })
})
