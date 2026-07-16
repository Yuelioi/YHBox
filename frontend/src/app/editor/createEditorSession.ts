import { shallowReactive } from 'vue'
import type { WorkflowTransport } from '@/app/transport/workflow31'
import { EditorSession } from './EditorSession'

export function createEditorSession(
  transport: WorkflowTransport,
  idFactory?: () => string,
): EditorSession {
  return shallowReactive(new EditorSession(transport, idFactory)) as EditorSession
}
