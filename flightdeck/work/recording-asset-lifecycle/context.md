# Recording Asset Lifecycle context

## What matters

录制过程不是 durable asset。只有用户完成命名并 Finalize 后才创建可引用资源；取消、丢弃、失败和
应用退出都不能留下幽灵资产。清理必须以当前引用事实为准。

## Decisions

- Finalize 是唯一创建 durable recording asset 的边界。
- 未引用判断只用于候选展示，执行删除前必须再次由 owner 复检。
- 普通用户看到清晰状态，原始事件和校准只向高级诊断逐步披露。

## Terms

- **Pending recording:** 尚未 Finalize、不能被其他对象引用的录制会话结果。
- **Finalize:** 验证、命名并写入正式 asset store 的原子操作。
