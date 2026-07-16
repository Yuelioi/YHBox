---
kind: note
summary: "Node Package manifest v1 的不可变身份、namespace、host API、WIT/Process payload 与当前未开放边界"
activation: action
read_when: "实现/修改 Node Package 安装、信任、升级、Catalog merge、Wasm/Process host、SDK/conformance 或插件 payload 解包前"
recheck_when: "改 internal/nodepackage manifest、Node Contract implementation ABI、package artifact lock、publisher namespace、host API generation 或 payload path/digest 规则时"
---
# Node Package manifest v1

当前 Node Package 的单一 manifest 核心在 `internal/nodepackage`。它只建立可验证、可 reopen 的内容寻址包身份；尚未发现、安装、信任或执行第三方代码。

## 已冻结的边界

- format/version 是 `yotta.node-package` / `1`；canonical JSON 经 `yotta/node-package-manifest/v1` domain hash 得到 manifest digest。
- publisher namespace 必须是 canonical HTTPS URI；package ID 以及包贡献的 Data Type、Capability、Node Type 必须是该 namespace 下以 `/vN` 结尾的版本化 URI。
- package release version 是严格 SemVer。host API 使用半开区间 `[min,maxExclusive)`，当前 generation 形态是 `major.minor`；未来破坏性 generation 不会被旧包自动接受。
- manifest 内嵌 exact Data Type / Capability / Node semantic artifacts；实现只允许 Node Contract 明确声明的 `wit/vN` 或 `process/vN` ABI。`builtin` 和 `host-instruction` 不能伪装成第三方包。
- executable/documentation bytes 不嵌入 JSON；`Payload` 锁定 portable relative path、digest、size、media type。路径拒绝 traversal、反斜杠/盘符、Windows reserved device name 和非 portable segment。
- implementation 同时锁定 entrypoint 和归一化 OS/architecture 集合。manifest getter 不泄漏可修改的内部 slice。

## 与现有 3.1 深模块的关系

- `nodecontract` 决定一个 Node Type 允许哪些 ABI；manifest 不能扩大它。
- `nodecatalog.ImplementationLock` 和 Program Snapshot 已能锁 package ID、artifact digest、ABI、entrypoint；后续 Catalog merge 必须从已验证/已启用的 manifest 生成 lock，不接受 UI 或 workflow 自报。
- `admission`、`capability` 与 `resource` 已有 plugin instance/session authority 维度；Wasm/Process host 必须消费这些既有边界，不建立本地可信旁路。

## Archive verification 已实现

- archive 根固定为 `yotta-node-package.json` 加 manifest 精确声明的 payload；`internal/nodepackage.ExtractArchive` 是验证和安全解包的唯一外部 seam。
- payload digest 是 exact bytes 的 raw SHA-256，与 BlobRef identity 一致，不使用 contract 的 domain-separated hash。ZIP header size 先与 manifest 比较，流式写入时再验证实际 bytes、SHA-256、CRC 和 cancellation。
- entry 必须是 portable regular file；重复、额外、缺失、目录、symlink/特殊文件、case-fold collision、traversal、反斜杠和 reserved path 都 fail closed。archive/expanded bytes 与 entry count 在解压前有上限。
- 解包只写 destination 同级的 private staging directory；Process payload 才获得 executable mode。成功后一次 rename；失败、取消或 destination 已存在时不会发布最终目录并清理 staging。

## Local lifecycle 已实现

- 在 publisher signature/key 尚未落地前，唯一信任类型是精确绑定 manifest digest 的 `local-artifact`；它只表示本机批准这一份 artifact，不能推导 publisher namespace ownership，也不能自动信任升级。
- `internal/nodepackage.Store` 的 immutable generation 位于 `generations/<manifest-digest>/`，canonical `registry.json` 是唯一 authority 与 commit record。generation 先 durable publish，registry-last 才使 install/update 生效；incoming/orphan bytes 不可运行且 reopen 时清理。
- Store reopen、generation 复用和 rollback 都调用 `OpenExtracted` 重验 manifest、精确 file set、size、raw SHA-256 和 host-owned mode。损坏 generation、dangling pointer、stale trust、未知 schema/entry、symlink 或超预算 registry 会使 Store fail closed，不静默跳过。
- local lifecycle 支持 snapshot-only list/get、install/update、enable/disable、quarantine、单步 rollback 与 uninstall。quarantine 强制禁用且不能由 Enable 绕过；rollback 只选已验证、未 quarantine generation 并保持 disabled；uninstall 先撤销 registry authority 再 best-effort 清理 inert bytes。

## 尚未实现

签名 envelope、publisher key identity/namespace ownership、revocation input、Catalog merge、Program package lock、Wasm Component/WIT host、Process protobuf host、sandbox/quota、SDK 和 conformance fixtures 都仍是后续切片。

在这些能力完成前，主程序必须保持“没有第三方包时无任意代码加载面”；不要把 manifest 存在误写成插件可运行。
