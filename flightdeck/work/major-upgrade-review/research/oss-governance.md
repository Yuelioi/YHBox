# Yotta 3.0：大型开源项目治理、供应链、发布与安全成熟度调研

> 调研日期：2026-07-13（Asia/Shanghai）
> 范围：治理、贡献者权利、GitHub 保护、依赖与 CI 供应链、桌面发布、漏洞响应、版本支持、可复现构建和维护者连续性。
> 方法：仓库与 GitHub API 只读审计；外部结论只采用 OSI、OpenSSF、SLSA、GitHub、CNCF、Electron、Apache 和 Reproducible Builds 项目的一手资料。本文不是法律意见。

## 一句话结论

Yotta 目前是“公开过旧版本源码、主要开发仍在本地进行的单维护者 source-available 项目”，还不能准确自称大型开源项目，也不具备可信稳定桌面发行版应有的发布链。最优先的不是再添加几个徽章，而是同时修复五个信任断点：**采用 OSI 许可证、统一项目身份并公开当前开发历史、保护主分支与发布标签、建立至少两人的维护/发布连续性、让每个发布产物可签名且可追溯到受保护源码。**

OSI 对 open source 的定义不仅要求源码可见，还要求自由再分发、允许修改和衍生作品、不得歧视商业等使用领域；Yotta 当前禁止商业使用和盈利分发的许可证直接违反这些条件。[OSI Open Source Definition](https://opensource.org/osd)；[OSI approved licenses](https://opensource.org/licenses)。

## 当前事实快照

### 已有的好基础

- 本地候选代码已有 `SECURITY.md`、威胁模型、平台支持矩阵、贡献指南、Go/Node/Rust 依赖更新配置，以及 Go 测试、vet、staticcheck、race、govulncheck、cargo-deny、Gitleaks 和许可证检查。这些是可以保留的工程资产。
- `SECURITY.md` 给出 7 天确认窗口，优于 OpenSSF Passing 对初次响应不超过 14 天的要求；但报告入口尚未启用，承诺目前不可执行。[OpenSSF Passing criteria](https://www.bestpractices.dev/en/criteria/0)
- release build 已使用部分确定性参数，例如 Go `-trimpath`；但完整工具链、安装步骤和产物集合仍不确定，因此还不能称为可复现构建。OpenSSF Silver 要求重复生成内容得到 bit-for-bit 相同结果，Gold 才明确要求 reproducible build。[OpenSSF Silver criteria](https://www.bestpractices.dev/en/criteria/1)；[OpenSSF Gold criteria](https://www.bestpractices.dev/en/criteria/2)

### P0/P1 缺口

| 优先级 | 现状证据（2026-07-13 快照） | 为什么是问题 | 3.0 要求 |
|---|---|---|---|
| P0 | `LICENSE` 禁止商业使用、盈利分发、SaaS 和付费服务 | 不是 OSI open source；无法通过 OpenSSF 最基础 FLOSS 条件 | 由权利人采用一个 OSI 批准许可证，完成历史代码与资产权利核对，并在 README、包元数据和发布物中统一声明 |
| P0 | `origin` 仍写作 `Yuelioi/YHBox.git`，GitHub 重定向到 `Yuelioi/Yotta`，Go module 与安全报告 URL 却使用 `github.com/yottaapp/yotta`；公开 `origin/main` 为 `124d48a`，本地审计时比它领先约 1,578 commits | 源码、模块、报告入口和 provenance 会指向不同主体；公众无法审阅当前开发过程。OpenSSF Passing 要求公开仓库包含发布之间供审阅的中间版本，而不是只公开最终快照 | 迁移到唯一组织/仓库身份（推荐 `yottaapp/yotta`），统一 remote、module、下载、SECURITY、更新源和签名主体；全历史 secrets/license 扫描后公开当前开发分支 |
| P0 | GitHub API 返回 `main` “Branch not protected”，rulesets 为 `[]`；直接协作者只有 `Yuelioi` 一个管理员 | 任一凭据失陷都能改源码、标签和发布；required checks 实际没有强制力；bus factor=1 | 组织化托管；至少两名独立管理员/发布者；启用 main 与 `v*` rulesets、无管理员绕过、PR/审查/required checks/防强推和防删除 |
| P0 | Actions 允许所有来源，`sha_pinning_required=false`；workflow 使用 `actions/*@vN`、`softprops/*@v2`、`dtolnay/rust-toolchain@stable`、Task `3.x`；前端 `@wailsio/runtime` 为 `latest` | tag、channel 和范围都可移动；构建同一 commit 可能执行不同代码。GitHub 明确说明完整 commit SHA 是 Action 唯一不可变引用方式 | 所有 Action 固定完整 SHA 并保留版本注释，由 Dependabot 更新；Rust/Task/Node/pnpm/Wails/NSIS 等精确固定；组织策略强制 SHA pin 与 Action allowlist |
| P0 | release workflow 只上传 `bin/Yotta.exe`；build 同时生成运行时使用的 `capture_wgc.dll` 和 `platform-tools`；现有公开 release 只含 `YHBox.exe`，`immutable=false` | 发布物不是受测的完整产品集合；名字也与当前产品不一致。缺少 auxiliary artifact 会造成发布与本地验收不等价 | 先 staging 完整安装树，再测试、签名、生成 manifest/SBOM/checksum，并发布同一批已测试字节；禁止“重新构建后发布” |
| P0 | release 未调用现有 signing task，没有 Authenticode/timestamp、SBOM、build provenance、artifact attestation 或 release approval environment | 用户无法证明下载物由 Yotta 发布，也无法追溯构建输入；Windows 会把未签名分发物视为低信任应用 | Windows exe、DLL、installer 签名并 timestamp；附 SPDX/CycloneDX SBOM、SHA-256、GitHub attestation；发布环境要求非发起人审批；启用 immutable releases |
| P1 | Private vulnerability reporting 为 `false`；Dependabot security updates、secret scanning/push protection 为 disabled；CodeQL default setup 为 `not-configured` | `SECURITY.md` 指向尚不存在的私密入口；安全控制只存在于文档而非平台 | 发布前启用并用非维护者账号验证私密报告；打开安全更新、secret scanning/push protection、CodeQL；设定安全响应角色与通知备份 |
| P1 | 缺 `CODE_OF_CONDUCT.md`、`GOVERNANCE.md`、`MAINTAINERS.md`、`CODEOWNERS`、DCO/CLA 规则、`SUPPORT.md`、`RELEASING.md`、公开 roadmap；git 历史与 collaborator 均显示单人 | 贡献接受、决策、冲突处理、发布和离任都依赖个人隐性知识 | 建立轻量但真实的治理与 contributor ladder；文档必须与 GitHub 权限、CODEOWNERS 和发布权限一致 |
| P1 | release 构建会执行 `go mod tidy`、非 frozen 的 `pnpm install`，Cargo 未显式 `--locked`；Rust `stable` 和 Task `3.x` 会漂移 | 发布过程会修改/重新解析依赖，无法证明 tag 对应固定输入，也无法可靠复建 | release 必须在 clean checkout 上使用 frozen/locked/offline-capable 安装，结束时 `git diff --exit-code`；所有生成物做 drift check |
| P1 | `SECURITY.md` 仅承诺 main 与最新 release，未定义稳定/预览渠道、EOL、回补边界和安全公告索引 | “最新”在新版本发布瞬间改变，使用者无法计划升级，也无法判断旧版是否仍安全 | 明确单一受支持稳定线、预览线免责、EOL 事件、漏洞影响范围、回补原则和发布节奏 |

上述 GitHub 设置由以下只读 API 审计得到，属于快照而非仓库文件推断：

```text
gh api repos/Yuelioi/Yotta/branches/main/protection       -> 404 Branch not protected
gh api repos/Yuelioi/Yotta/rulesets                       -> []
gh api repos/Yuelioi/Yotta/collaborators?affiliation=direct -> 仅 Yuelioi/admin
gh api repos/Yuelioi/Yotta/private-vulnerability-reporting -> enabled:false
gh api repos/Yuelioi/Yotta/actions/permissions            -> allowed_actions:all, sha_pinning_required:false
gh api repos/Yuelioi/Yotta/environments                   -> total_count:0
gh api repos/Yuelioi/Yotta/code-scanning/default-setup    -> not-configured
```

## 换成 OSI 许可证之后，还必须补什么

许可证只回答“别人可以如何使用代码”，不回答“谁能合并、谁能发布、产物来自哪里、漏洞由谁响应、项目离开创始人后能否继续”。OpenSSF 的分层标准很适合作为 Yotta 的外部成熟度合同：Passing 覆盖公开仓库、构建、测试、发布说明和漏洞报告；Silver 增加 DCO/CLA、治理、角色、连续性、roadmap、依赖监控与 bit-for-bit 重复构建；Gold 增加 bus factor、非关联贡献者、2FA、双人审查、可复现构建和安全评审。[Passing](https://www.bestpractices.dev/en/criteria/0)；[Silver](https://www.bestpractices.dev/en/criteria/1)；[Gold](https://www.bestpractices.dev/en/criteria/2)

### 1. 法律与项目身份

- **许可证决策**：若目标是最大化桌面/开发工具生态采用，Apache-2.0 是合理默认；它已被 OSI 批准，并提供明确的贡献与专利授权条款。若选择 AGPL-3.0，应把它当作商业/社区战略决策，而非“更开源”的技术升级。[OSI license list](https://opensource.org/licenses)；[Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0)
- **历史权利清查**：当前 git shortlog 只有一个作者，使重授权相对简单，但仍需检查复制进入仓库的代码、图标、字体、ADB/runtime 二进制、示例和模型输出。为 bundled dependencies 生成 `THIRD_PARTY_NOTICES`，不要把 CI 中的 license report 仅留作临时 artifact。
- **唯一身份**：组织、仓库、Go module、应用名、exe/installer 名、更新 URL、安全报告 URL、签名证书 subject 和 provenance source URI 必须一致。GitHub artifact attestation 会记录 repository、commit SHA、workflow 与 event；身份漂移会被永久写入 provenance。[GitHub artifact attestations](https://docs.github.com/en/actions/concepts/security/artifact-attestations)
- **商标与域名**：代码许可证不等于项目名/Logo 的无限授权。进入多维护者阶段后应补简明 trademark/brand use policy，并把域名、GitHub org、签名服务和恢复凭据归项目组织而非个人持有。

### 2. DCO 还是 CLA

推荐 Yotta 3.0 使用 **DCO 1.1 + 每 commit sign-off + required DCO check**，暂不引入 CLA。

| 机制 | 它解决什么 | 成本与适用条件 | Yotta 决策 |
|---|---|---|---|
| DCO | 贡献者逐 commit 声明自己有权按项目许可证提交；DCO 1.1 的完整声明由 Linux Foundation 维护 | 低摩擦、适合 inbound=outbound 的社区项目；必须让命令行、web edit、bot/squash 流程都能保留 sign-off | 现在采用；在 `CONTRIBUTING.md` 链接全文、示例 `git commit -s`，DCO check 设为 required |
| CLA | 贡献者额外授予项目实体一组版权/专利许可；Apache 用 ICLA/CCLA 来支撑基金会长期分发、争议防御和大额软件捐赠 | 需要确定法律实体、协议文本、签署/隐私/雇主授权流程；会提高首次贡献门槛 | 只有未来成立基金会、需要双许可证/重授权权利或接收大规模代码捐赠时，经法律评估再引入 |

DCO 正文：[Developer Certificate of Origin 1.1](https://developercertificate.org/)。OpenSSF Silver 明确把 DCO 作为最常见、易实施方案，同时允许 CLA 作为替代；这支持 Yotta 当前选择 DCO。[OpenSSF Silver project oversight](https://www.bestpractices.dev/en/criteria/1)。Apache 对 CLA 的用途、个人/公司协议及大额软件 grant 有清楚说明，不能简单复制一份 CLA 就假设法律问题已解决。[ASF Contributor Agreements](https://www.apache.org/licenses/contributor-agreements.html)

还要避免两个常见混淆：`Signed-off-by` 是贡献权利声明，GPG/SSH/S/MIME commit signature 是提交身份验证，两者不是同一件事；GitHub 官方也明确区分 sign-off 与 cryptographic signing。[GitHub commit signoff policy](https://docs.github.com/en/organizations/managing-organization-settings/managing-the-commit-signoff-policy-for-your-organization)；[GitHub signature verification](https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification)

### 3. 治理、角色与 CODEOWNERS

先建立轻量 maintainer council，不需要一开始模仿 Kubernetes 的组织规模，但必须写清楚：

- contributor、reviewer、maintainer、release manager、security responder 的权限与责任；
- 晋升、主动离任、长期不活跃、紧急移除和 emeritus 流程；
- 普通变更、breaking contract、治理变更、安全紧急修复如何决策；
- 冲突解决与 Code of Conduct 执行联系渠道；
- 哪些讨论可以私密（仅 embargoed vulnerability/CoC/credential incident），其余决策如何公开留痕；
- GitHub team、CODEOWNERS、branch ruleset、release environment 实际权限如何映射到文档角色。

CNCF 的治理模板认为 `GOVERNANCE.md` 应让贡献者知道项目如何运行、领导角色如何获得，并定义决策、分歧解决和维护者连续性。[CNCF Maintainer Council template](https://contribute.cncf.io/projects/best-practices/governance/templates/governance-maintainer/)。其毕业级标准进一步要求完整 maintainer lifecycle、当前维护者及责任/归属、代码与文档 ownership 对齐、security response roles，以及至少两个组织的维护者来证明 survivability；Yotta 可把这些作为“成熟阶段”而非首次公开当天的硬性 CNCF 申请目标。[CNCF graduation application template](https://github.com/cncf/toc/blob/main/.github/ISSUE_TEMPLATE/template-graduation-application.md)

`CODEOWNERS` 只是自动路由与审查门禁，不是治理文档的替代品。应至少覆盖：

```text
/.github/                 @yottaapp/maintainers
/internal/security/       @yottaapp/security @yottaapp/backend
/internal/services/mcp*/  @yottaapp/security @yottaapp/backend
/internal/workflow/       @yottaapp/workflow
/frontend/                @yottaapp/frontend
/native/                  @yottaapp/native
/build/                   @yottaapp/release
/SECURITY.md              @yottaapp/security
/GOVERNANCE.md            @yottaapp/maintainers
```

CODEOWNERS 文件自身必须有 owner，并与 ruleset 的 “Require review from Code Owners” 联动；GitHub 官方特别指出，否则 CODEOWNERS 本身可被未授权改写。[GitHub CODEOWNERS](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners)

Electron 是更贴近桌面开发工具的成熟案例：它按 API、Community & Safety、Ecosystem、Infrastructure 等职责建立 working groups，并由 Administrative WG 处理跨组冲突；Yotta 初期不必建立这么多组，但应按职责而非按个人建立可演进的角色边界。[Electron Governance](https://www.electronjs.org/governance)

### 4. GitHub 强制保护，而不是文档倡议

#### `main` ruleset（3.0 公开前必须）

- 所有改动经 PR；阻止直接 push、force-push 和删除；规则不得由管理员常态绕过。
- 至少 1 个非作者 approval；有第二位稳定维护者后启用 “require last push approval”、dismiss stale approvals 和 CODEOWNER review。成熟阶段升为敏感目录 2 人审查。
- require conversation resolution；required checks 使用 strict/up-to-date，或启用 merge queue。
- required checks 至少包括：Go test/vet/staticcheck/coverage、race group、portable-core matrix、frontend test/typecheck/lint/i18n/format/build、Rust deny/test/build、Gitleaks、govulncheck、dependency review、CodeQL、generated-contract drift、package smoke。
- required check 绑定预期 GitHub App，避免同名 status 被别的集成伪造。

GitHub branch protection 支持 required review、Code Owner review、required status checks、signed commits、linear history、merge queue、禁止绕过和限制 push；required checks 只有成功/中立/跳过后才允许合并。[GitHub protected branches](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)

#### `v*` tag ruleset 与 `release` environment

- 只有 release manager GitHub App/team 可以创建 `v*`；任何人都不能移动或删除已发布 tag。
- 使用 annotated signed tag，tag 必须指向已经通过 main required checks 的 commit。[GitHub signing tags](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-tags)
- 构建与发布拆 job：build job 无写权限；publish job 引用 `release` environment，required reviewer 不得是触发者，且 environment 禁止管理员绕过。
- workflow 顶层默认 `contents: read`；只在 attestation job 增加 `id-token: write`、`attestations: write`，只在最终 publish job 增加 `contents: write`。GitHub 建议 `GITHUB_TOKEN` 只给最小权限。[GitHub GITHUB_TOKEN permissions](https://docs.github.com/en/actions/tutorials/authenticate-with-github_token)
- release environment 只允许受保护的 `v*` tag；GitHub environment 可在 job 运行/读取 secrets 前要求审批、限制 branch/tag，并禁止自我审批。[GitHub deployments and environments](https://docs.github.com/en/actions/reference/workflows-and-actions/deployments-and-environments)

#### Action 与 dependency policy

- 所有 `uses:` 固定 **完整 40 位 commit SHA**，同行注释上游版本，如 `# v4.2.2`；Dependabot 的 `github-actions` ecosystem 继续负责升级 PR。
- 组织设置启用 “Require actions to be pinned to a full-length commit SHA”，并只允许 GitHub 官方、已评审的明确 allowlist 和仓库内 action。
- 删除 `stable`、`latest`、`3.x` 等动态输入；Rust 用 `rust-toolchain.toml` 精确 channel/components/profile，Node/pnpm/Task/NSIS/Wails 都采用机器可校验的精确版本。
- build 脚本不得自动 `tidy` 或修改 lock；依赖更新是单独 PR。release 使用 `pnpm install --frozen-lockfile`、`cargo build --locked`、`go mod download`，构建后必须 clean tree。

GitHub 明确说明完整 commit SHA 是 Action 唯一不可变引用方式，并可在仓库/组织策略中强制；tag 可被移动或删除。[GitHub secure use reference](https://docs.github.com/en/actions/reference/security/secure-use)。OpenSSF Scorecard 的 Pinned-Dependencies 与 Branch-Protection checks 也把这些风险列为供应链检查项。[OpenSSF Scorecard checks](https://github.com/ossf/scorecard/blob/main/docs/checks.md)

### 5. 一个可信桌面 release 应包含什么

建议把发布流水线设计成不可逆的五段，而不是 tag push 后直接上传裸 exe：

```text
protected source/tag
  -> hermetic-ish clean build
  -> stage exact install tree
  -> test staged bytes
  -> sign + timestamp
  -> SBOM/checksum/provenance + draft release
  -> independent approval
  -> publish immutable release
```

#### 完整产物

Windows stable release 至少发布：

- 签名并 timestamp 的 installer（主入口）；
- 签名的 portable archive（如果产品承诺 portable），内含 `Yotta.exe`、`capture_wgc.dll`、`platform-tools`、许可证、第三方 notices 和 machine-readable artifact manifest；
- `SHA256SUMS`；
- artifact-level SPDX 或 CycloneDX SBOM；
- GitHub/SLSA build provenance attestation；
- 人类可读 release notes，列出 breaking changes、权限变化、支持平台、已修 CVE/GHSA 和已知问题。

仓库 dependency graph 导出的 SPDX SBOM 是好起点，但它描述仓库依赖，不一定覆盖最终 bundle 中的 ADB、DLL、生成资源和安装器内容；release 应从 **staged artifact** 生成或补全 SBOM。GitHub 支持从 dependency graph 导出 SPDX，也支持为 build artifact 生成与 SBOM 关联的 attestations。[GitHub SBOM export](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/establish-provenance-and-integrity/export-dependencies-as-sbom)；[GitHub build attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations)

#### 三种不同的“签名”都需要分清

1. **Signed git tag**：证明发布标签由受信维护者创建。
2. **Windows Authenticode + RFC3161 timestamp**：证明 exe/DLL/installer 的 publisher 身份，并让签名在证书到期后仍可验证。
3. **Build provenance attestation**：证明某个 hash 的 artifact 由哪个 repository/commit/workflow/build platform 产生。

它们互不替代。Electron 官方发布指南明确建议准备分发的 Windows/macOS 应用做 code signing；未签名应用会触发操作系统安全阻拦或复杂的手工绕过。[Electron Code Signing](https://www.electronjs.org/docs/latest/tutorial/code-signing)

#### SLSA 目标

- **Yotta 3.0 GA：SLSA Build L2**。要求 provenance 被签名并由 hosted build platform 产生，重点防止构建后 artifact/provenance 被篡改。
- **成熟阶段：SLSA Build L3**。采用经过隔离的 reusable build workflow/hardened build platform，减少 build process 内部篡改风险。

SLSA v1.0 的 Build L1/L2/L3 分别是“存在 provenance”“hosted platform 生成的 signed provenance”“hardened build platform”；不要把有 checksum 或有 SBOM 误报成 SLSA L2。[SLSA security levels](https://slsa.dev/spec/v1.0/levels)

#### Immutable release

发布前先建 draft、上传全部 asset、完成验证后一次 publish，并启用 GitHub immutable releases。该功能锁住 tag 与 assets，自动生成 release attestation；用户可用 `gh release verify` 与 `gh release verify-asset` 核验。[GitHub immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases)；[Verifying release integrity](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/verify-release-integrity)

### 6. 可复现构建：provenance 之外的第二道证明

Provenance 说明“官方 builder 构建了这些字节”，reproducible build 允许独立方验证“这些源码确实会生成相同字节”。二者应同时存在。

Yotta 的可复现目标应分两层：

- **unsigned payload reproducible**：两个 clean runner 在不同 workspace path/time/username 下，用同一已记录工具链构建，exe、DLL、前端 dist 和未签名 archive 的 hash 一致。
- **signed distribution traceable**：签名本身含时间/证书服务输出，通常不适合作为跨构建 bit-for-bit 对象；只签一次已验证的 unsigned payload，并在 manifest 中同时记录 pre-sign 与 final hash，发布最终字节的 provenance。

落地要求：精确工具链、frozen locks、稳定文件排序与 archive timestamp、`SOURCE_DATE_EPOCH`、不嵌入 wall clock/absolute path/hostname、生成器版本固定、两次 build compare job、可供第三方运行的 `REPRODUCING.md`。Reproducible Builds 项目把目标定义为由给定源码重建 byte-for-byte 相同二进制，并要求定义/记录 build environment 与提供比较协议。[Reproducible Builds planning](https://reproducible-builds.org/docs/plans/)；[documentation index](https://reproducible-builds.org/docs/)

### 7. 漏洞响应不是一份 SECURITY.md

Yotta 应保留现有威胁模型和 7 天确认目标，同时增加可操作的响应系统：

- 至少两名 security responder 接收 GitHub private vulnerability reports；通知规则经过测试，避免报告只落到一个人邮箱。
- `SECURITY.md` 写清 supported versions、scope、确认/初判/状态更新时间目标、embargo 协调、公开披露和 credit 规则；不要承诺无法维持的修复时限。
- 使用 GitHub Security Advisory 的 temporary private fork 协作；修复后发布 GHSA，并为适用漏洞请求 CVE；release note 明确 affected/fixed ranges。
- 建安全 advisory 索引和 incident runbook：token/证书泄露、Action compromise、恶意依赖、错误发布、MCP/文件/进程越权分别如何撤销、重建和通知。
- Dependabot alerts + security updates、dependency review、CodeQL、secret scanning/push protection、govulncheck、cargo-deny 和 Gitleaks 都进入平台门禁；不能只运行一部分 CLI 后不强制结果。
- 每半年做一次桌面发布密钥与安全报告 tabletop：第二个人能否在创始人离线时接收报告、构建、签名、撤销和发公告。

GitHub 建议项目清楚公布私密报告入口，并指出 private vulnerability reporting 能让研究者直接私密创建 draft advisory；维护者应及时修复并在公开时让生态知道修复版本。[GitHub coordinated disclosure](https://docs.github.com/en/code-security/concepts/vulnerability-reporting-and-management/coordinated-disclosure)；[Configuring private vulnerability reporting](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configuring-private-vulnerability-reporting-for-a-repository)

依赖 review 可在 PR 引入已知漏洞时使 check 失败；CodeQL default setup 对公开仓库可用，并直接支持 Go 与 JavaScript/TypeScript。[GitHub dependency review](https://docs.github.com/en/code-security/concepts/supply-chain-security/dependency-review)；[GitHub CodeQL default setup](https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/configure-code-scanning/configure-code-scanning)

### 8. 支持窗口：允许破坏性升级，也要有明确合同

“3.0 不兼容 2.x”是一次版本策略，不等于可以没有支持策略。建议按当前维护能力采用保守合同：

- `3.x stable`：只支持最新 minor 的最新 patch；安全修复只回补当前 stable line。
- `preview/nightly`：不承诺数据兼容与回补，不能覆盖 stable 安装，UI 和下载页显著标识。
- `2.x`：在 3.0 GA 当天 EOL，不再修复；提前在 2.x 最后一个 release 和 3.0 release notes 中明确数据不可打开、备份方式和 EOL 日期。3.0 不携带 migration/fallback。
- 下一 major：至少在 roadmap/release notes 宣布 breaking epoch 与 EOL 日期；发现被主动利用的严重漏洞时，项目可缩短窗口并清楚公告。
- `SUPPORT.md` 同时列 host OS、target adapter、架构、安装形式和支持等级；不要把“CI 能编译”写成“产品支持”。

这比承诺多个长期维护分支更诚实。成熟后再按人力增加窗口。Electron 的官方政策是支持最新三个 stable majors、每条只支持最新 minor，并清楚写明 oldest line 仅接收 security fixes；值得借鉴的是**明确且可执行**，不是机械复制“三个版本”。[Electron release and support policy](https://www.electronjs.org/docs/latest/tutorial/electron-timelines)

### 9. Bus factor 与维护者连续性

当前只有一个 git 作者和一个 GitHub admin，bus factor 明确为 1。OpenSSF Silver 要求任意一人离开后，项目仍能在确认后 1 周内创建/关闭 issue、接受变更和发布版本，并建议 bus factor≥2；Gold 则把 bus factor≥2 和至少两个非关联重要贡献者设为硬要求。[OpenSSF Silver](https://www.bestpractices.dev/en/criteria/1)；[OpenSSF Gold](https://www.bestpractices.dev/en/criteria/2)

最低连续性设计：

- 仓库迁入 GitHub Organization；至少两名 owner，强制 secure 2FA（passkey/security key/app，不仅 SMS）。GitHub 可对组织成员、outside collaborators 和 billing managers 强制 2FA。[GitHub organization 2FA](https://docs.github.com/en/organizations/keeping-your-organization-secure/managing-two-factor-authentication-for-your-organization/requiring-two-factor-authentication-in-your-organization)
- 两人都能完成 issue moderation、PR merge、GHSA response 和 release；但日常 release 仍要求相互审批，不能共享个人账号或私钥。
- 域名、组织、签名服务、包 registry、更新服务器、社交账号与财务账号列入 access inventory；恢复码/紧急访问由组织控制并定期演练。
- signing key/service 使用独立 release role、审计日志和撤销 runbook；证书或 OIDC policy 失陷时第二人能立即停发。
- 每次 release 由 shadow release manager 跟做；关键流程必须只靠 `RELEASING.md` 与组织凭据完成。
- 大型项目目标不是“再加一个管理员”，而是培养至少 3 名活跃 maintainer、2 名 release manager、2 名 security responder，并逐步达到至少两个组织的维护者与实际非关联贡献者。

## Yotta 的最低可公开门槛

项目已经在 GitHub 公开，因此这里的“门槛”指：在 README、官网或 3.0 宣传中准确自称 open source，并发布新的 stable binary 之前必须全部满足。任一项未完成，应把下载标记为 experimental/source-available preview，不能发布 `v3.0.0` stable。

### Source open gate

- [ ] OSI 许可证已选，历史权利/第三方资产审计完成，LICENSE/NOTICE/SPDX/README 一致。
- [ ] 唯一项目身份已确定；公开仓库包含当前开发历史，默认分支不再落后本地主线。
- [ ] `CODE_OF_CONDUCT`、`GOVERNANCE`、`MAINTAINERS`、`CODEOWNERS`、`CONTRIBUTING`+DCO、`SECURITY`、`SUPPORT`、`ROADMAP`、`RELEASING` 可发现且相互链接。
- [ ] 至少两名独立管理员；组织强制 secure 2FA；角色和实际 GitHub 权限一致。
- [ ] main ruleset 强制 PR、非作者审查、required checks、conversation resolution、防强推/删除且无常态 bypass。
- [ ] private vulnerability reporting 已启用并实测；security responders 和通知备份存在。

### Stable binary gate

- [ ] `v*` tag ruleset、signed annotated tag、release environment 和非自我审批生效。
- [ ] release workflow 的 Actions/工具链/依赖均不可变固定；构建后工作树 clean。
- [ ] 发布的是经过 staged smoke 的完整 installer/archive，不是单独裸 exe；测试字节与发布字节相同。
- [ ] exe/DLL/installer Authenticode 签名并 timestamp；失败即阻断。
- [ ] 每个 release 有 SHA256SUMS、artifact SBOM、build provenance/attestation、人类 release notes、第三方 notices。
- [ ] GitHub release immutable；用户文档给出 `gh release verify`、attestation、Authenticode 和 checksum 验证方法。
- [ ] 支持渠道/EOL/漏洞回补边界已发布；3.0 对 2.x 的拒绝与备份提示清楚。
- [ ] 达成并公开 OpenSSF Best Practices Passing；供应链目标至少 SLSA Build L2。

## 成熟阶段

| 阶段 | 可对外定位 | 硬性退出条件 |
|---|---|---|
| L0 当前 | public source snapshot / source-available preview | 非 OSI license、公开主线滞后、单 admin、无保护分支、裸且不可验证 release；不得称“大型开源项目” |
| L1 可公开开源 | open source preview | 完成 Source open gate；当前开发公开；DCO/CoC/governance/required checks/PVR/双管理员生效；可以接受社区贡献，但 stable binary 仍受发布门槛限制 |
| L2 可信 3.0 | stable open-source desktop tool | 完成 Stable binary gate；OpenSSF Passing；SLSA Build L2；signed complete artifacts；明确 support/EOL；可由第二人独立发布和响应漏洞 |
| L3 大型项目 | mature community project | OpenSSF Silver；SLSA Build L3；unsigned payload 可复现；≥3 active maintainers、至少 2 org；公开 contributor ladder/RFC-roadmap/release train；依赖与安全门禁全自动；第三方安全评审开始周期化 |
| L4 关键生态 | security-mature critical project | OpenSSF Gold；至少两个非关联重要贡献者；敏感变更双人审查；90% statement/80% branch coverage（适用时）；最近 5 年安全评审；多组织治理、事故演练、密钥轮换和独立重建验证持续运行 |

阶段指标直接对应 OpenSSF 的 Passing/Silver/Gold 条件及 SLSA Build levels，而不是自行发明徽章。[OpenSSF Badge program](https://openssf.org/projects/best-practices-badge/)；[SLSA levels](https://slsa.dev/spec/v1.0/levels)

## 推荐实施顺序

### Wave 0：先停止制造新的不可信状态

1. 冻结新的 stable tag；先不要让当前 `push v*` 自动发布裸 exe。
2. 决定 OSI license 与唯一组织身份；对本地领先历史执行 secret、license、large binary 和 author audit。
3. 迁移/统一仓库后立即启用 main/tag rulesets、2FA、第二管理员和 private vulnerability reporting。

### Wave 1：公开协作合同

1. 落 DCO、CoC、governance、maintainer ladder、CODEOWNERS、support、roadmap、release/security roles。
2. 让仓库记录的每个 documented check 真正在 CI 运行并设 required；打开 CodeQL、dependency review、secret scanning/push protection、Dependabot security updates、OpenSSF Scorecard。
3. 所有 Action 与工具链精确固定；release build 禁止 `tidy`/非 frozen install。

### Wave 2：重建发布链

1. 先 staging 完整 Windows 产品树与 artifact manifest；对 staging 执行 Wails 启动、bindings、DLL load、ADB probe、uninstall/upgrade smoke。
2. 发布完整 installer/portable archive；Authenticode + timestamp；生成 checksums、artifact SBOM 和 third-party notices。
3. 生成 GitHub build attestation，进入 protected release environment 审批后发布 immutable release；将验证命令写入 release notes。
4. 做两次独立 clean build compare，逐项消除时间、路径、排序和 toolchain variance。

### Wave 3：从“有第二个人”升级为社区

1. 用真实贡献晋升 reviewer/maintainer，而不是预先授予名义角色；记录决策、ownership 和 offboarding。
2. 每个 release shadow rotation，每半年漏洞/证书事故演练。
3. 依次申请 OpenSSF Passing、Silver；SLSA L2 后升级 L3；达到多组织与可复现构建后再宣称大型成熟项目。

## 可验收的外部检查

发布 `v3.0.0` 前应能由一台无项目私钥的干净机器完成：

```powershell
# Release/tag/asset 不可变与匹配
gh release verify v3.0.0 -R yottaapp/yotta
gh release verify-asset v3.0.0 .\Yotta-3.0.0-windows-x64.zip -R yottaapp/yotta

# Build provenance
gh attestation verify .\Yotta-3.0.0-windows-x64.zip -R yottaapp/yotta

# Windows publisher signature与 timestamp
signtool verify /pa /all /v .\Yotta.exe
signtool verify /pa /all /v .\Yotta-Setup.exe

# Published checksum
Get-FileHash .\Yotta-3.0.0-windows-x64.zip -Algorithm SHA256
```

项目维护者还应能证明：

```text
main protection/ruleset: active, no bypass
v* tag ruleset: active, update/delete restricted
required checks: all current job names present
private vulnerability reporting: enabled and test report received
Action policy: allowlist + full-SHA required
release environment: non-self reviewer + protected tags only
CodeQL/secret scanning/push protection/Dependabot security updates: enabled
two clean build hashes: identical before signing
release build 结束后: git diff --exit-code
```

## 最终优先级

1. **许可证与公开主线**：先成为真正可审阅的 OSI open-source project；不解决这一点，其余全是包装。
2. **身份、branch/tag protection 与第二维护者**：把“一个账号即可重写全部历史和发布”变成受治理的双人系统。
3. **完整、签名、可追溯的 release**：发布测试过的完整安装树，同时提供 Authenticode、SBOM、checksum、provenance 和 immutable release。
4. **可执行的漏洞响应**：启用并验证 PVR，明确 responder、支持版本、GHSA/CVE 和应急撤销流程。
5. **固定并可复现的供应链**：full-SHA Actions、精确工具链、frozen install、SLSA L2→L3 与独立 clean rebuild。
