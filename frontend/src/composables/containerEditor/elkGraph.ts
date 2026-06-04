// elkGraph.ts —— 自动布局纯函数集(可单测，无 vue-flow / 全局 registry 依赖)。
export interface Size { width: number; height: number }

const ROW_H = 26 // 每个 pin 行约高
export function estimateNodeSize(kind: string, cfg: Record<string, any> = {}): Size {
  switch (kind) {
    case 'CommentBox':
      return { width: Number(cfg.width) || 320, height: Number(cfg.height) || 160 }
    case 'Switch': {
      const n = Array.isArray(cfg.cases) ? cfg.cases.length : 1
      return { width: 240, height: 90 + n * ROW_H }
    }
    case 'Parallel':
    case 'Race': {
      const n = Number(cfg.n) || 2
      return { width: 240, height: 90 + n * ROW_H }
    }
    case 'Expr': {
      const n = Array.isArray(cfg.inputs) ? cfg.inputs.length : 1
      return { width: 240, height: 80 + n * ROW_H }
    }
    default:
      return { width: 220, height: 90 }
  }
}
