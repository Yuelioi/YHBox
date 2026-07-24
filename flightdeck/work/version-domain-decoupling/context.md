# 版本域解耦与自动化维护 context

## What matters

版本是兼容性边界的名称，不是产品代号的复制品。产品发版、持久化 artifact、宿主接口、进程协议、
存储布局和单个节点语义必须只在各自发生兼容性变化时升级。

## Decisions

- 不建立一个全局 Contract Version；每个拥有独立兼容性边界的 module 自己拥有 `format` 与当前版本。
- 产品发布版本保留 SemVer，并通过一个权威源生成构建工具要求的静态副本。
- 当前尚未发布稳定版，开发期 `3.1` artifact 可一次性破坏性切换到独立 `v1`，不增加旧开发数据
  migration、fallback reader 或第二套 runtime。
- 正式 `v1` 之后，持久化格式只通过显式 `format + version` migration chain 演进。
- Node Type 保留独立 SemVer 与 semantic digest；Compiler 实现保留 build digest，不借用产品版本。
- 版本 inventory 只汇总和检查各域，不要求它们相等。
- 自动化脚本必须覆盖版本展示、生成、提升与校验，并由 `task check` 执行适用门禁。

## Terms

- **Product Release Version:** 标识一次 Yotta 应用发布的 SemVer，不表达数据或协议兼容性。
- **Artifact Contract Version:** 某一种持久化 artifact 的独立格式代际，与其 `format` 共同决定 reader 和 migration。
- **Host Interface Version:** 宿主与可执行扩展之间用于兼容判断的独立接口版本。
- **Wire Protocol Version:** 两个进程或传输端点握手和解释消息时使用的协议代际。
- **Storage Layout Version:** 标识某个本地 store 的目录、marker 和提交语义的独立代际。
- **Revision:** 同一实体内容变化的顺序计数，不称为版本。
