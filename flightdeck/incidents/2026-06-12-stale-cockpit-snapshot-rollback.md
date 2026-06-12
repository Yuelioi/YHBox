---
status: active
when_to_read: preflight 读完 cockpit 后发现内容跟 git log 最近 land commit 对不上时; 工作区 cockpit.md 有未提交改动且不是本会话写的时; 怀疑"下一步"指向的活其实早干完了时
applies_to: [flightdeck, cockpit, preflight, stale-state, turn-end-hook, git-truth]
last_updated: 2026-06-12
---

# 工作区 cockpit 被旧快照盖回 → preflight 差点带用户重跑已完成的 P1+P2

**现象**: 2026-06-12 数据层大整理 P1+P2 已全部落地归档 (land commit `f7cc58b`), 但工作区 cockpit.md 被一份**开工前的旧快照**盖回 (Active focus 退回"说 go 即开 P1", 待验证条目丢失)。新会话 preflight 照读了这份脏文件并报"下一步: go 开 P1", 用户说 go 后差点对着已完成的活重新执行 — 全靠用户一句"这个应该干完了吧?"拦下。

**根因**: cockpit.md 的回合末钩子/上个会话残留写入, 把内存里的旧板面状态在 land commit **之后**写回了磁盘。preflight 只读工作区文件, 不交叉验证 git 历史, 对"工作区比 HEAD 旧"这种倒灌无感知。

**怎么做**:

- preflight 读 cockpit 时, 若 `git status` 显示 cockpit.md 有未提交改动 → **跟 HEAD 版 diff 一眼**: 工作区内容比 HEAD **旧**(退回更早状态) = 倒灌, 信 git, 用 HEAD 版内容恢复; 比 HEAD 新 = 正常的未落地增量。
- 判旧/新的锚: git log 最近的 `land(flightdeck)` commit 描述 vs cockpit `Last updated` 行 — land commit 说"P1+P2 落地归档"而 cockpit 还在"待开干", 必有一边是假的, **commit 不会说谎**。
- 用户报"这个不是干完了吗"类质疑 → 先查 git log 再答, 别拿板面当唯一真相。
