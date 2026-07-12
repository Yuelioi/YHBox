// ELK 实例按需加载：自动布局不是 editor 首屏能力，不能把约 470 KB gzip 的
// bundled engine 静态塞进 ContainerEditor chunk。
//
// worker 升级路（spike 验证 Vite/wails 打包跑通后启用）：
//   import ELK from 'elkjs/lib/elk-api'
//   export function createElk() {
//     return new ELK({
//       workerFactory: () =>
//         new Worker(new URL('elkjs/lib/elk-worker.min.js', import.meta.url), { type: 'module' }),
//     })
//   }
import type { ElkNode, ELK as ElkInstance } from 'elkjs/lib/elk-api'

let elkPromise: Promise<ElkInstance> | undefined

export function loadElk(): Promise<ElkInstance> {
  elkPromise ??= import('elkjs/lib/elk.bundled.js').then(({ default: ELK }) => new ELK())
  return elkPromise
}

/** 基础布局参数。direction 由 useElkLayout 按 LR/TB 传入。 */
export function baseLayoutOptions(direction: 'RIGHT' | 'DOWN'): Record<string, string> {
  return {
    'elk.algorithm': 'layered', // 显式固定，不靠默认（默认未来可能变）
    'elk.direction': direction,
    'elk.edgeRouting': 'ORTHOGONAL',
    'elk.layered.spacing.nodeNodeBetweenLayers': '80',
    'elk.spacing.nodeNode': '40',
  }
}

// exec 主轴的 edge priority 字段。node smoke 已验证此 key 被 elk.layout 接受不报错；
// 视觉效果（exec 主链是否够直）留到 T8 smoke 在真实容器上肉眼微调。
export const PRIORITY_KEY = 'elk.layered.priority.straightness'

export type { ElkNode }
