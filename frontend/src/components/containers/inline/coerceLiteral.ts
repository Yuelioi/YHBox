// 内联 literal 写回前按 pin 的规范类型强制收口。
//
// 为什么: PinInput / PinLiteral 按 widget kind 选控件, 某个 widget 没显式分支就掉到文本框兜底,
// 文本框一律 String(v) → number pin 被存成字符串 → 保存被 LITERAL_TYPE_MISMATCH 拦
// (Sleep.Duration 的 'duration' widget 漏配即此坑)。逐个补 widget 分支治标且易再漏;
// 这里在 emit 出口按 pin 的规范类型统一 coerce, 从结构上消灭"标量存错类型"整类。
//
// 标量 (number/bool/string) 强制收口; 结构化 (point/list/any/几何等) 原样透传不碰。
import type { PinType } from '../pinSpec'

// safeCoerceForFix: 校验错误「一键修复」用。把 literal 值安全 coerce 到期望的 pin 类型,
// **只在能无损/无歧义转换时**返回修复值; 含糊不清 → 返回 undefined (不提供修复, 留用户手改)。
//
// 同时充当面板的「能不能修」判定 (返回非 undefined = 可修)。expected 是后端发的规范类型字符串
// (number/bool/string/point/...，见 canonPinType)。
//
// 关键分寸: number 只治**干净数字串** ("13000" → 13000); "500ms" / "abc" 一律不碰, 绝不静默改 0
// 掩盖手误 —— 那种留给用户自己改。
const CLEAN_NUMERIC_RE = /^-?\d*\.?\d+$/
export function safeCoerceForFix(v: unknown, expected: string): unknown {
  switch (expected) {
    case 'number':
      if (typeof v === 'string' && CLEAN_NUMERIC_RE.test(v.trim())) {
        const n = Number(v)
        if (Number.isFinite(n)) return n
      }
      return undefined
    case 'bool': {
      if (typeof v === 'number') return v !== 0
      if (typeof v === 'string') {
        const s = v.trim().toLowerCase()
        if (s === 'true' || s === '1') return true
        if (s === 'false' || s === '0') return false
      }
      return undefined
    }
    case 'string':
      // 标量 → 字符串 (无损); null / 对象 不碰 (避免 "[object Object]" / 误抹空)。
      if (v != null && typeof v !== 'object') return String(v)
      return undefined
    default:
      return undefined // point / geometry / any / list 等不机械修
  }
}

export function coerceLiteral(v: unknown, type: PinType): unknown {
  switch (type) {
    case 'number': {
      const n = Number(v)
      return Number.isFinite(n) ? n : 0
    }
    case 'bool':
      return !!v
    case 'string':
      if (v == null) return ''
      // string pin 不该携带对象; 真撞到 (防呆) 就原样, 别 String(obj) 糊成 "[object Object]"。
      if (typeof v === 'object') return v
      return String(v)
    default:
      return v // point / list / any / 结构化 → 原样
  }
}
