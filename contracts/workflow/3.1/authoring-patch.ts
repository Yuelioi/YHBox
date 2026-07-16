/* Generated from Workflow Authoring Patch 3.1 Go types. Do not edit. */

export type Command =
  | {
      kind: 'rename-workflow'
      renameWorkflow: RenameWorkflowCommand
    }
  | {
      addStateVariable: AddStateVariableCommand
      kind: 'add-state-variable'
    }
  | {
      kind: 'remove-state-variable'
      removeStateVariable: RemoveStateVariableCommand
    }
  | {
      addNode: AddNodeCommand
      kind: 'add-node'
    }
  | {
      kind: 'remove-node'
      removeNode: NodeCommand
    }
  | {
      kind: 'move-node'
      moveNode: MoveNodeCommand
    }
  | {
      kind: 'set-node-label'
      setNodeLabel: SetNodeLabelCommand
    }
  | {
      kind: 'set-node-disabled'
      setNodeDisabled: SetNodeDisabledCommand
    }
  | {
      kind: 'set-config'
      setConfig: SetConfigCommand
    }
  | {
      clearConfig: FieldCommand
      kind: 'clear-config'
    }
  | {
      bindValue: BindValueCommand
      kind: 'bind-value'
    }
  | {
      bindDefault: PortCommand
      kind: 'bind-default'
    }
  | {
      bindBlob: BindBlobCommand
      kind: 'bind-blob'
    }
  | {
      clearBinding: PortCommand
      kind: 'clear-binding'
    }
  | {
      connect: EdgeCommand
      kind: 'connect'
    }
  | {
      disconnect: EdgeCommand
      kind: 'disconnect'
    }
export type JSONValue =
  | null
  | boolean
  | number
  | string
  | JSONValue[]
  | {
      [k: string]: JSONValue
    }
export type TypeExpression =
  | {
      kind: 'ref'
      ref: TypeRef
    }
  | {
      element: TypeExpression
      kind: 'list'
    }
  | {
      kind: 'union'
      /**
       * @minItems 2
       * @maxItems 64
       */
      members: [TypeExpression, TypeExpression, ...TypeExpression[]]
    }
  | {
      /**
       * @maxItems 32
       */
      constraints?: string[]
      kind: 'variable'
      variable: string
    }

export interface YottaWorkflowAuthoringPatch {
  baseRevision: number
  /**
   * @minItems 1
   * @maxItems 256
   */
  commands: [Command, ...Command[]]
  workflowId: string
}
export interface RenameWorkflowCommand {
  name: string
}
export interface AddStateVariableCommand {
  default: JSONValue
  name: string
  type: TypeExpression
}
export interface TypeRef {
  semanticDigest: string
  typeId: string
}
export interface RemoveStateVariableCommand {
  name: string
}
export interface AddNodeCommand {
  graphId: string
  handle?: string
  nodeTypeId: string
  position: Position
}
export interface Position {
  x: number
  y: number
}
export interface NodeCommand {
  graphId: string
  nodeId: string
}
export interface MoveNodeCommand {
  graphId: string
  nodeId: string
  position: Position
}
export interface SetNodeLabelCommand {
  graphId: string
  label: string
  nodeId: string
}
export interface SetNodeDisabledCommand {
  disabled: boolean
  graphId: string
  nodeId: string
}
export interface SetConfigCommand {
  fieldId: string
  graphId: string
  nodeId: string
  value: JSONValue
}
export interface FieldCommand {
  fieldId: string
  graphId: string
  nodeId: string
}
export interface BindValueCommand {
  graphId: string
  nodeId: string
  portId: string
  value: JSONValue
}
export interface PortCommand {
  graphId: string
  nodeId: string
  portId: string
}
export interface BindBlobCommand {
  blob: BlobRef
  graphId: string
  nodeId: string
  portId: string
}
export interface BlobRef {
  digest: string
  mediaType: string
  size: number
}
export interface EdgeCommand {
  edge: Edge
  graphId: string
}
export interface Edge {
  channel: 'data' | 'exec' | 'error'
  from: Endpoint
  to: Endpoint
}
export interface Endpoint {
  nodeId: string
  portId: string
}
