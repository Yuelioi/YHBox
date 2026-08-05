# Node and data-type development

## One contract chain

```text
Data Type + Capability/Configured Target
                 ↓
          versioned Node Contract
                 ↓
sealed builtins/Catalog ──→ Authoring Projection
                 ↓                 ↓
        implementation lock   frontend editors
                 ↓
          runtime adapter → Compiler/Executor/Run journal
```

权威路径：`internal/datatype`、`internal/nodecontract`、`internal/nodes`、`internal/nodecatalog`、
`internal/nodeauthoring`、`internal/noderuntime`、`internal/workflow/compiler`。不存在包级 mutable registry、
`init()` 隐式注册、按 title/kind 猜实现或第二套 legacy runtime。

## Change sequence

1. 先定义语义：复用 exact `TypeRef`；新类型同时定义 schema、representation/carrier、authoring 和关系覆盖。
   effect 必须选择正确 authority：AI/File/Blob/Stream/guest 用 capability requirement；Network/Application/
   Automation 用 Configured Target。不能把 Target 再包成 consent/grant，也不能把 credential 写进 config。
2. 在 `internal/nodes` seal Contract：稳定、不含产品版本的 `nodeTypeId`，独立 SemVer，strict JSON Schema，
   稳定 lowercase/kebab port，完整 execution/instruction/error/status/state/target/capability/ABI。
3. 用 `BuiltinDefinition` 绑定 exact implementation entrypoint/version/conformance；在 `internal/noderuntime`
   安装匹配 implementation lock 的 adapter。adapter 只处理 typed input/output、窄 session 与真实
   AdapterAction，不拥有调度。
4. `internal/nodeauthoring` 从同一 Catalog 派生端口、类型、editor adapter、target/credential 和 availability。
   前端消费 Projection；特殊值使用类型级 editor，不按 node ID 维护第二套表。
5. 补 zh/en 节点、端口、配置和错误文案；默认/config builder 每次返回独立值，不能跨节点或 revision
   共享可变对象。

## Type and execution rules

- 关系区分 exact、assignable、generic binding、lossless convert、lossy/fallible 和 assertion；Compiler 是最终
  权威，前端不把它们压成自维护 boolean。
- 会改变表示或可能失败的转换是图中可见节点。只有唯一、确定、无副作用、无损、无 capability 的单步转换
  才可自动插入。
- data、exec、error 是不同 channel；纯数据节点没有伪 exec pin，status event 不作为画布 handle。
- BlobRef 可 durable；Stream/Resource/HeldInput handle 必须带 Run lease binding，不能写入 Source 或 durable
  Run value。
- State slot 在 Source 声明 exact type，Compiler 冻结到 Program，每个 Run 独立；连接不能反向改变声明。
- Script 是一次性隔离 typed effect，不恢复 DOM、文件、网络、process、Catalog 或 ambient variable API。

## Verification and generated views

- contract seal/open、semantic digest、implementation lock、strict config 和 typed value round-trip。
- type × capability/target closure、Projection/Compiler parity、adapter conformance 和 declared error/action。
- effect 至少覆盖 Source → compile → admission/target → adapter → Run journal 的纵向测试。
- Windows/ADB/Browser/recording/asset 节点按[构建指南](../build/build.md)补真实 target smoke。
- `task nodes` 生成当前 Markdown；`task nodes:catalog` / `task nodes:authoring` 生成 machine view；
  `task contracts:check` 拒绝 tracked contract 漂移。不要手抄节点总数、端口全集或类型颜色表。
