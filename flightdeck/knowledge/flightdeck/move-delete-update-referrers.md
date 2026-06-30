# ⚠ 移动/删除/归档文件后必须全仓 grep 更新引用者

SUMMARY: 移/删/归档文件只动文件本身会留断链, 必须全仓 grep 旧路径更新所有引用者
READ WHEN: 移动/删除/重命名文件或归档 spec·plan 前; 改完后验断链

---

**现象**: 2026-06-02 Yotta rebrand 删整个旧技术文档目录 + 把 5 个 done spec/plan 归档, 两个动作各留断链, 直到 walkaround Audit 7 才兜住:

- `README.md` 仍指向已删除的技术文档目录 → 目标目录已删
- 某 incident 仍以 markdown 链接指向旧 specs 路径 → spec 已移入 cold archive package
- 某 spec 仍以 markdown 链接指向 `2026-06-01-retire-global-game.md` → 目标已移入 cold archive package

**根因**: 移/删文件只动了**文件本身**, 没动**指向它的引用者**。markdown 链接 / import / 文档交叉引用是单向的 —— 移动源文件不会自动修引用端, 引用端就成了死链。

**怎么做**:

- 删目录 / 删文件 / 重命名 / 归档 **之前或紧接着**: 全仓 grep 旧文件名/路径, 找所有引用者一起改。
- flightdeck 归档尤其易漏: active knowledge / cockpit / work index 不应依赖历史 spec/plan 路径;关键内容要摘进 knowledge 本体,历史只写成 cold archive package 注记。
- 验证: 改完跑 `/flightdeck:walkaround` Audit 7 (dangling refs), 或 `grep -rl '<旧路径>'` 确认零残留。
- 同源精神见 [storage-convention-consumer-audit-gap.md](../nodes/storage-convention-consumer-audit-gap.md)——本质都是「改/移一个东西前先找全依赖它的」。
