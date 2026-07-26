---
title: 定义 Data Type 3.1 与 Value Envelope
label: wayfinder:research
parent: ../map.md
status: closed
assignee:
blocked_by: []
---

# 定义 Data Type 3.1 与 Value Envelope

## Question

Data Type 的稳定身份、schema dialect、版本、兼容关系、显式 Conversion、null/optional/union/list/record、瞬时 host handle 与大对象传输应如何表达，才能让 Go、TypeScript、Wasm、Process Node、Compiler、runtime、UI 和文档使用同一语义，同时彻底删除现有隐式 coercion 与 `any` 降级？

## Resolution

已接受。规范依据与跨语言约束见研究资产：[Data Type 3.1 与 Value Envelope](../../../research/data-type-value-envelope-3.1.md)。

### 1. 建立四层模型

1. **TypeRef**：引用一个精确的名义 Data Type。
2. **TypeDefinition**：Data Type 的不可变语义定义及独立的 authoring annotations。
3. **ValueEnvelope**：节点、运行时、Wasm 与 Process Node 之间交换的 typed discriminated union。
4. **BindingState**：独立表达输入是否提供；它不属于 ValueEnvelope。

`absent`、`present(null)`、`present(default value)` 是三个不同状态。JSON Schema 的 `default` 只生成提示；Compiler 在明确的 default application phase 产生值并记录来源，validator 不得填默认值。

### 2. TypeRef 使用绝对、版本化、带摘要的名义身份

所有持久化、编译和跨执行载体的 TypeRef 都必须同时 pin：

```json
{
  "typeId": "https://schemas.yotta.dev/types/core/string/v1",
  "semanticDigest": "sha256:..."
}
```

- `typeId` 是发布者控制的绝对 URI，版本属于身份；内置类型保留 Yotta namespace，Node Package 只能发布其获准 namespace 下的类型。
- `semanticDigest` 固定本次 Catalog Snapshot 实际绑定的 validation schema、representation 与 codec 语义；仅在一个已冻结 registry 内可以用 `typeId` 作为查表优化，不能据此省略持久化 pin。
- 相同 JSON Schema 但不同 TypeRef 的类型不兼容。例如 normalized point、screen pixel point 和 AE composition point 必须是不同类型。
- 未知 TypeRef、同 TypeRef 不同 semantic digest、namespace 冒充或远程替换一律是 Compiler error，不得降级成 `any`。

### 3. TypeDefinition 固定 JSON Schema profile，但分离语义与展示

```json
{
  "format": "yotta.data-type",
  "version": "3.1",
  "typeRef": {
    "typeId": "https://schemas.yotta.dev/types/core/point/v1",
    "semanticDigest": "sha256:..."
  },
  "semantic": {
    "schemaDialect": "https://json-schema.org/draft/2020-12/schema",
    "schema": {},
    "representations": ["inline-json"]
  },
  "authoring": {
    "titleKey": "type.core.point.title",
    "descriptionKey": "type.core.point.description",
    "color": "#a855f7",
    "icon": "point",
    "examples": [],
    "editorAdapter": "point"
  }
}
```

- 只接受 JSON Schema Draft 2020-12 和 Yotta 明确登记的 vocabulary/format assertion profile。
- `$id`、`$schema` 和所有 `$ref` 必须随 Node Package 离线 bundle；Compiler 禁止联网解析 `$ref`，并实施 schema 字节、深度、引用和求值预算。
- `semanticDigest` 的 preimage 固定为 domain `yotta/data-type-semantic/v1` 加下列对象的 JCS bytes：`typeId`、schema dialect、按绝对 `$id` 排序的完整离线 schema bundle、representation 与 codec 语义。preimage 明确排除 `semanticDigest` 字段本身和全部 authoring annotations；先计算摘要，再把结果写入 TypeRef，加载时必须重算并比对，因此不存在自引用摘要。
- 标题、翻译、颜色、icon、examples 等 authoring annotations 不改变类型兼容性或 Program identity；它们进入独立的 presentation catalog generation。
- 改变既有值是否合法、值的单位/含义、canonical codec、representation 或 Conversion 结果时必须发布新版本 TypeRef。仅修改 authoring annotations 可保留 TypeRef，但会产生新的 presentation catalog generation。
- 第三方类型必须能由 schema 通用渲染；`editorAdapter` 只能引用 Yotta 内置 allowlist，不能加载插件 JavaScript。

### 4. Port 使用 Type Expression，运行值必须落成精确类型

Node Contract 的 Data port 不再保存自由字符串，而使用规范化 Type Expression：

- `ref(TypeRef)`：一个精确类型；
- `list(TypeExpr)`：同质有序列表；
- `union(TypeExpr...)`：显式、排序、去重的联合；
- `var(name, constraints)`：泛型节点的编译期类型变量。

TypeDefinition 只定义名义 `ref` 叶子；`list` 是 envelope version 固定的内置类型构造器，第三方包不能重定义。运行值使用规范化的 Resolved Type：`ref(TypeRef)` 或递归的 `list(ResolvedType)`。其身份是带构造器 tag 的完整规范树；不为 `list<T>` 发明第二个会漂移的 schema digest。union 只表达端口可接受集合，运行值必须携带实际选中的 Resolved Type；type variable 也必须在 seal Program 前解析，runtime ValueEnvelope 不得携带 union 或未解变量。

optional 是 BindingState 的规则，不是把类型变成 nullable；nullable 必须让 union 显式包含 `core/null`。业务 record、enum 和 tagged variant 由具名 TypeDefinition 的 schema 定义。结构 list 的 validation、codec 与 representation 规则属于内置构造器；非 inline list 的 materialization 由 Blob Store、Stream 与 Resource Broker 票据继续定义。

旧 `*` 和 wildcard `Any` 删除。`core/json/v1` 只表示 JSON value tree，不兼容 Window、Image、File、resource 或任意插件值。`Eq`、`Log`、变量、list 等通用节点通过 type variable 与 trait constraint 表达，而不是通过 `any` 绕过检查。

### 5. ValueEnvelope 是四分支 sealed union

```text
ValueEnvelope
├─ inline-json  小型、可持久化、按 schema 校验的标量或结构值
├─ blob-ref     不可变、内容寻址、可持久化的大型二进制
├─ stream-ref   有序、增量、带背压和终态的运行期流
└─ handle-ref   由宿主 broker 解析的临时 resource capability
```

共同字段包括 envelope version 和完整 Resolved Type；`repr` 是强制判别字段，一次只能出现一个分支。ValueEnvelope 只表达语义值与承载方式；value ID、produced-by、derived-from、attempt 等运行元数据由后续 Run Value 包装，不进入值摘要。

#### inline-json

- 值必须通过 TypeDefinition schema，并按 Yotta JCS profile canonicalize。
- 拒绝重复键、非 UTF-8、NaN/Infinity、负零和超预算值；JCS 不做 Unicode normalization，NFC 等变化只能由显式 Conversion 完成。
- JSON number 仅允许有限 binary64 和 interoperable safe integer。`int64`、`uint64`、BigInt、decimal 使用各自 TypeRef 的规范十进制字符串 codec，绝不转成 TypeScript `number`。
- 语义 digest 对 `envelopeVersion + ResolvedType + repr + canonical value` 做版本化 domain-separated SHA-256；不得散列 Protobuf serialization。
- inline 的绝对硬上限设为 1 MiB；host/transport 可以配置更小阈值并把同一 Data Type materialize 为允许的 blob representation。具体 content-store policy 由后续票据决定。

#### blob-ref

- 使用 `mediaType + sha256 digest + raw byte size` 描述不可变内容；locator 只用于查找，不能成为身份，读取后必须重验 size 与 digest。
- Image、模型、长音频和大型文件不再在 `node.Image.Data`、JSON base64、Protobuf `bytes` 或 Wails payload 中复制。
- blob 可持久化、缓存和重放；它与 stream/handle 之间只能通过显式 materialize/open/export 节点转换。

#### stream-ref

- 是运行期、带类型的有序增量通道，必须定义 chunk/item 上限、背压、取消/deadline、half-close、terminal error 和 producer/consumer cleanup。
- stream 不等于 list，也不能持久化。`Stream -> Blob` 必须是受配额控制的显式 drain/materialize effect。

#### handle-ref

- token 至少绑定 authority、plugin/process instance、session、run/invocation scope、resource TypeRef、owner 和允许的 operation set；原始 HWND、Go pointer、JS object、Wasm address、文件描述符和 Blob URL 不得进入 envelope。
- 3.1 不允许插件之间直接委托 handle。任何转交必须回到 host broker 重新授权并产生新 token。
- token 高熵、不可猜测、可撤销；在 run 结束、plugin crash、timeout 或 explicit drop 后失效。handle/stream 禁止进入 Workflow Source、Program Snapshot、durable variable、cache key 或 replay record。
- 需要持久化或重放时，必须显式 snapshot/export 成 inline/blob，或保存可以重新 resolve 的领域 selector/recipe。

### 6. Compatibility 与 Conversion 只有一个权威算法

- assignability 只在 Type Expression grammar 内计算：名义 `ref` 必须完整 TypeRef 相等；immutable `list` 递归比较 element expression；输出 union 的每个分支都必须能赋给输入，输入 union 至少有一个分支接受当前输出。除这些集合规则外不使用 JSON Schema 结构包含关系，也不复用 TypeScript structural compatibility。
- schema compatible 不代表业务兼容；不同版本 TypeRef 也不自动兼容。
- Conversion 是 registry 中具名、版本化、画布可见的 Node Type，声明 from/to、lossless/lossy、determinism、effects/capabilities 和 representation requirements。
- UI 可以推荐 Conversion；Compiler 不得生成隐藏 cast。即使自动插入，也必须写回 Workflow Source 成真实节点和边。
- Go、TS、Wasm 和 Process adapters 不实现自己的 compatibility switch；它们消费同一 Type Catalog projection 和 conformance corpus。

### 7. 传输格式不是语义真相源

- JSON Schema/TypeDefinition 是类型和文档真相源；JCS bytes 是 inline 值 identity 的基础。
- Process Node 使用 length-delimited binary Protobuf frames 承载 sealed envelope、BindingState、request ID、deadline 和 protocol version。ProtoJSON/换行 JSON 只能用于受限调试 projection。
- Protobuf adapter 必须保留 optional presence 和 unknown fields；`google.protobuf.Value` 只可映射 `core/json`，`Any` 不得成为无约束通用值袋。
- Wasm Node 从相同 TypeDefinition 生成/映射 WIT record、variant、option/result、list、stream 和 resource。WIT 与 Protobuf 都必须通过同一 golden conformance vectors；它们不定义新的类型兼容规则。

### 8. 初始迁移方向

| 旧 tag | 3.1 方向 |
| --- | --- |
| `String` | `core/string/v1` inline |
| `Number` | `core/float64/v1` inline finite binary64 |
| `Integer` | `core/safe-integer/v1` inline；需要 64 位时用 decimal-string `core/int64/v1` |
| `Bool` | `core/bool/v1` inline |
| `Duration` | `core/duration/v1`，规范 int64 纳秒字符串，UI 使用人类可读 formatter |
| `JSON` | `core/json/v1` inline JSON tree，不是 wildcard |
| `*` / `Any` | 删除；改为 type variable、明确 union 或真实 TypeRef |
| `List` | 端口使用 `list(TypeExpr)`，运行值使用 `list(ResolvedType)`，删除 `[]any` |
| `Point` / `Rect` / `Geometry` / `Color` | 分别使用具名 inline TypeDefinition 和精确 schema |
| `Image` | image Data Type + `blob-ref`，不再内联 `[]byte` |
| `Window` | host window Data Type + `handle-ref`，不再传 `HWND` |
| `File` | 拆成可持久化 path/selector、content blob 与 capability-bound file handle，不再混为一个 struct |

### 9. 强制验收不变量

- 同一测试向量覆盖 absent、present-null、present-zero/default、union、unknown TypeRef、digest mismatch。
- 覆盖 `list<string>`、嵌套 list、list union assignability、不同 element digest，以及 ValueEnvelope 中不得出现 union/type variable 的拒绝测试。
- Go/TS 对全部内置 inline 类型执行 parse/validate/JCS/digest golden tests；Wasm/Process adapter 使用同一 vectors。
- `int64` 边界、Unicode 非规范等价字符串、重复键、对象键顺序、`-0`、NaN/Infinity 都有明确测试。
- 未安装 TypeRef、远程 `$ref`、恶意递归 schema、namespace collision、同 ID 不同 digest 在 registry seal 前失败。
- blob 覆盖 size/digest mismatch；stream 覆盖背压、取消和 producer failure；handle 覆盖越权、过期、跨 session 与跨插件使用。
- Authoring Projection 能从 TypeDefinition 生成类型名、精确 TypeRef、约束、单位、格式、nullable/optional、representation、生命周期、安全提示、examples 和可用 Conversion。
- 完成迁移后删除 `CanonicalPinType`、`PinTypeCompat`、`CoerceInputValue/coerceToType`、前端 `PinType` union 与 `backendTypeToPinType`；仓库不得保留第二套兼容算法。
