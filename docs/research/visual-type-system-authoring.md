# 强类型可视化工作流的类型系统与创作交互研究

## Research Read

- 研究问题：Yotta 如何在保持编译期强类型、确定执行和版本化契约的前提下，让用户能够自然地连接循环索引、状态变量、比较、日志和转换节点？
- 目标表面：工作流画布、端口连线、拖线候选菜单、节点目录、Run 状态面板、Inspector 和编译诊断。
- 用户任务：从一个已有值出发，快速找到“接下来能做什么”；读取或写入一个有类型的状态；把循环索引用于比较、计算和日志；在确实需要转换时理解转换的语义与风险。
- 触发失败：系统已经拒绝不兼容连线，却没有给出完整的运算节点、转换节点和上下文候选；`Repeat.index` 是整数，但数值比较只消费另一种数值类型；状态只能在面板中声明，不能直接拖出读取节点。
- 约束：Yotta 的 Node Contract、Data Type、Catalog 和 Compiler 是执行语义权威；前端不能成为第二套类型检查器；保存后的图必须显式、可复现、可审计，不能依赖运行时猜测或隐藏转换。

## 结论摘要

Yotta 应继续采用强类型，但需要把“强类型”从当前的精确 `TypeRef` 相等判断升级为一个完整的创作系统：

1. 分开定义 **可赋值关系**、**泛型统一**、**无损转换**、**有损转换**、**解析** 和 **类型收窄**；不能把它们都叫做 `convert`。
2. 类型关系和转换边都必须来自后端生成的同一份版本化 Authoring Projection，Compiler 与前端消费同一语义；前端不得继续复制一套简化版 `assignable()`。
3. 只有“总是成功、确定、无副作用、无损且语义不变”的单步转换可以在用户选择后自动插入；插入的转换节点必须保留在图上可见。当前安全整数 `integer -> number` 是典型候选，`number -> integer`、`string -> number` 和领域单位转换不是。
4. 常见操作应优先由泛型或重载节点覆盖，而不是逼用户先转成某个“万能类型”。例如 `Log<T>` 应直接接受可观察值，`Equal<T>` 应约束两端为同一类型，数值比较应同时支持整数和小数。
5. 从端口拖到空白处，应打开按兼容性排序的搜索菜单；选择节点后自动连线，必要时显式插入一个已说明的无损转换节点。拖到不兼容端口时要显示原因和可选修复，而不只是拒绝。
6. Run 状态需要可搜索的 Blackboard 体验：变量带类型、作用域和默认值；拖到画布默认创建读取节点，并提供明确的读取/写入选择与快捷方式。
7. 每个新增类型都必须通过一个自动生成的“能力覆盖矩阵”：字面量编辑、状态读写、日志/检查、相等性、适用运算、集合支持、转换入/出、序列化、调试显示。类型存在但无法消费，不算交付完成。

## Source Matrix

### Unreal Engine Blueprints

| 一手来源 | 可验证事实 | 对 Yotta 的启示 |
| --- | --- | --- |
| [Epic：Connecting Nodes](https://dev.epicgames.com/documentation/en-us/unreal-engine/connecting-nodes-in-unreal-engine) | 兼容端口悬停时显示可连接反馈；不兼容时显示原因；某些不同类型连接会创建一个可见的 Conversion Node；变量可以直接拖到端口。 | 连线反馈必须解释原因；自动转换应成为真实图节点；变量和端口都应是创作入口。 |
| [Epic：Blueprint Basic User Guide](https://dev.epicgames.com/documentation/en-us/unreal-engine/blueprint-basic-user-guide-in-unreal-engine) | Context Sensitive 默认开启；从已有 pin 拖出菜单时，候选会按该 pin 的上下文过滤；允许关闭过滤查看全部；文档也明确展示不同类型连接时自动创建转换节点。 | 拖线候选应以锚点端口类型、方向、通道和目标能力过滤，同时保留“显示全部”的逃生口。 |
| [Epic：Blueprint Variables](https://dev.epicgames.com/documentation/en-us/unreal-engine/blueprint-variables-in-unreal-engine) | 从变量列表拖入图会询问创建 Get 或 Set；Ctrl 拖拽创建 Get，Alt 拖拽创建 Set；也可以从 pin Promote to Variable。 | Run 状态不应只是配置表；拖拽创建读写节点、从端口提升为变量都是核心创作路径。 |
| [Epic：Blueprint Editor Settings API](https://dev.epicgames.com/documentation/en-us/unreal-engine/API/Editor/BlueprintGraph/UBlueprintEditorSettings) | 官方设置包含类型提升、Context Menu time slicing、在不兼容连接时显示兼容节点列表、显示 pin 值和上下文收藏等能力。 | 大目录搜索要异步/分片，不能阻塞画布；失败连线可转成“可用适配节点”菜单；候选排序应支持最近使用和收藏。 |
| [Epic：Resolve Wild Card Pin](https://dev.epicgames.com/documentation/en-us/unreal-engine/BlueprintAPI/RigVMController/ResolveWildCardPin) 与 [Make Array](https://dev.epicgames.com/documentation/en-us/unreal-engine/BlueprintAPI/Utilities/Array/MakeArray) | Blueprint/RigVM 存在可解析为具体类型的 wildcard pin；数组构造的元素和结果使用关联的 wildcard。 | 泛型不是 `any`。一个节点实例内同名类型变量必须统一，并在首个可靠连接后解析所有关联端口。 |
| [Epic：FTypePromotion API](https://dev.epicgames.com/documentation/en-us/unreal-engine/API/Editor/BlueprintGraph/FTypePromotion) 与 [UK2Node_PromotableOperator API](https://dev.epicgames.com/documentation/en-us/unreal-engine/API/Editor/BlueprintGraph/UK2Node_PromotableOperator) | 官方 API 为 promotable operator 维护 promotion table，并根据已连接 wildcard pins 选择 promoted type 和最佳匹配函数重载。 | `Greater`、算术等节点应由受约束的类型变量/重载集合解析，不应为每种数值类型复制 UI，也不应把缺口全推给转换节点。 |

### Unity Visual Scripting

| 一手来源 | 可验证事实 | 对 Yotta 的启示 |
| --- | --- | --- |
| [Unity：创建变量并加入 Script Graph](https://docs.unity3d.com/Packages/com.unity.visualscripting@1.8/manual/vs-add-variable-graph.html) | Blackboard 变量有作用域、类型和值；从 Blackboard 拖动 handle 到图中会创建 Get Variable 节点。 | 状态面板需要类型、作用域、搜索和直接拖拽，而不是只允许新增/删除。 |
| [Unity：Variable nodes](https://docs.unity3d.com/Packages/com.unity.visualscripting@1.8/manual/vs-variables-reference.html) | 拖入变量默认创建 Get，Alt 创建 Set，Shift 创建 Is Defined；但 Unity 该变量节点是动态类型，API 读取后需要手动 cast。 | 借鉴拖拽交互，不照搬动态变量语义。Yotta 的变量引用节点必须在选中 slot 后专化成该 slot 的精确类型。 |
| [Unity：Control nodes](https://docs.unity3d.com/Packages/com.unity.visualscripting@1.8/manual/vs-control.html) | For Loop 接收三个整数并输出当前 index；For Each 输出当前 index 和 item。 | 循环 index 是一等的整数值，应能直接进入整数运算、状态、日志和适用的泛型节点。 |
| [Unity：ConversionUtility API](https://docs.unity3d.com/Packages/com.unity.visualscripting@1.8/api/Unity.VisualScripting.ConversionUtility.html) | 官方 API 明确区分 implicit/explicit numeric conversion，并提供 `CanConvert(..., guaranteed)`、`GetRequiredConversion` 与 `TryConvert`。 | 转换元数据至少要表达隐式/显式、是否保证成功和是否可尝试；一个布尔 `compatible` 不足以驱动编辑器。 |
| [Unity：Object types](https://docs.unity3d.com/Packages/com.unity.visualscripting@1.8/manual/vs-types.html) | Visual Scripting 基于 C# 类型；Blackboard 变量先选类型，部分端口只允许正确类型连接；类型菜单支持搜索。 | 类型选择器必须搜索化；若引入宽类型，必须是明确的 `unknown/object` 契约，而不是类型信息丢失后的兜底。 |
| [Unity：Visual Scripting Preferences](https://docs.unity3d.com/Packages/com.unity.visualscripting@1.8/manual/vs-set-preferences.html) | 连线时可以淡化无兼容端口的节点；Fuzzy Finder 有搜索结果上限；运行时可显示或预测连接值。 | 不兼容候选可降权而非完全消失；大目录要有结果预算；调试 wire 应能显示实际值与精确类型。 |

### Node-RED

| 一手来源 | 可验证事实 | 对 Yotta 的启示 |
| --- | --- | --- |
| [Node-RED：TypedInput Widget](https://nodered.org/docs/api/ui/typedInput/) | 同一个字段可以显式选择 literal string/number/boolean、`msg`/`flow`/`global` 引用、JSON、表达式、环境变量或 credential；类型和值分别持久化；TypeDefinition 可提供校验与 autocomplete。 | 节点输入的“值来源”和“值类型”应在 UI 中明确，而不是把引用、字面量和表达式混成字符串；状态/消息引用选择器应支持搜索和自动完成。 |
| [Node-RED：Editor APIs](https://nodered.org/docs/api/ui/) | 编辑器把 TypedInput、SearchBox、EditableList、TreeList 作为一等扩展组件。 | 搜索不是节点目录的附加项，而应复用于状态、类型、端口、节点和引用选择。 |

Node-RED 的 TypedInput 主要解决配置字段的值来源，不提供强类型数据边或编译期泛型统一。因此它适合参考输入控件，不适合成为 Yotta 连线类型语义的依据。

### LabVIEW

| 一手来源 | 可验证事实 | 对 Yotta 的启示 |
| --- | --- | --- |
| [NI：Using Wires to Link Block Diagram Objects](https://www.ni.com/docs/en-AS/bundle/labview/page/using-wires-to-link-block-diagram-objects.html) | wire 的颜色、样式和粗细表达数据类型；不兼容连接成为带红叉的 broken wire；可转换的不同类型会在消费端显示 coercion dot；官方警告某些 coercion 会增加内存/时间并降低精度。 | 类型必须有不依赖颜色的文本/图标提示；即使系统允许隐式 coercion，也必须可见；有性能或精度代价的转换不能静默。 |
| [NI：Coercion Dots](https://www.ni.com/docs/en-US/bundle/labview/page/coercion-dots.html) | 相近数值表示可自动 coercion，但端口保留可见标记；符号、精度和数组规模会影响正确性或成本。 | 转换等级必须包含精度、范围和成本；“可转换”不代表“可静默”。 |
| [NI：Optimizing LabVIEW Embedded Applications](https://www.ni.com/en/support/documentation/supplemental/21/optimizing-labview-embedded-applications.html) | NI 建议在时间敏感代码中避免自动数值转换，并使用显式 Conversion functions。 | 自动转换策略必须保守；影响精度、性能或表示的转换应是显式节点。 |
| [NI：Type Cast and Convert Function Difference](https://knowledge.ni.com/KnowledgeArticleDetails?id=kA00Z000001DcZ9SAK&l=en-US) | Type Cast 是重解释位，Convert 是转换到新表示并尽量保持值，两者不是同一种操作。 | Yotta 必须把“表示重解释”与“值转换”分成不同且高风险等级不同的节点；普通自动插入绝不能使用 reinterpret cast。 |
| [NI：Malleable VIs](https://www.ni.com/docs/en-US/csh?context=lvcore_lvconcepts_malleable_vis_intro) 与 [Type Specialization Structure](https://www.ni.com/docs/en-US/csh?context=lvcore_glang_type_specialization_structure) | terminal 可适配调用处类型，编译期按类型约束选择实现。 | 通用节点需要“一个创作表面 + 受约束的具体实现选择”，不能只靠固定端口类型。 |

### 文本类型系统与数据规范对照

| 一手来源 | 可验证事实 | 对 Yotta 的启示 |
| --- | --- | --- |
| [TypeScript：Generics](https://www.typescriptlang.org/docs/handbook/2/generics.html) | `identity<T>(arg: T): T` 保留输入输出之间的同型关系；constraint 限制 `T` 必须具有某种能力；类型参数通常可以从输入推断。 | Yotta 的类型变量必须表达端口之间的关系，并由连接推断；constraint 必须有机器可执行语义，不能只是字符串注释。 |
| [TypeScript：`unknown` 与 narrowing](https://www.typescriptlang.org/docs/handbook/release-notes/typescript-3-0.html) | 任意值可以赋给 `unknown`，但 `unknown` 在断言或控制流收窄前不能赋给具体类型或执行具体操作。 | 若 Yotta 需要动态边界，应使用安全的 `unknown/json` + 显式验证/收窄节点，不能引入会绕过检查的 `any`。 |
| [TypeScript：Type Assertions](https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#type-assertions) | 类型断言在运行时没有检查，也不会转换数据。 | UI 中不得把“断言”“验证”“转换”混成一个动作；断言若存在必须显著标为不安全，并且不能由编辑器自动添加。 |
| [GraphQL 规范：Input Coercion](https://spec.graphql.org/October2021/#sec-Input-Coercion) | coercion 规则必须可观察且明确；无法匹配规则时在执行前报错；Int 输入不接受数值字符串，Float 可以接收 Int，Float 到 Int 的有损截断不被接受。 | 只允许规范化的定向转换；`integer -> number` 可无损，反向必须显式且定义舍入/越界/失败；字符串解析是独立的 fallible 节点。 |
| [JSON Schema 2020-12 Validation：`type`](https://json-schema.org/draft/2020-12/json-schema-validation#name-type) | `integer` 匹配零小数部分的 number；`number` 包含整数。 | JSON Schema 的值集合关系可以证明数值包含关系，但不能直接替代 Yotta 的领域类型关系；例如 duration 虽由 integer 表示，也不能自动当作无单位整数。 |
| [Go Specification：Conversions](https://go.dev/ref/spec#Conversions) | 非常量数值转换可能截断、舍弃小数或舍入，并且某些溢出没有信号；常量只有可表示时才可转换。 | runtime 语言“允许 cast”不等于工作流可以静默转换。Yotta 应显式定义范围、舍入、溢出和失败通道。 |

## Patterns

### 1. 把类型语义分成六种关系

编辑器和 Compiler 应返回结构化关系，而不是一个 `valid: boolean`：

| 关系 | 含义 | 是否改变保存图 | 示例 | 默认交互 |
| --- | --- | --- | --- | --- |
| `exact` | 同一已解析类型 | 否 | `integer -> integer` | 直接连线 |
| `assignable` | 类型系统证明可安全替代，不执行转换 | 否 | 将来若有名义子类型/只读容器协变 | 直接连线，并在 tooltip 解释关系 |
| `generic-bind` | 为当前节点实例绑定类型变量 | 否，但节点端口显示会专化 | `T -> Equal<T>.a` | 直接连线，刷新关联端口类型 |
| `lossless-convert` | 需要执行确定且无损的值转换 | 是，插入转换节点 | safe `integer -> number` | 候选中明确标注“一并添加转换” |
| `lossy-or-fallible` | 可能丢失信息或失败 | 是 | `number -> integer`、`string -> number` | 只提供显式节点，要求用户选择策略 |
| `narrow-or-assert` | 从动态/联合类型证明具体类型，或绕过证明 | 是 | `json -> Point` validate、unsafe assert | 验证节点可正常使用；断言节点必须高风险且永不自动插入 |

解析字符串、改变单位、读取资源、跨 durable/runtime carrier、获取 capability 都不是普通类型 coercion。它们必须是显式、可失败或带 effect/capability 的节点。

### 2. 采用名义类型核心，显式声明少量安全关系

Yotta 的 `TypeRef(typeId, semanticDigest)` 适合继续作为名义身份。不能仅因两个 JSON Schema 都是 `integer` 就互相可赋值，否则会把 `duration-milliseconds`、枚举、按键码和普通计数混为一谈。

建议在版本化 Catalog/Authoring Projection 增加由后端封装的关系声明：

```text
TypeRelation {
  source: TypePattern
  destination: TypePattern
  kind: exact | subtype | lossless-conversion | lossy-conversion | parser | narrowing
  nodeRef?: NodeRef
  total: boolean
  deterministic: boolean
  preservesValue: boolean
  preservesSemanticDomain: boolean
  failureCodes: string[]
  cost: integer
  autoInsert: boolean
}
```

规则：

- 不从 JSON Schema 自动推导领域类型关系；Schema 只用于验证值域。
- `integer` 当前限制在 JavaScript safe integer 范围时，可以显式声明到 IEEE double `number` 的无损边。
- `duration-milliseconds -> integer` 即便底层表示相同，也默认不是可赋值关系；需要领域明确声明的“取毫秒值”节点。
- 列表初期保持不变性；只有证明容器只读且元素关系安全时再引入协变，避免把 `List<Subtype>` 写入 `Supertype` 后破坏类型。
- 关系和转换节点都绑定 semantic digest；Catalog 变化后重新编译，不能用节点标题或 `typeId` 后缀猜测。

### 3. 泛型端口是关联类型，不是万能输入

同一个节点实例中的 `T` 必须统一：

- `Equal<T>(a: T, b: T) -> boolean`：连接 `a` 后，`b` 和节点显示解析为同一个类型。
- `Select<T>(condition, whenTrue: T, whenFalse: T) -> T`：三个数据端口共享绑定。
- `ForEach<T>(items: List<T>) -> item: T`：连接集合后解析 item。
- `StateRead<T>`/`StateWrite<T>`：选择状态 slot 后由 slot 声明绑定 `T`，而不是让任意边反向决定状态类型。
- `Log<T: Observable>`：消费可安全展示的类型，不要求用户先转 string。

Constraint 必须成为 Catalog 中有定义的 capability set，例如：

```text
Equatable     // 有确定的相等语义
Ordered       // 有全序或明确声明的偏序语义
Numeric       // 支持适用算术族
Observable    // 有脱敏且有界的调试显示
Durable       // 可安全写入 Run state
InlineJSON    // 可作为纯数据节点输入
```

Compiler、候选搜索和文档生成都必须解析同一 constraint registry。未知 constraint、无任何成员类型的 constraint、或 runtime adapter 未覆盖的 constraint 都应在 Catalog 构建时失败。

### 4. 数值系统需要语义闭包，而不是零散转换

建议最小数值语义如下：

| 操作 | integer | number | mixed integer/number |
| --- | --- | --- | --- |
| `+ - *` | 输入/输出 integer，越过 safe range 走声明的失败通道 | 输入/输出 finite number | integer 通过可见无损转换提升为 number |
| `/` | 输出 number，除零失败 | 输出 number，除零/非有限失败 | 提升为 number |
| `%` | 分别提供 integer modulo 与 number remainder，名称/说明区分 | 明确定义 remainder 语义 | 不隐式混合，除非产品明确选择 number 语义 |
| `< <= > >=` | 直接支持 | 直接支持 | integer 可无损提升到 number |
| `== !=` | `Equal<T: Equatable>` | `Equal<T: Equatable>` | 默认要求同型；候选可建议提升 integer |
| floor/ceil/round | 对 integer 通常无意义，候选中降权或隐藏 | 输出策略必须明确：number 或 checked integer | 不适用 |

循环 `index`、文本长度、集合长度、重试次数和数组索引应统一产出基础 `integer`。它们应立即可用于 integer 比较、integer arithmetic、typed state 和 `Log<T>`。

不要用一个 `To String(any)` 节点填补所有操作缺口。日志是观察行为，字符串转换是业务数据变换，两者的错误、脱敏和大小限制不同。

### 5. 无损自动插入必须同时满足全部条件

只有转换边满足以下条件，拖线候选才可以提供“一步完成”：

1. 单步且存在唯一最低成本路径；有歧义就让用户选择。
2. 对源类型的所有合法值总是成功。
3. 数学值和领域语义都不丢失。
4. 确定、无副作用、无 capability、无外部 I/O。
5. 不改变 durable/runtime carrier，不产生资源 lease。
6. runtime 有实现，Compiler 有验证，Authoring Projection 有同 digest 描述。
7. 转换节点会写入 Source 并在画布可见；Undo 一次撤销节点、两条边和原始操作。

禁止自动插入的典型情况：

- 浮点转整数（需要 trunc/floor/ceil/round 策略与范围检查）。
- 字符串解析为数值、布尔、JSON、日期、枚举。
- 任意值 stringify（可能丢失类型并泄露敏感信息）。
- JSON/unknown 转领域对象（需要 schema validation 或 discriminator narrowing）。
- 单位、坐标系、编码、时区、图像格式转换。
- blob/stream/handle 或任何 capability/effect 边界。
- reinterpret cast、unsafe assertion。

### 6. 从端口出发的候选菜单

从端口拖到空白画布时，菜单应保留锚点上下文，并按以下组排序：

1. **直接使用**：exact、assignable、generic-bind。
2. **无损适配**：显示目标节点和将自动加入的转换节点，例如“`大于` · 通过 Integer → Number”。
3. **显式转换**：有损、可失败或需要策略的转换；先创建转换节点，不假装已经完成目标操作。
4. **其他节点**：用户主动切换“显示全部”后出现。

候选项至少显示：节点名称、匹配端口、解析后的签名、匹配级别、是否插入转换、失败可能性和目标平台可用性。搜索应覆盖标题、别名、标签、端口名、类型名和常用动词，并支持中英文关键词。

完成选择后：

- 唯一最佳输入端口：创建节点并自动连线。
- 多个同等级输入端口：在候选项中展开端口，不要猜。
- 泛型绑定：立即刷新节点所有关联端口的类型标签和颜色。
- 无损适配：创建紧凑但真实的 conversion node；默认可折叠视觉尺寸，但不能成为隐藏 edge metadata。
- 无候选：保留搜索框并显示具体原因，例如“当前值是 Integer；大于节点目前只接受 Number”，同时提供可用的修复动作。

尝试直接连到不兼容端口时，反馈应分层：

- 端口高亮和图标提供即时可连/不可连反馈。
- 文本说明具体的 source type、destination type 和原因，不依赖颜色。
- 若存在转换路径，显示“一并添加转换节点”；若需要策略，打开转换选择器。
- Esc 取消、键盘可遍历候选、焦点回到锚点；菜单构建应支持分片/异步，避免大 Catalog 阻塞画布。

### 7. Run 状态应成为可搜索的 typed Blackboard

每个状态声明至少展示：名称、精确类型、作用域、默认值摘要、引用数和可变性。交互建议：

- 面板顶部统一搜索，按名称、类型、作用域过滤；大量变量使用虚拟列表。
- 从状态拖到画布默认创建 `Read State`；拖拽中或放下时提供“读取 / 写入”选择。
- 快捷方式可以借鉴成熟系统，但必须同时有可发现菜单：Ctrl 拖读取、Alt 拖写入只是加速器。
- 从一个输出端口执行“提升为 Run 状态”，创建同类型 slot，并按用户选择创建写入节点；不能静默改变原图执行顺序。
- 从状态拖到兼容数据输入时，直接创建读取节点并连线；拖到输出附近则建议写入节点。
- 变量改类型前列出所有引用和修复预览；不能让既有引用悄悄变成动态类型。
- 删除仍被引用的变量时，提供“查看引用”，而不是只有错误字符串。

状态读取节点的输出类型必须由所选 slot 决定。节点目录不需要为每种类型复制一个静态条目，但放到图上的实例必须解析为具体类型并可编译。

### 8. 字面量、引用与表达式要显式区分

借鉴 TypedInput，但保持 Yotta 强类型：

- 一个数据输入可以来自 edge、typed literal、Run state reference、graph input 或受支持的 expression；来源是独立的 binding kind。
- 字面量编辑器由目标精确类型/Editor Adapter 决定；`"42"` 不因目标是 integer 就自动解析。
- 选择状态引用时只列出 exact/assignable 的 slot；可无损转换的 slot 放入单独分组并说明将添加什么。
- expression 的结果类型必须由编译器推导或声明验证；不能以字符串逃逸类型系统。
- credential、target、resource handle 等仍走现有安全契约，不进入普通状态或 literal 选择器。

## Local Application

### 当前实现中值得保留的基础

- `internal/datatype.TypeExpression` 已有 `ref`、`list`、`union` 和 `variable`，可以承载名义类型、参数化列表和节点实例内的类型变量。
- `TypeRef` 同时固定 `typeId` 与 `semanticDigest`，适合保证保存图和 Catalog 的确定性。
- Node Authoring Projection 已集中投影端口类型、颜色、control、lifecycle、representation、carrier 和 resource lease，正确方向仍是后端生成、前端消费。
- `Repeat`/`ForEach` 已声明精确 integer index，Run state 也已经是 typed declaration；问题主要在操作覆盖和创作入口，而不是需要退回动态类型。

### 当前缺口

1. `datatype.Assignable` 的具体 ref 只接受完全相同的 `TypeRef`；没有 subtype 或转换关系。
2. `TypeExpression.Variable` 可以携带 `constraints`，但当前 `MatchResolved` 绑定变量时没有执行这些 constraint，属于“能描述、不能证明”。
3. 前端 `connectionCompatibility.ts` 复制了简化 assignability 逻辑，且对 variable 直接返回 false；Compiler 与编辑器可能给出不同答案。
4. Authoring Projection 没有转换边、转换等级、总/部分函数、失败码、成本或 auto-insert 元数据，前端无法可靠排序转换候选。
5. 基础 number 与 safe integer 是不同名义类型；JSON Schema 能验证它们的值域，但当前操作节点覆盖不闭合。若比较节点只接受 number，integer index 即使数学上可比较也无法直连。
6. 状态面板已经能声明 typed slot，但缺少搜索、拖拽创建读取/写入、从端口提升状态和引用导航。

### 推荐的模块边界

```text
Data Type definitions
       + Constraint registry
       + Conversion relation registry
       + Node contracts / overloads
                    │
                    ▼
        sealed Catalog + Authoring Projection
             │                       │
             ▼                       ▼
 Compiler Type Engine        Frontend Type Client
 (唯一判定/求解实现)        (查询/展示，不重写规则)
             │                       │
             └──── diagnostics ──────┘
```

实现上应把类型判定做成可查询的后端/生成式服务边界，例如针对一个锚点返回已排序的 `ConnectionPlan[]`。前端负责搜索、展示、选择和应用原子 authoring patch；它不自行寻找转换路径或解释 constraint。

`ConnectionPlan` 应包含最终图变更预览：直接 edge、绑定的类型变量、要插入的 node refs/ports、诊断和成本。实际应用仍通过 authoring engine 再验证，防止 Projection 过期或恶意输入。

## 覆盖矩阵与验收

### 类型能力覆盖矩阵

由 Catalog 自动生成下表，不靠手写清单：

| 能力 | 基础标量要求 | 领域/资源类型要求 | 示例验证 |
| --- | --- | --- | --- |
| literal/default editor | 必须 | 适用时由 Editor Adapter 提供 | integer 能输入且范围校验 |
| state read/write | 所有 durable 类型必须 | runtime-only 明确禁止并解释 | index 可写入 integer state |
| debug inspect | 必须有有界、脱敏展示 | handle 只显示 metadata | index 可直接检查 |
| log/observe | 所有 `Observable` 类型 | secret/credential 永不满足 | `Log<Integer>` 可连接 |
| equality | 所有 `Equatable` 类型 | 不可比较类型不声明 | 同型端口统一 |
| ordering | `Ordered` 类型 | 领域显式 opt-in | integer/number 都可比较 |
| arithmetic | `Numeric` 类型的声明闭包 | 单位类型使用领域节点 | integer add 输出 integer |
| collection | 可作为 list element 的类型 | runtime lease 受限制 | `List<T> -> ForEach<T>` |
| conversions | 每条边都有 runtime + contract + projection | 高风险边有 failure/capability | integer→number 一致 |
| serialization | durable 类型必须 round-trip | runtime-only 禁止持久化 | state reopen 类型不漂移 |
| debug trace | 类型和表示可追踪 | blob/stream 有预算 | 节点输出显示精确 TypeRef |

### 自动化验证规则

1. **Catalog seal tests**：未知 constraint、重复/歧义 auto-insert 边、负成本、无 runtime node 的转换、转换节点签名与声明不一致都失败。
2. **类型关系性质测试**：exact 自反；lossless auto edges 必须在合法值 corpus 上保持规范化值；禁止把 lossless 环形成多条等价最低成本路径。
3. **Compiler/editor parity**：从同一 Projection 生成 source/target/port 组合，编辑器展示的每个 ConnectionPlan 都必须被 authoring engine 和 Compiler 接受；拒绝理由 code 一致。
4. **泛型求解矩阵**：`Equal<T>`、`Select<T>`、`ForEach<T>`、state nodes 的绑定顺序互换后得到相同结果；冲突端口提供稳定诊断。
5. **节点覆盖门禁**：新增 TypeRef 时自动检查其声明 constraint 对应的节点族、literal/editor、state、observe 和 serialization 覆盖；允许明确 waiver，但 waiver 必须有理由和到期版本。
6. **转换节点合约测试**：边声明的 total/lossless/fallible 与 runtime 行为相符；fallible 节点声明稳定 failure code 和 error route。
7. **前端创作测试**：从 Repeat.index 拖线搜索 `大于`、`日志`、`状态`；验证候选排序、自动连线、转换预览、Undo 原子性、键盘操作和非颜色提示。
8. **规模测试**：在大 Catalog、大状态列表下，搜索与拖线菜单不阻塞画布；结果有稳定上限、分页/虚拟化和取消机制。

### 必须通过的用户旅程

```text
Repeat.index
  ├─ 拖到空白处 → 搜索“日志” → Log<Integer> → 自动连线
  ├─ 拖到空白处 → 搜索“大于” → Integer overload → 自动连线
  ├─ 拖到 Number-only 端口 → 提示 Integer → Number 无损转换
  │                          → 用户确认 → 插入可见转换节点并连线
  └─ Promote to Run state → 建立 integer slot → 插入写入节点

Run state: retryCount (integer)
  ├─ 搜索可找到
  ├─ 拖到画布 → Read State<Integer>
  ├─ Alt 拖 → Write State<Integer>
  └─ 删除时若被引用 → 查看引用并定位节点
```

## 不建议采用的方向

- 不回退到所有端口都接收 `json/any`；这只会把错误推迟到运行时。
- 不为每个缺口临时增加一个转换节点而没有全局关系图和覆盖门禁。
- 不让前端通过颜色、标题、schema primitive 或 type ID 字符串猜类型关系。
- 不把自动转换藏在 edge metadata；用户必须能看到、选择、撤销和调试转换。
- 不把日志强制等同于 stringify；观察节点应有自己的泛型、安全和预算语义。
- 不使用运行时值来改变已保存端口类型；类型变量应在编译/authoring 阶段解析。
- 不把 constraint 保存为未执行的自由文本；未定义约束应立即拒绝 Catalog。

## Next Step

建议按一个阶段完成再批量验收：

1. 写并冻结 Type Semantics ADR：基础类型、名义关系、constraint、泛型统一、转换分类、数值提升、null/absence 和 carrier 边界。
2. 扩展 Catalog/Authoring Projection，加入 constraint registry、conversion graph 和 `ConnectionPlan`；删除前端自有的类型判定权威。
3. 先补闭环节点族：Integer/Number 算术与比较、checked number-to-integer 策略节点、严格 string parsers、`Log<T: Observable>`、typed state read/write。
4. 重做作者交互：端口拖线候选、兼容性排序、显式转换预览、状态搜索/拖拽、Promote to State、引用导航和原子 Undo。
5. 建立 Catalog 生成的类型 × 能力覆盖矩阵以及 Compiler/editor parity 门禁。
6. 阶段末一次运行完整门禁，再用上述两个用户旅程做桌面 smoke；中途只运行能快速暴露当前改动错误的定向检查。
