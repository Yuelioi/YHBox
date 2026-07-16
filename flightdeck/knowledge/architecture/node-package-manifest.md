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

## 尚未实现

签名与 namespace ownership、trust store、atomic install/update/disable/uninstall/rollback/quarantine、revocation、Catalog merge、Program package lock、Wasm Component/WIT host、Process protobuf host、sandbox/quota、SDK 和 conformance fixtures 都仍是后续切片。

在这些能力完成前，主程序必须保持“没有第三方包时无任意代码加载面”；不要把 manifest 存在误写成插件可运行。
