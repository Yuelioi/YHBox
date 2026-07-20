# ⚠ Wails service/method 数量不是可审查的 RPC contract
统一流程是：

1. `node frontend/scripts/generate-bindings.mjs` 生成 bindings，并拒绝非零 warning；
2. `pnpm -C frontend bindings:check` 从 gitignored TypeScript bindings 提取 module、method、binding ID、参数、返回类型和 model/enum 字段；
3. 与 tracked `contracts/wails-rpc.json` 精确比较；
4. 有意修改 RPC 后先审查 contract diff，再运行 `pnpm -C frontend bindings:update`。

不要把 generator summary、`main.go` 的 slice capacity 或前端手写 DTO 当事实源。Wails binding ID 本身也不足够：它按方法身份生成，参数/返回 DTO 变化未必改变 ID，所以 manifest 必须同时记录签名和模型结构。
