# ⚠ 移动/删除/归档文件后必须全仓 grep 更新引用者

SUMMARY: 移/删/归档文件只动文件本身会留断链, 必须全仓 grep 旧路径更新所有引用者
READ WHEN: 移动/删除/重命名文件或归档 spec·plan 前; 改完后验断链

---

**现象**: 2026-06-02 Yotta rebrand 删整个 `docs/tech/` + 把 5 个 done spec/plan 归档到 `landed/`, 两个动作各留断链, 直到 walkaround Audit 7 才兜住:

- `README.md` 仍 `[docs/tech/](docs/tech/)` → 目标目录已删
- `incidents/2026-06-02-wails-dev-fetch-transport-flattens-error.md` 仍 `[spec](../specs/2026-06-02-error-i18n-catalog.md)` → spec 已移 `landed/specs/`
- `specs/2026-06-01-node-widget-keycapture-consistency.md` 仍 `[已立 spec](2026-06-01-retire-global-game.md)` → 已移 `landed/specs/`

**根因**: 移/删文件只动了**文件本身**, 没动**指向它的引用者**。markdown 链接 / import / 文档交叉引用是单向的 —— 移动源文件不会自动修引用端, 引用端就成了死链。

**怎么做**:

- 删目录 / 删文件 / 重命名 / 归档(move 到 `landed/`) **之前或紧接着**: 全仓 grep 旧文件名/路径, 找所有引用者一起改。
- flightdeck 归档尤其易漏: spec 移 `landed/` 后, 引用它的 incident / 别的 spec / cockpit 链接都要从 `specs/x.md` 改成 `landed/specs/x.md`。
- 验证: 改完跑 `/flightdeck:walkaround` Audit 7 (dangling refs), 或 `grep -rl '<旧路径>'` 确认零残留。
- 同源精神见 `incidents/2026-05-29-storage-convention-consumer-audit-gap.md`(改存储约定前 exhaustive grep 全消费者)——本质都是「改/移一个东西前先找全依赖它的」。
