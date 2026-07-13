---
kind: trap
summary: "Yotta 的 CI 曾硬编码 `14 Services, 107 Methods`，但 Recording 增加 5 个 RPC 后真实生成面已是 14/112；数量断言只会报“变了”，也抓不到同数量的重命名、签名或 DTO 字段变化。"
activation: symptom
read_when: "修改 Wails bound service、Go RPC request/response、frontend bindings 生成、CI gui-build 或准备更新 RPC 基线时"
recheck_when: "Wails generator 输出格式变化、`contracts/wails-rpc.json` schema 升级或 v3 presentation contract 取代当前 bindings 后"
---
# ⚠ Wails service/method 数量不是可审查的 RPC contract
统一流程是：

1. `node frontend/scripts/generate-bindings.mjs` 生成 bindings，并拒绝非零 warning；
2. `pnpm -C frontend bindings:check` 从 gitignored TypeScript bindings 提取 module、method、binding ID、参数、返回类型和 model/enum 字段；
3. 与 tracked `contracts/wails-rpc.json` 精确比较；
4. 有意修改 RPC 后先审查 contract diff，再运行 `pnpm -C frontend bindings:update`。

不要把 generator summary、`main.go` 的 slice capacity 或前端手写 DTO 当事实源。Wails binding ID 本身也不足够：它按方法身份生成，参数/返回 DTO 变化未必改变 ID，所以 manifest 必须同时记录签名和模型结构。
