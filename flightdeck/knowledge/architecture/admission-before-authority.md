---
kind: note
summary: "Yotta 3.1 admission 必须先用可信 Host Profile 规划 exact target/credential binding，再由 Policy seal bounded Grant 并 durable create QUEUED；published-unconfirmed 不能重试或执行。"
activation: action
read_when: "新增 provider、Automation Target、credential binding、Policy/consent、Run queue、Worker recovery 或 production execution composition 时"
recheck_when: "Host Profile、Run Grant、Run Store create outcome、Worker notification 或 provider installation lock 改动后"
---
# Admission precedes runtime authority

唯一顺序是 strict-open Program → 按 target slot 对全部 attributed requirements 求 Host Profile 候选交集 → 显式消歧 target/credential → Policy 对 exact proposal 做 allow/deny/consent-required → seal short-lived Run Grant → durable create QUEUED RunRecord → notify Worker。

Program、Source、prompt、插件 manifest 和 adapter 返回值都不能构造 Host Profile 或扩大 proposal。Policy decision 不携带替换后的 operation/scope/binding；ConsentOnce/ConsentEveryRun 没有 durable consent lineage 时 Grant sealing 必须失败。Grant projection 只含 non-secret binding metadata，不含 token、cookie、key 或 credential material。

RunRecord 内嵌 canonical Grant artifact 用于重启恢复，但 artifact 自身不是 authority；Worker 必须以 strict-open Program Plan 与 Catalog 再次 OpenRunGrant，Run Owner 还要把 Grant 的 provider artifact digest/ABI 与实际安装实例逐项比较。

Run Store create 有三种结果：not-applied 可安全失败；durable 才可通知 Worker；published-unconfirmed 必须返回原 Grant/Record identity 与稳定错误，禁止生成第二个 Run、禁止执行，等待同一 identity 的持久性确认或安全终止。不得以 memory Grant、manual binding、legacy runtime 或同名 provider 替代这些边界。
