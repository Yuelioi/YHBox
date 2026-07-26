# Data Type 3.1 与 Value Envelope：跨 Go / TypeScript / Wasm / Process 插件的数据契约研究

> 状态：非规范研究资产。示例是决策输入，不是 wire contract；最终规范以 [定义 Data Type 3.1 与 Value Envelope](../wayfinder/node-system-3.1/tickets/define-data-types-and-value-envelope.md) 的 Resolution 为准。

> 调研日期：2026-07-15
> 范围：节点端口类型、运行时值、持久化、跨进程/Wasm 边界、大对象与流、插件自定义类型、可生成文档的 schema 来源。
> 来源约束：只采用正式标准、规范仓库和官方语言/项目文档。

## 结论

Yotta 不应把“端口类型”“一次运行中的值”“二进制载荷”“流”和“宿主资源句柄”继续压成同一种 JSON 对象。建议 3.1 明确分成四层：

1. **TypeRef**：稳定、名义化、带版本的语义类型身份；
2. **TypeDefinition**：以 JSON Schema 2020-12 为可校验、可生成文档的描述，并以摘要固定具体版本；
3. **ValueEnvelope**：值在节点边界上的带判别联合，只表达一种明确 representation；
4. **BindingState**：独立表达“未连接/未提供”，不能借 `null` 或默认值代替。

推荐的最小表示集合是 `inline-json`、`blob-ref`、`stream-ref`、`handle-ref`。前三者分别覆盖小型可持久化值、内容寻址的大对象、带背压的增量序列；`handle-ref` 是当前宿主会话中的临时 capability，默认禁止持久化、缓存、跨进程复制和重放。

这不是选择 JSON、Protobuf 或 WIT 三者之一。JSON Schema 是**类型定义和文档真相源**；进程插件可用 JSON/Protobuf 作为传输适配；Wasm 插件应投影为 WIT 原生类型与 resource；所有适配都必须保持同一 TypeRef 和 3.1 语义。

## 1. 必须先固定的语义

### 1.1 类型身份是名义的，不是仅按结构相等

JSON Schema 的 `$id` 为 schema resource 建立规范 URI；它是标识符，不保证是可联网获取的位置。根 schema 推荐使用不带 fragment 的绝对 URI。`$schema` 则声明 dialect/meta-schema，应位于根部；缺失时采用什么 dialect 属于实现定义。`$ref` 是应用器，URI reference 按当前 base URI 解析；Draft 2020-12 允许 `$ref` 旁边存在其他关键字。[JSON Schema Core 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)

因此 TypeRef 建议至少包含：

```json
{
  "typeId": "https://schemas.yotta.dev/types/media/image/v1",
  "schemaRef": "https://schemas.yotta.dev/schemas/media/image/v1",
  "schemaDigest": "sha256:..."
}
```

- `typeId` 是语义身份，必须由发布者控制的命名空间产生并显式版本化；形状相同但单位、坐标系或业务语义不同的类型不得自动兼容。
- `schemaRef` 是 schema 的规范 `$id`；执行时默认从已安装 registry/包读取，不能把任意网络拉取放进验证路径。
- `schemaDigest` 固定本次安装实际使用的 schema 内容，防止同一 URI 被静默替换。
- 插件类型使用其发布者命名空间，例如 `https://example.org/yotta/types/acme/color/v1`；禁止用 `Color` 这种全局短名注册。

WIT package 也采用 `namespace:name@version` 身份，Protobuf `Any` 官方建议在破坏兼容时使用版本化类型名；两者都支持“命名空间 + 版本是契约的一部分”的方向。[WIT packages](https://component-model.bytecodealliance.org/design/wit.html#packages)、[Protobuf `Any`](https://protobuf.dev/reference/protobuf/google.protobuf/#any)

### 1.2 `null`、未提供、默认值是三个状态

JSON Schema 中 `null` 是一个值，不等于属性不存在；对象属性默认不是必需，只有 `required` 明确要求存在。[JSON Schema null](https://json-schema.org/understanding-json-schema/reference/null)、[JSON Schema required](https://json-schema.org/understanding-json-schema/reference/object#required)

`default` 是 annotation，验证器不会据此填充值；它只给 UI、代码生成或其他非验证工具提供提示，并且默认值本身最好也通过对应 schema 验证。[JSON Schema annotations](https://json-schema.org/understanding-json-schema/reference/annotations)、[JSON Schema Validation 2020-12](https://json-schema.org/draft/2020-12/json-schema-validation)

所以节点输入必须另有 BindingState：

```ts
type BindingState =
  | { state: "absent" }
  | { state: "present"; value: ValueEnvelopeV31 }
```

当且仅当 schema 允许 `null` 时，`present` 的 `inline-json.value` 才可以是 `null`。默认值由绑定/配置层在明确阶段应用，并记录来源；校验层不得悄悄把 absent 改成 present。

### 1.3 联合必须有稳定判别字段

JSON Schema 的 `anyOf` 表示满足一个或多个分支，`oneOf` 表示恰好一个分支；当分支可能重叠时，单靠结构匹配会产生歧义，`oneOf` 还可能有额外验证成本。[JSON Schema combining schemas](https://json-schema.org/understanding-json-schema/reference/combining)

ValueEnvelope 和业务联合都应采用显式 `repr`/`kind` 判别字段，再用 `oneOf` 校验各分支。这个模型能直接映射到 WIT `variant` 和 TypeScript discriminated union；TypeScript 结构类型本身会允许形状相似值兼容，不能替代运行时名义 TypeId。[WIT types](https://component-model.bytecodealliance.org/design/wit.html#built-in-types)、[TypeScript narrowing](https://www.typescriptlang.org/docs/handbook/2/narrowing.html#discriminated-unions)、[TypeScript type compatibility](https://www.typescriptlang.org/docs/handbook/type-compatibility)

## 2. 推荐的数据模型

### 2.1 TypeDefinition 3.1

```ts
interface TypeDefinitionV4 {
  typeId: string                 // absolute, versioned URI
  schemaRef: string              // schema root $id
  schemaDialect: "https://json-schema.org/draft/2020-12/schema"
  schemaDigest: `sha256:${string}`
  schema: object                 // bundled, immutable for this digest
  representations: Array<"inline-json" | "blob-ref" | "stream-ref" | "handle-ref">
  annotations?: {
    title?: string
    description?: string
    deprecated?: boolean
    examples?: unknown[]
  }
}
```

schema 是验证、表单生成、端口提示和参考文档生成的共同来源；`title`、`description`、`examples`、`deprecated`、`readOnly`、`writeOnly` 属于标准 annotation。`format` 在 2020-12 中默认是 annotation，而非必然 assertion，因此 Yotta 必须固定采用哪些 format，并把额外断言写进自己的 vocabulary/profile，不能依赖不同验证器的默认行为。[JSON Schema Validation 2020-12](https://json-schema.org/draft/2020-12/json-schema-validation)

自定义 schema keyword 应通过带 URI 的 vocabulary 注册；未知关键字在核心模型中通常作为 annotation 传播，不应假设每个验证器都会执行它。[JSON Schema Core 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)

### 2.2 ValueEnvelope 3.1

```ts
type ValueEnvelopeV31 =
  | {
      envelopeVersion: "3.1"
      typeId: string
      repr: "inline-json"
      value: unknown
      valueDigest?: `sha256:${string}`
      provenance?: Provenance
    }
  | {
      envelopeVersion: "3.1"
      typeId: string
      repr: "blob-ref"
      blob: {
        mediaType: string
        digest: `sha256:${string}`
        size: number
        locator?: string
      }
      provenance?: Provenance
    }
  | {
      envelopeVersion: "3.1"
      typeId: string
      repr: "stream-ref"
      stream: {
        streamId: string
        itemTypeId: string
        ordering: "ordered"
        expectedItems?: number
        expectedBytes?: number
      }
      provenance?: Provenance
    }
  | {
      envelopeVersion: "3.1"
      typeId: string
      repr: "handle-ref"
      handle: {
        handleId: string
        resourceTypeId: string
        scope: "invocation" | "workflow-run" | "plugin-session"
        expiresAt?: string
      }
      provenance?: Provenance
    }
```

约束：

- `typeId` 决定语义；`repr` 只决定当前承载方式，不能借更换 repr 改变类型。
- `provenance`、时间戳、locator 等可变元数据不进入语义值摘要。
- `blob-ref.digest` 是原始字节内容摘要；`inline-json.valueDigest` 是带 TypeId 的规范值摘要；二者不能混用。
- `stream-ref` 与 `handle-ref` 默认是 ephemeral。持久化工作流只存可重新建立它们的配置，不存活的 stream/handle。
- locator 只是一种取回提示，不是身份；读取后必须校验 `size` 和 `digest`。

OCI descriptor 已验证了 `mediaType + digest + size` 的内容描述模式，并允许可选 URL/annotations/embedded data；OCI distribution 还定义了按 digest 获取 blob、HEAD/Range 和分块上传。Yotta 可借用这个形状而不必实现 OCI 协议。[OCI Image Spec descriptor](https://specs.opencontainers.org/image-spec/descriptor/?v=v1.1.0)、[OCI Distribution Spec](https://github.com/opencontainers/distribution-spec/blob/main/spec.md)

## 3. 规范化与摘要

RFC 8785 JSON Canonicalization Scheme（JCS）用 ECMAScript 原始值序列化、I-JSON 子集和确定性的对象属性排序生成 UTF-8 canonical bytes，适合签名和摘要；它是 Informational RFC，不自动替 Yotta 定义领域值等价关系。[RFC 8785](https://www.rfc-editor.org/rfc/rfc8785)

推荐规则：

1. `inline-json` 的摘要输入固定为 `JCS({envelopeVersion:"3.1",typeId,repr:"inline-json",value})`，然后 SHA-256；排除 provenance、缓存位置和时间戳。
2. schema digest 对完整 bundled schema 使用同一 JCS profile；`$id` 和 `$schema` 必须包含在摘要输入中。
3. 解析前拒绝重复对象键；拒绝非有限数、无法稳定表达的 `-0`，并固定 UTF-8。
4. JCS 不做 Unicode normalization；需要 NFC 等语义的类型必须在显式转换/构造器中规范化，不能在 hash 层暗改字符串。
5. Protobuf 的 deterministic serialization 不是 canonical serialization，官方明确警告序列化字节不适合作为跨版本、跨实现的稳定语义 hash；未知字段还会阻碍 canonicalization。因此不得对 protobuf wire bytes 直接生成持久值身份。[Proto serialization is not canonical](https://protobuf.dev/programming-guides/serialization-not-canonical/)

JCS/I-JSON 的数值边界与 JavaScript `number` 对齐，不能无损承担任意精度整数或 decimal。3.1 应把 `int64`、`uint64`、BigInt、decimal 定义为具有明确正则和范围的十进制字符串类型；只有有限 IEEE-754 数可直接作为 JSON number。ProtoJSON 同样把 64 位整数编码成十进制字符串，`bytes` 编码成 base64，这说明跨 TS 边界不能假设普通 JSON number 能保持 64 位整数精度。[ProtoJSON format](https://protobuf.dev/programming-guides/json/)

Go 的 `encoding/json.Decoder.UseNumber` 可避免先把 interface 中的数解成 `float64`，`json.RawMessage` 可延迟变体解析。后端应先读 `envelopeVersion/typeId/repr`，查 registry 并验证，再解到精确 Go 类型；不要先解为 `map[string]any` 后依赖推断出的数值类型。[Go `encoding/json`](https://pkg.go.dev/encoding/json)

## 4. Protobuf / gRPC 适配边界

Protobuf 适合进程插件的高效传输，但不能成为 Data Type 3.1 的唯一语义模型：

- proto3 未声明 `optional` 的基础标量不追踪 presence，默认值与未设置混在一起；repeated/map 也不追踪 presence。新协议字段应使用显式 presence，且 BindingState 仍需独立建模。[Protobuf field presence](https://protobuf.dev/programming-guides/field_presence/)
- binary protobuf 在解析再序列化时保留 unknown fields，但转成 JSON 或逐字段复制会丢失它们；ProtoJSON 对 unknown fields 的兼容能力也弱于 binary。不能宣称 JSON/Protobuf 往返是无损的。[Proto3 unknown fields](https://protobuf.dev/programming-guides/proto3/#unknowns)、[ProtoJSON](https://protobuf.dev/programming-guides/json/)
- `Any` 只是 `type_url + serialized bytes`，解包仍依赖可信 descriptor/type registry；它不是任意安全值容器。接收端必须校验允许的 type URL、大小、对应 schema/descriptor 和插件权限。[Protobuf `Any`](https://protobuf.dev/reference/protobuf/google.protobuf/#any)
- `google.protobuf.Value` 只覆盖 null、double、string、bool、struct、list；number 是 double，不能无损表示 int64/decimal，也不能表示 blob、stream 或 resource handle。它不能直接替代 ValueEnvelope。[Protobuf `Value`](https://protobuf.dev/reference/protobuf/google.protobuf/#value)
- `oneof` 遇到未来新增的未知成员时，旧客户端看到 NOT_SET，无法区分真正未设置与未知分支；协议升级必须保留 envelope version/type/repr 并做兼容测试。[Proto3 oneof compatibility](https://protobuf.dev/programming-guides/proto3/#backwards-compatibility-issues)

流不能建模成一次性 `list<T>`。gRPC 支持 unary、server/client streaming 和 bidirectional streaming；同一 RPC 内消息有序，但双向两侧独立。flow control 用于防止接收端被压垮，写调用返回也不代表数据已上网；不谨慎的同步读写会死锁。[gRPC core concepts](https://grpc.io/docs/what-is-grpc/core-concepts/)、[gRPC flow control](https://grpc.io/docs/guides/flow-control/)

因此 `stream-ref` 的 transport adapter 必须定义：有界 chunk、顺序、背压、取消/deadline、半关闭/关闭、最终错误、最大 item/byte 限制。长生命周期 RPC 虽省去反复建连，但开始后难以重新负载均衡，也更难调试并可能影响扩缩容，不能把所有节点边都默认变成长流。[gRPC performance best practices](https://grpc.io/docs/guides/performance/)

binary protobuf wire format 本身不提供连续消息的边界，官方建议流中多消息采用 size + payload framing；同时官方把“大数据集/大消息”列为不适合直接塞进 protobuf 的场景，并提示消息通常需要整体进入内存、超过约 1 MB 就应考虑替代方案。[Protobuf streaming multiple messages](https://protobuf.dev/programming-guides/techniques/#streaming)、[Protobuf large data sets](https://protobuf.dev/programming-guides/techniques/#large-data-sets)

所以 process plugin 的 stdin/stdout 若使用 protobuf，必须使用 length-delimited frame，并包含 protocol version、message kind、request/run ID、长度上限和 deadline；大型 payload 只传 blob/stream 引用。换行分隔 JSON 只能作为受限调试 profile，不能承担任意字符串、二进制和大消息的正式边界。

## 5. Wasm Component / WIT 适配边界

WIT 的 `record`、`variant`、`option<T>`、`result<T,E>`、`list<T>` 可以直接承载大部分静态端口类型；`list<T>` 是一次性完整有序值，`stream<T>` 才是异步、增量的有序序列。WIT 定义调用契约而不定义业务行为。[WIT type definitions](https://component-model.bytecodealliance.org/design/wit.html)

WIT resource 用 handle 表示不能或不应复制的外部实体；`own<T>` 转移所有权，owned handle 被丢弃会销毁资源，`borrow<T>` 只在调用期间临时借用。WASI capability 模型把宿主资源暴露为不可伪造的 handle，实例通常只看到自己表中的索引而不是宿主指针。[WIT resources](https://component-model.bytecodealliance.org/design/wit.html#resources)、[WASI capabilities](https://github.com/WebAssembly/WASI/blob/main/docs/Capabilities.md)

Yotta 的 Wasm adapter 因而应：

- 优先把已知 TypeDefinition 生成/映射为 WIT record/variant/option/result；不要把所有值降级为 JSON string。
- 将 `blob-ref` 映射为 descriptor 或 blob resource，将 `stream-ref` 映射为 stream，将 `handle-ref` 映射为带 own/borrow 规则的 resource。
- 不暴露宿主内存地址、原始文件路径、任意文件描述符或数据库对象；handle lookup 必须检查实例、插件、运行、资源类型和生命周期。
- 跨 Component 调用时明确所有权；同一个 mutable resource 不得因 envelope 被复制而产生两个“owner”。

process 插件没有共享的 WIT handle table。它得到的 `handleId` 必须是高熵 broker token，由宿主在每次操作时校验 process/session/owner/type/capability，并在进程退出、运行结束或超时后失效；若需要重放或跨会话传递，先显式转换成 durable `blob-ref` 或领域 ID。

## 6. 大型二进制、图像和流

大型图像/视频/模型不应 base64 塞入核心 JSON envelope。base64 可保留为严格大小阈值以下的小值 transport convenience；超过阈值写入 content store，传 `blob-ref`，按需 Range/chunk 读取并在消费前校验 size/digest。这样也避免 Go/TS/process/Wasm 各边界重复分配整个 payload。

建议区分：

- **BlobRef**：不可变、按原始字节寻址、可缓存和持久化；适合图片、文件、模型权重。
- **StreamRef**：一次运行中的有序增量通道，带背压和终态；适合实时帧、长输出、未知总长度数据。
- **HandleRef**：对宿主实体或可变资源的临时权限；适合 AE 项目对象、GPU surface、打开的文件/设备。

三者不能隐式互转。`Stream -> Blob` 必须通过 drain/materialize 节点并受配额、取消和摘要校验约束；`Handle -> Blob` 必须调用该资源允许的 export/snapshot 操作；`Blob -> Handle` 是显式 open/import，会产生新生命周期和副作用。

WASI I/O 的正式接口也把 byte stream 定义为 resource：读取允许少于请求数量甚至暂时为空，写入前先取得可写额度，说明 partial I/O 和 backpressure 是协议语义而不是实现细节。[WASI I/O streams](https://github.com/WebAssembly/wasi-io/blob/main/wit/streams.wit)

前端的 Blob/Object URL 只应是 UI 会话中的投影。File API 将 `Blob` 定义为不可变原始字节，并规定 blob URL 有宿主作用域和可撤销生命周期；它不是 durable locator，不能写入 graph、缓存键或跨进程 envelope。[W3C File API](https://www.w3.org/TR/FileAPI/#url)

## 7. 显式转换与插件扩展

每个 conversion 应是 registry 中可发现的独立契约，至少声明：

```ts
interface ConversionDefinition {
  conversionId: string
  fromTypeId: string
  toTypeId: string
  loss: "lossless" | "lossy"
  determinism: "deterministic" | "environment-dependent"
  effects: Array<"filesystem" | "network" | "process" | "gpu" | "host-resource">
  supportedRepresentations: string[]
}
```

编辑器连线只在 TypeId 精确相同或存在已注册 conversion 时成立；不按 schema 形状、字段名或 TypeScript structural compatibility 自动转换。即使是看似安全的 number widening，也要遵守目标范围和精度规则。lossy conversion 必须在 UI 中可见，自动插入时要落为真实节点/边数据，不能只存在于执行器隐式逻辑里。

插件注册自定义类型时必须提供：版本化 TypeId、bundled schema、schema digest、允许的 representations、文档 annotation、兼容/迁移声明和测试向量。宿主拒绝命名空间冒充、同一 TypeId 不同摘要、递归/复杂度超限 schema、远程 `$ref`、未批准 vocabulary，以及把持久类型只实现成临时 handle 的注册。

## 8. 文档生成与单一真相源

可以直接生成文档，但前提是把“机器约束”和“人类说明”放在同一 TypeDefinition：

- 端口页：TypeId、标题、description、必需/可空、联合分支、约束和 examples；
- representation 页：inline/blob/stream/handle 的可用性、大小阈值、生命周期和安全说明；
- conversion 图：所有 from/to、loss、effects、determinism；
- 插件 SDK：从同一 registry 生成 Go 类型/validator、TS discriminated unions、WIT 定义和 Protobuf adapter，而不是维护四份手写契约。

生成器必须固定 JSON Schema dialect、Yotta vocabulary 版本和 format assertion profile；生成结果带 `typeId + schemaDigest + generatorVersion`，CI 对生成物和 registry 做一致性检查。文档 annotation 不参与值摘要，schema 内容本身则参与 schema digest。

## 9. 会改变最终 API 的开放决策

以下不是规范能替 Yotta 决定的事项，应在实现前形成 ADR：

1. **TypeRef 是否内嵌 schemaDigest**：建议执行/持久化边界必须 pin，编辑器 registry 内部可以用 typeId 查当前定义，但发布物必须同时记录 digest。
2. **JSON number profile**：建议只允许有限 binary64；int64/u64/bigint/decimal 一律用专门 TypeId 的规范十进制字符串。若允许任意精度 JSON number，JCS 就不再足够，需要另选 canonical encoding。
3. **inline 阈值和 content store**：阈值应由宿主策略按 transport 调整，而非写进类型语义；但所有 adapter 必须有硬上限和配额。
4. **handle 的能力粒度**：建议 token 绑定具体操作集合，而不仅是资源 ID；是否允许插件间委托需要单独授权协议。
5. **schema 兼容策略**：TypeId 版本何时递增、何时仅换 digest必须定义。建议任何会改变既有值验证、语义或转换结果的修改都发布新 TypeId。

## 10. 建议验收标准

- 同一测试矩阵覆盖 absent、present-null、present-default-value、联合分支、未知 TypeId、schema digest mismatch。
- Go 与 TS 对所有内建类型执行 JSON round-trip 和 canonical digest golden tests；覆盖 int64 边界、Unicode 非规范等价串、对象键顺序、重复键、`-0`、NaN/Infinity。
- Protobuf adapter 覆盖 unknown field、optional presence、Any allowlist、ProtoJSON 往返损失和消息大小限制。
- Wasm adapter 覆盖 variant、option/result、own/borrow、handle 越权/过期/跨实例使用。
- Blob/stream 覆盖摘要不符、size 不符、取消、背压、生产者失败、消费者提前退出、配额和 chunk 重排/重复。
- 自定义插件类型覆盖命名空间冲突、同 ID 不同 digest、远程 `$ref`、恶意递归 schema、显式 lossy conversion 展示。
- schema registry 能稳定生成端口文档、Go/TS/WIT/Proto adapter，并能从生成物追溯 `typeId + schemaDigest`。

最终建议：先冻结这份语义契约和测试向量，再迁移节点定义与 UI。若先围绕现有 `any`/JSON 对象改执行器，会把 absent/null、精度、临时 handle 和大对象复制问题固化进下一版协议。
