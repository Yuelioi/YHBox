# Codex working agreement

SUMMARY: Codex 在本仓库的长期工作规范 — 源码优先、根因优先、验证优先、Git 边界
READ WHEN: 每次开始任务 / 写方案 / 修 bug / 准备提交或切分支前
RECHECK WHEN: 用户调整协作方式 / Flightdeck 流程变化 / Git 习惯变化 / 构建验证入口变化时

---

## 基本协作

- 对话、报告、提问、验收说明一律用中文；代码标识符、命令、commit message 按项目原有规范。
- 用户的明确指令高于 skill、插件、默认流程和本文件；冲突时按用户最新指令执行。
- 用户拍板后直接执行，只有业务歧义、破坏性操作或缺少必要输入时才停下来问。
- 面向用户的报告说人话：说明做了什么、怎么验证、剩下什么风险；不要堆内部术语。

## 工程纪律

- 有源码就先读源码。使用 API、函数、节点、数据结构前先 grep/阅读实现，确认名称、键、返回值和调用链，不凭文档或记忆猜。
- 复刻既有系统时先读入口、状态流转和 helper，方案里要能讲清“原系统在什么条件下做什么”。
- 修 bug 先定根因，再在唯一权威来源一次性修干净。不要用 fallback、shim、兼容分支掩盖没定位清楚的问题。
- 同一错误信号连续拒绝多个修复时，停止堆补丁，改用最小失败复现或 bisection 找第一个失败点。
- 构建、测试、类型检查只信工具退出码；IDE 诊断、缓存面板、截断/串扰的工具输出只能当线索。
- 注释只写不明显的 WHY；不要为显而易见的内部实现写旁白。导出 API 的文档注释按代码规范处理。

## Git 边界

- 当前约定：Codex 可以按逻辑边界自主创建本地 commit；没明确说“推送”就不 push。
- 新任务可以按需要切分支或建 worktree；不要沿用旧的“所有改动必须直提 main”规则。
- 不使用 `git reset --hard`、`git checkout --`、`git clean` 等会丢用户改动的命令，除非用户明确要求并确认范围。
- 遇到工作区已有改动，默认视为用户或其他 agent 的改动；先识别范围，只改自己任务需要的部分。

## Flightdeck 落点

- 常驻规则放 `flightdeck/knowledge/`。
- 任务状态、下一步、开放问题放 `flightdeck/cockpit.md` 或具体 `flightdeck/work/`。
- 错误复盘和踩坑放 `flightdeck/incidents/`；可复用流程放 `flightdeck/checklists/`；外部资料和三方源码放 `flightdeck/references/`。
- 不再写 auto-memory 文件；不要把易过期状态塞进常驻规范。

## 本项目约束

- 项目未发布，默认不为旧 schema、旧 API、旧 spec 写兼容层。删字段就同步改调用点和测试，清掉 deprecated 标记。
- 例外：仍服务用户文件的格式可以保留必要兼容，例如 cook 的 TOML；pure Go runtime 跨小版本兼容按实际价值判断。
- 构建、测试、smoke 命令以 `flightdeck/knowledge/build/build.md` 为准，不凭记忆敲命令。

## 本机环境

- 默认 shell 是 PowerShell；环境变量用 `$env:NAME`，空值是 `$null`，路径优先写成可跨工具识别的 forward slash。
- 本机工具输出可能串扰、截断或延迟刷新。遇到空结果、乱序结果或不完整内容，先单独重读关键文件一次，再下结论。
