# Yotta 3.1 已安装进程包：无 shell 执行、Windows LPAC 与进程树约束

> 调研截点：2026-07-16。仅使用 Go 标准库、Microsoft Windows 与 OWASP 的官方文档。文中的“Yotta 应当”是基于这些一手资料形成的架构建议，不是上游项目或供应商的承诺。

## 决策摘要

Yotta 3.1 不应把旧 `RunProgram(Target, Args, WorkingDir)` 包装成一个新名字继续保留。推荐冻结以下边界：

1. **工作流只能调用已安装、内容寻址、启动时密封的进程包 operation。** executable、完整包摘要、平台/架构、结构化 argv 规格、环境、工作目录、资源预算和权限都属于安装事实；工作流不能提供 executable、裸 command line、环境变量名、工作目录或任意参数数组。
2. **禁止 shell 但不把“无 shell”当作沙箱。** `os/exec` 不解释管道、重定向、glob 或 shell 元字符，但被调用程序仍会解释自己的参数；OWASP 明确把 argument injection 视为独立风险。因此 variable argument 必须进入具名、typed、allowlisted 的单参数槽位，不能进入字符串模板。
3. **Windows production provider 必须用 LPAC/AppContainer 与 Job Object 共同启动。** LPAC 收回 ambient 文件、registry、COM、网络、credential、process/window 权限；Job Object 收拢进程树并执行 memory/process/CPU/kill 预算。两者作用不同，不能互相替代。
4. **3.1 首版只支持单进程包。** `PROCESS_CREATION_CHILD_PROCESS_RESTRICTED`、`ActiveProcessLimit = 1`、无 breakaway flag，并通过 `PROC_THREAD_ATTRIBUTE_JOB_LIST` 在创建时原子加入 Job。需要 child process 的包在 3.1 admission 阶段明确 unsupported，不提供普通用户 token 的回退执行。
5. **失败即关闭能力。** LPAC、Job、精确 handle list、包摘要验证或 budget 安装任一失败，都返回稳定的 `process.isolation_unavailable` / `process.package_invalid`，绝不退回 `exec.Command`、`ShellExecute` 或主进程内执行。
6. **跨平台只共享 contract，不共享不成立的安全承诺。** Windows provider 完成之前，Linux/macOS 只有在实现等价的 filesystem/network/process-tree/resource isolation 后才能声明该 host capability；否则调度结果是 unsupported。

## “无 shell”是否足够

不够，但它仍是必要条件。

Go 官方说明 `os/exec` 刻意不调用系统 shell，也不实现 shell 常见的 glob、环境展开、pipeline 和 redirection。这意味着 `&`、`|`、`>` 等字符不会自动变成第二条命令或重定向语法。[Go `os/exec` overview](https://pkg.go.dev/os/exec#pkg-overview)

它只解决“shell 将数据重新解释为语法”这一层，并没有解决下面几层：

- OWASP 明确指出，攻击者即使只能控制一个 argument，也可能借助目标程序自己的 option 语义触发信息泄露或代码执行；首选防御仍是不用外部命令，其次才是参数化与 allowlist validation。[OWASP OS Command Injection Defense](https://cheatsheetseries.owasp.org/cheatsheets/OS_Command_Injection_Defense_Cheat_Sheet.html)
- 可执行文件本身仍可能是恶意、被替换或存在漏洞；是否经过 shell 与其读取文件、访问网络、加载 DLL、创建子进程无关。
- 普通 `exec.Command` 启动的进程默认继承当前用户可访问的宿主资源。没有 sandbox 时，一个“正确分隔 argv”的进程依然拥有 ambient authority。
- Windows 进程接收的是一条 command-line string；Go 会按常见 `CommandLineToArgvW` 约定从 `Args` 重新组装并 quoting，但 `cmd.exe`、batch files、`msiexec.exe` 等使用不同解析规则。Go 官方因此允许调用者提供 raw `SysProcAttr.CmdLine`，而 Yotta 应反过来禁止 workflow/package 使用这个逃生口。[Go `exec.Command`](https://pkg.go.dev/os/exec#Command)

所以 3.1 的规则不是“把用户 command string 拆成 `[]string`”即可，而是：

```text
installed operation
  = exact package digest
  + exact executable relative path
  + fixed argv grammar
  + typed variable slots with positive validation
  + fixed sandbox/runtime policy
```

`cmd.exe`、PowerShell、`.cmd`、`.bat`、raw command line、`ShellExecute` association 和“任意脚本 host + 用户源码参数”都不能成为通用 Process provider 的合法 entrypoint。确需某个宿主应用集成时，应建立具名、可审查的专用 adapter capability，而不是借 Process 绕过 capability model。

## executable 与 argv 边界

### executable identity

Go 的 `Command(name, ...)` 在 `name` 没有 path separator 时会用 `LookPath` 搜索；Windows 还会依据 `PATH` / `PATHEXT` 解析候选程序。虽然现代 Go 会拒绝隐式解析到当前目录的 `ErrDot`，这仍不是安装身份验证。[Go `LookPath`](https://pkg.go.dev/os/exec#LookPath)

Yotta 应当：

- 安装时把 package 解包到 Yotta 管理的 immutable/content-addressed root；
- manifest 的 entrypoint 只能是包内规范化 relative path，安装后解析为 exact absolute path；
- artifact digest 覆盖 manifest、entrypoint 与全部运行依赖，启动前重新验证 sealed digest；
- Windows 调用 `CreateProcessW` 时总是把 exact absolute executable 放入非空 `lpApplicationName`。Microsoft 明确警告：`lpApplicationName = NULL` 加上含空格且未正确引用的 command line 可能被解析成另一个 executable。[Microsoft `CreateProcessW`](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-createprocessw)
- 不使用 `PATH`、`PATHEXT`、current directory、App Paths 或文件关联做解析，也不允许 operation 在运行时换 entrypoint。

### structured argv，不是字符串模板

推荐 manifest 用一个结构化序列描述每个 argv element：

```yaml
operation: image.resize
entrypoint: bin/image-tool.exe
arguments:
  - literal: resize
  - literal: --
  - value: source
    schema: { type: string, format: staged-relative-path, maxLength: 240 }
  - prefixedValue:
      prefix: --width=
      input: width
      schema: { type: integer, minimum: 1, maximum: 8192 }
acceptedExitCodes: [0]
```

具体规则：

- 一个 `value` 必须产生恰好一个 argument；不能通过空格或 quoting 增加 argument 数量。
- `literal`、prefix、argument 顺序与 option terminator 都是安装事实。只有目标工具官方保证支持 `--` 时才可使用，不能把它当作跨程序通用语义。
- variable slot 必须有 Yotta `TypeRef`、长度/数量上限与 positive validation。enum、整数范围、staged relative path 等优先于自由字符串。
- 对 repeated argument 必须在 manifest 显式声明并设 `maxItems`；默认不允许。
- 不接受 raw `[]string`、单个 `Args` 字符串、用户提供的 quote/escape mode 或 `%VAR%` 展开。
- package 声明的 Windows argv mode 必须与 entrypoint 真实 parser 一致；使用非标准 command-line parser 的工具需要专用 adapter 和测试，不能走 generic provider。

这样能消除 shell injection 和“增加 argv element”的路径，但不能消除工具自身把某个合法单参数解释为配置文件、response file、plugin 或 URL 的风险。slot schema 必须按目标 operation 的真实语义限制值；sandbox 是最后的权限边界。

## env、cwd 与 handle 继承边界

### environment

`exec.Cmd.Env == nil` 会继承 Yotta 主进程环境；Windows 上即使设置了非 nil `Env`，Go 仍会在未显式置空时补入 `SYSTEMROOT`。[Go `Cmd.Env`](https://pkg.go.dev/os/exec#Cmd)

Yotta 不应把 ambient environment 复制给进程包。建议 Windows launcher 直接构造 deterministic Unicode environment block，只包含：

- 固定且由 host 生成的 `SystemRoot`；
- 指向本 attempt 私有 sandbox 目录的 `TEMP`、`TMP`、`LOCALAPPDATA`；
- manifest 中安装时密封的非 secret 常量，且变量名来自固定 allowlist。

默认不得包含 `PATH`、`PATHEXT`、`USERPROFILE`、Yotta 主进程 secret、proxy、credential、token 或 workflow 输入。workflow 数据通过 typed request/argv slot/stdin frame 进入，不通过环境变量。

### current directory

Go `Cmd.Dir == ""` 与 Windows `CreateProcessW(lpCurrentDirectory = NULL)` 都表示继承调用进程 current directory。[Go `Cmd.Dir`](https://pkg.go.dev/os/exec#Cmd) [Microsoft `CreateProcessW`](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-createprocessw)

因此 Yotta 必须总是传入一个 absolute、per-attempt sandbox directory，不能让 manifest 或 workflow 指向 workspace、项目目录、用户目录或任意 host path。输入文件应由 broker 复制/物化成受预算约束的 staged artifact；输出只从 sandbox 中按 manifest 声明收回并重新验证。不能把“cwd 在 sandbox 内”误认为 filesystem sandbox，真正的访问边界仍由 LPAC token 与 DACL 决定。

### inherited handles

Windows `CreateProcessW(bInheritHandles = TRUE)` 默认会复制调用进程所有标记为 inheritable 的 handles；Microsoft 特别指出多线程进程中这会造成继承竞态。`PROC_THREAD_ATTRIBUTE_HANDLE_LIST` 可把继承范围缩到显式列表，并要求列表中的 handle 本身可继承。[Microsoft `CreateProcessW`](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-createprocessw) [Microsoft `UpdateProcThreadAttribute`](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute)

Yotta 应当只继承：

- attempt stdin read end；
- bounded stdout write end；
- bounded stderr write end；
- 未来协议确实需要时的单个、具名 IPC handle。

父进程持有的 pipe ends 必须清除 inherit flag；database、log file、credential store、network socket、window/process/job handle 都不能进入 child list。`STARTF_USESTDHANDLES` 下的三个 standard handles 必须是有效且与 exact handle list 一致的值。创建完成后父子双方立即关闭不用的 pipe ends。

## Windows：如何防止子进程逃逸并实施预算

### 创建时原子加入 Job

Job Object 把关联进程作为一个单元管理。默认情况下，job 内进程通过 `CreateProcess` 创建的 child 会进入同一个 job；`BREAKAWAY_OK` 与 `SILENT_BREAKAWAY_OK` 会改变这一行为，所以 Yotta 不得设置它们。[Microsoft Job Objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects)

不能采用“先正常启动，之后 `AssignProcessToJobObject`”的顺序，因为 entrypoint 在赋值前可能已经执行。Windows 10 / Server 2016 起，`PROC_THREAD_ATTRIBUTE_JOB_LIST` 能在创建时把新进程加入给定 jobs；这是 Yotta Windows 3.1 唯一允许的 production 路径。[Microsoft `UpdateProcThreadAttribute`](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute)

### 3.1 的单进程策略

每个 attempt 独立 Job，至少设置：

```text
JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION
JOB_OBJECT_LIMIT_ACTIVE_PROCESS, ActiveProcessLimit = 1
JOB_OBJECT_LIMIT_PROCESS_MEMORY
JOB_OBJECT_LIMIT_JOB_MEMORY
JOB_OBJECT_LIMIT_PROCESS_TIME
no BREAKAWAY_OK
no SILENT_BREAKAWAY_OK
```

同时把 `PROC_THREAD_ATTRIBUTE_CHILD_PROCESS_POLICY` 设为 `PROCESS_CREATION_CHILD_PROCESS_RESTRICTED`。Microsoft 说明该 policy 只有在 sandbox 阻止进程取得高权限 process handles 时才有效；这正是它必须与 LPAC 配套、不能单独使用的原因。[Microsoft child-process policy](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute)

`ActiveProcessLimit = 1` 提供第二道、由 Job 管理的限制。Microsoft 说明超过 active process limit 的关联进程会被终止且关联失败。[Microsoft `JOBOBJECT_BASIC_LIMIT_INFORMATION`](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_basic_limit_information)

若以后确实要支持多进程包，必须发布新的 isolation profile：manifest 声明精确 child 上限、所有 descendant 保持同一 LPAC/token 与 Job、无 breakaway，并新增真实 escape tests。3.1 不应提前留下一个“允许任意 child”的布尔兜底。

### memory、CPU、wall time 与强制终止

`JOB_OBJECT_LIMIT_PROCESS_MEMORY` 与 `JOB_OBJECT_LIMIT_JOB_MEMORY` 约束的是可 commit 的 virtual memory；两者独立，前者限制单进程，后者限制整个 job。[Microsoft `JOBOBJECT_EXTENDED_LIMIT_INFORMATION`](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_extended_limit_information)

Job 的 process user-time limit 不是 wall-clock deadline。Yotta 仍需父进程 timer/context 管 wall time，并在 deadline、cancel、协议错误、stdout/stderr 超限或 host shutdown 时调用 `TerminateJobObject`。Microsoft 说明这个调用会终止 job 及其 nested child jobs 中的所有进程，关联进程不能推迟或处理该终止。[Microsoft `TerminateJobObject`](https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-terminatejobobject)

`JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` 保证最后一个 job handle 关闭时终止 job 进程树，覆盖父进程异常退出/cleanup 路径。[Microsoft Job Objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects)

只用 `exec.CommandContext` 不满足以上保证：Go 默认 cancel 只对直接 `Process` 调用 `Kill`，默认 `WaitDelay` 为零；Go 还明确说明 `os.Process.Kill` 只杀该 process，不杀它启动的其他进程。孤儿 child 持有 I/O pipe 时，`Wait` 也可能一直等不到 EOF。[Go `CommandContext` and `Cmd.WaitDelay`](https://pkg.go.dev/os/exec#CommandContext) [Go `os.Process.Kill`](https://pkg.go.dev/os#Process.Kill)

父进程必须并发 drain stdout/stderr，分别与合计实施 byte caps；任何 cap 超限立即终止 Job 并关闭 pipe。不得调用无界 `CombinedOutput` 后才检查长度。最终必须等待 process/job 收敛、关闭全部 handles，并将 peak memory、exit code、termination strength、output byte count/digest 写入 attempt receipt；raw stdout/stderr 只作为受类型和大小限制的 node data，不进入通用日志。

## AppContainer / LPAC 的文件与网络隔离含义

Job Object 是资源与进程树边界，不是权限沙箱。普通 child 仍在调用用户的安全上下文中运行；Microsoft 对 `CreateProcessW` 的定义也明确说明这一点。[Microsoft `CreateProcessW`](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-createprocessw)

AppContainer 使用 Package SID、Capability SID、token 和 DACL 建立 security boundary。资源的最终访问是 user/group 权限与 AppContainer 权限的交集；没有对应 capability/DACL grant 就不能访问受保护资源。没有 network capability 的 AppContainer 不能访问网络。[Microsoft Launch an AppContainer](https://learn.microsoft.com/en-us/windows/win32/secauthz/implementing-an-appcontainer)

LPAC 比普通 AppContainer 更严格：普通 AppContainer 默认仍能访问部分 system files、registry keys 和 COM objects，而 LPAC 对 registry、COM 等也要求额外 capability。Yotta 的不可信进程包应使用 LPAC，并从零 capability 开始。[Microsoft Launch an AppContainer](https://learn.microsoft.com/en-us/windows/win32/secauthz/implementing-an-appcontainer)

具体含义是：

- **文件：**只给 exact package tree read/execute、per-attempt scratch read/write；不给 workspace、user profile、Yotta data/settings/database、credential store。AppContainer 本身不负责 staging，也不会让一个普通目录自动变成 chroot；Yotta 仍要设置精确 DACL。
- **网络：**不授予 Internet、private network 或 server capability，也不创建 loopback exemption。需要 HTTP 的工作流继续调用已安装 HTTP egress provider，不能让 Process 节点绕过 origin policy。
- **registry/COM/credential/process/window：**LPAC 不授予相关 capability；Job UI restrictions 继续限制 clipboard、desktop、global atoms、display/system settings 等 UI surface。
- **身份与状态：**推荐每个 attempt 使用独立 LPAC profile/identity 和 scratch，结束后删除。若复用 package identity，package profile 目录会成为跨 attempt 的隐式持久状态，破坏 replay 与隔离；这种共享状态只能作为将来显式声明、显式授权的新 capability，不能默认存在。
- **启动验证：**创建后、发送 request 前，父进程验证 token 确为 AppContainer + LPAC、capability set 与 sealed profile 完全一致、进程已经在预期 Job 中；验证失败先 terminate job，再报告 isolation unavailable。

Microsoft 将 AppContainer isolation 分为 credential、device、file、network、process 和 window 等维度，也说明 file/registry 与 network 权限必须明确授予。[Microsoft AppContainer isolation](https://learn.microsoft.com/en-us/windows/win32/secauthz/appcontainer-isolation)

## 建议冻结的 Yotta 3.1 process-package contract

### 安装事实

```yaml
format: yotta.process-package
version: 3.1
packageId: com.example.image-tool
artifactDigest: sha256:...
platform: windows
architecture: amd64
entrypoint:
  path: bin/image-tool.exe
  digest: sha256:...
  argvMode: command-line-to-argvw
operations:
  image.resize:
    arguments: [...]          # literal/value/prefixedValue only
    inputType: yotta.example.image-resize-input@...
    outputType: yotta.process-result@...
    acceptedExitCodes: [0]
runtime:
  isolation: yotta.windows.lpac-job/v1
  children: 0
  network: none
  filesystem: staged-only
  environment: fixed
  timeoutMillis: 30000
  processMemoryBytes: 268435456
  jobMemoryBytes: 268435456
  stdoutBytes: 262144
  stderrBytes: 262144
  scratchBytes: 67108864
```

安装器必须 canonicalize manifest、校验所有范围、拒绝 unknown fields、验证 entrypoint 在包 root 内且是 regular file，并生成包含 package/operation/runtime policy semantic digests 的 provider artifact。设置页编辑任何 executable、operation、argument schema、environment 或 budget 都会产生新安装 identity，并撤销旧 consent lineage。

### 工作流节点

一个 node instance 只绑定：

```text
installation slot + exact operation ID + operation semantic digest
```

推荐端口：

```text
exec input:   in
exec outputs: completed, failed
data input:   arguments (exact operation input TypeRef)
data outputs: exit-code, stdout, stderr
```

工作流看不到 package path、executable、cwd、env 或 sandbox path。若 operation 有文件输入，data pin 接收 blob/artifact ref，provider 把已授权内容 staged 到 attempt scratch；不能把 absolute host path 放进 argv。

进程成功启动并正常退出时，exit code 是 process result。Yotta 可按 operation 的 `acceptedExitCodes` 决定 `completed`/`failed`，但 receipt 必须保留 exit code、stdout/stderr byte count 与 digest；不能把 stderr 原文拼进 durable error。spawn failure、digest mismatch、isolation failure、timeout、cancel、budget exceed 与 output overflow 使用各自稳定 error code。

### capability、consent 与 journal

- requirement 精确归属到 node、installation、operation、artifact、target profile 与 budget digest；不能只声明宽泛的 `process.spawn`。
- 该 effect 至少是 dangerous + explicit consent；grant 绑定 exact Program、Run、node、operation 与 installation generation。
- adapter action journal 记录 operation identity、exit code、termination kind、duration、resource/output计数与内容 digest；不记录 executable host path、argv values、env、cwd、stdout/stderr 原文。
- replay 读取 recorded typed result，不重新启动进程；未知/ambiguous attempt 不自动重试。

## 对现有 Yotta 边界的直接结论

1. 删除旧 `RunProgram` 的 `Target`、`Args`、`WorkingDir`、`WindowState` 语义，不提供 converter 或 fallback。
2. “打开 URL/文档”“启动一个已安装工具”“调用 AE/UE 等宿主 adapter”是三个不同 capability，不能继续由 `ShellOpen` 一个入口混合。
3. Windows launcher 可以复用现有 Script isolation 已验证的底层原语，但 Process 必须拥有独立 package/profile/operation contract；不要复制一套稍有差异的 Job/AppContainer 实现。应抽出一个小而严格的 Windows isolated-process launcher，由 Script 与 Process 各自提供 request protocol 和 policy。
4. generic provider 不接受 shell、batch、PowerShell、raw command line、ambient user executable 或 arbitrary host path。
5. Linux/macOS 在等价 provider 完成前不注册 process isolation host feature；编译通过不等于 capability 可用。

### 与桌面应用生命周期能力的边界

以上 LPAC/Job 结论适用于“工作流把数据交给一个进程包执行并取回结果”的 **Process Node**。普通桌面自动化还需要另一种显式能力：启动或终止用户已经信任并安装的 GUI 应用。该能力不能冒充 sandbox，也不能与 Process Plugin Host 共用 provider、target kind、operation 或 consent。

Yotta 3.1 将桌面应用生命周期定义为独立的 dangerous capability：受信设置封存 exact executable path、文件内容摘要与固定逐项 argv，workflow 只持有逻辑 installation slot，不提供 executable、argument、environment、working directory、PID、URL 或文档路径。启动使用 exact executable 且不经过 shell；终止只作用于与该 sealed executable 文件身份一致的进程。由于目标应用按设计需要当前桌面用户权限，这条能力明确承认它拥有 ambient user authority，并以安装、摘要复验、精确 target、ConsentOnce、Run Grant 和 effect journal 约束“workflow 能触发哪个已信任应用”，而不是声称隔离目标应用。

第三方代码、CLI 数据处理、AE/UE adapter worker 与任何需要 stdout/stderr 结果的扩展仍属于 Process Node/Plugin Host，必须遵守本文的 LPAC + atomic Job + typed protocol 约束；不得借桌面应用生命周期 provider 绕过 sandbox。打开 URL/文档继续是另一种尚未安装的 capability，不能回到 `ShellExecute` 混合入口。

## 必须通过的安全与契约测试

1. shell 元字符只能成为一个 argument 的普通字符，不能改变 argv 数量；argument slot 的 option/response-file 绕过 fixture 被拒绝。
2. `lpApplicationName` 是 sealed absolute path；包文件替换、symlink/reparse escape、digest drift 在启动前失败。
3. child 观测不到 Yotta 主进程 `PATH`、proxy、token、user profile 或 secret environment。
4. child cwd 永远是 per-attempt scratch，且不能读取 workspace、settings、database、credential 或其他 attempt scratch。
5. child 无法访问 public、private 与 loopback network；需要网络的 fixture 只能失败。
6. inherited handle audit 只看到声明的 stdio/IPC handles。
7. 创建 child process 失败，job active process count 不超过 1；不存在 breakaway descendant。
8. cancel、deadline、memory、CPU、stdout、stderr 与 scratch budget 任一超限都会 terminate 整个 Job 并完成 reap。
9. launcher 若无法安装 LPAC、atomic Job List、handle list 或 limits，返回 isolation unavailable，且没有普通进程被启动。
10. journal 与错误投影不含 executable host path、argv、env、cwd、stdout/stderr 原文或 staged file name。
11. 相同 Program replay 使用 recorded result，不再次执行 package。
12. Linux/macOS 未安装等价 sandbox 时，admission 返回 unsupported，不调用 `os/exec` fallback。

## 结论

Yotta 3.1 的 Process 能力不应是“执行一个命令字符串”，而应是“调用一个已安装、内容寻址、operation-scoped、零 ambient authority 的进程包”。

`os/exec` 的结构化 argv 是正确的 API 基线，却不是完整安全边界。真正可审查的边界来自 exact executable identity、typed argv grammar、固定 env/cwd/handles、LPAC 的权限隔离、Job Object 的进程树与预算、admission/grant 的精确归属，以及失败时绝不降级执行。
