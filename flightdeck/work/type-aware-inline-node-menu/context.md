# Type-aware Inline Node Menu Plan context

## What matters

候选节点必须来自当前 Catalog 和精确 Data Type 兼容关系；自动连线不能引入隐式 coercion 或绕过
Compiler diagnostics。旧计划只用于理解早期交互目标。

## Decisions

- 候选过滤使用版本化类型事实，不使用前端颜色或宽泛 tag 推断。
- 类型转换必须在 Workflow Source 中显式可见。

## Terms

- **Inline candidate menu:** 从连线端口上下文打开的节点候选入口。
- **Conversion:** Source 中显式存在、由 Compiler 验证的数据类型转换。
