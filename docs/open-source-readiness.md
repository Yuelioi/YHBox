# Public-release readiness

本页记录仓库本身能证明的发布事实，以及必须在 GitHub、签名基础设施或真实设备上另行确认的事项。当前
`LICENSE` 允许个人、教育和研究使用，但禁止未经书面许可的商业使用、营利分发、SaaS 和付费服务；因此
Yotta 当前是 **source-available**，不是 OSI open source。

## 仓库当前能证明什么

| 事实 | 代码或自动化 authority |
| --- | --- |
| 产品/模块 identity | 根 `VERSION` 当前为 `4.0.0`；Go module 是 `github.com/yottaapp/yotta` |
| Windows 候选构建 | `task package` 要求 clean worktree，运行全量门禁、正式构建、allowlisted staging 和 frozen-candidate smoke |
| 候选 payload | `scripts/stage-release.ps1` 只收集 Yotta GUI/CLI、ScriptWorker、WasmPluginRunner、WGC DLL、ADB runtime、许可/notice 和 artifact manifest |
| 供应链材料 | `.github/workflows/release.yml` 生成 SPDX/CycloneDX SBOM、SHA-256 checksums、provenance/SBOM attestation |
| 可复现性检查 | `.github/workflows/reproducible-build.yml` 在两个 clean checkout 构建并比较输出 hash |
| 持续门禁 | CI 包含 `task check:full`、Windows race、portable core、跨平台 production GUI compile、Gitleaks、govulncheck、license、cargo-deny、CodeQL、dependency review 和 Scorecard |

Release Candidate workflow 只有 `workflow_dispatch` 入口。它上传保留 14 天的 **unsigned candidate** artifact，
不会创建 GitHub Release、tag、更新 remote，也不会把候选发布给终端用户。仓库另有
`task release:sign-and-stage`，但证书选择、签名执行和最终分发仍是维护者控制的外部步骤。

## 仓库不能自行证明什么

本地文件无法证明 GitHub 仓库名/URL、默认分支、ruleset、owner 权限、private vulnerability reporting 是否
已启用，也无法证明代码签名证书归属、维护者值守或某次真机 smoke 的实际结果。这些项目必须在目标仓库和
真实发布环境逐项检查；不能从 README、旧 Work 记录或“CI 文件存在”推断已经完成。

## 首次公开发布检查

1. 用 `git remote get-url origin`、GitHub 页面和 package/module identity 共同确认 canonical Yotta 仓库；旧项目
   地址或描述仍存在时先修正，再生成 tag/链接。
2. 明确保留当前 source-available 许可，或在版权/贡献授权审查后显式替换许可证；发布材料不得使用与实际
   `LICENSE` 冲突的 “open source” 表述。
3. 在 GitHub 启用并实测 private vulnerability reporting，确认 [SECURITY.md](../SECURITY.md) 的入口可用。
4. 在 clean release commit 上运行 `task package`；需要正式签名时，在同一 frozen payload 上运行签名与 restage，
   不重编译后偷偷替换文件。
5. 按[平台支持](platform-support.md)和发布指南要求记录 Windows GUI、存储迁移、WebView、native adapter、
   Android ADB、Browser CDP 等实际需要的外部 smoke。未执行的项明确标为未验证。
6. 人工核对 ZIP、artifact manifest、SBOM、checksums 和 attestation 后，再创建 tag/GitHub Release 并上传；
   当前 workflow 不替代这一步。

可执行的候选流程见 [RELEASING.md](../RELEASING.md)。该文件同样是维护流程说明；实际任务和 payload 仍以
`Taskfile.yml`、release workflow 与 staging script 为准。
