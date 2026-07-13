---
kind: checklist
summary: "新加一个前端可调的 Go 服务的五步全链路, 漏一步前端就调不到"
activation: action
read_when: "新增一个 wails 后端 service (前端要新 RPC) 前; 改服务注册 / bindings 生成链路时"
recheck_when: "服务注册方式 / bindings 生成命令 / backend.ts 门面约定变化时"
---
# 新增后端 Service 全链路 checklist
新加一个前端可调的 Go 服务, 五步, 漏一步前端就调不到:

1. **Go 服务**: `internal/services/<name>/service.go` — `NewService(...)` 构造, 导出方法即 RPC。
   - 落盘的: 路径用 `filepath.Join(dataDir, ...)`, 原子写 (tmp + rename), 加 `sync.Mutex`。极简单文件整存整取参 `codesnippet`; 按 ID 分目录的 store 模式参 `schedule`。
   - 方法签名只用 JSON 可编码类型 (struct/slice/map/基本类型) — 函数/接口参数 bindings 会 warn 且运行期炸。
   - 配一个 `service_test.go` (落盘服务最少: 空读 / 往返 / 清空)。
2. **main.go 注册**: import + 构造 (放数据层那一段, ~schedule 附近) + 加进 `wailsServices` 的 `application.NewService(xxxSvc)` 列表。三处都要, 漏注册 = 前端 invoke 报 method not found。
3. **生成 bindings**: `go build ./...` 过编译后跑 `wails3 generate bindings -clean=true` (或下次 `task dev`/`task build` 自动)。产物在 `frontend/bindings/github.com/yottaapp/yotta/internal/services/<name>/` (gitignore, 不提交)。
4. **backend.ts 门面**: import 生成的 service.js, 在 `backend` 对象加一组方法, 全走 `invoke()` (自动错误 toast; 不想 toast 的场景参 updateSilent 写裸调)。**组件/store 不许直接 import bindings** — wails3 API 漂移时只改 backend.ts 这一层。
5. **前端消费**: store/组件经 `backend.xxx.yyy()` 调。

**真机验证必须重启后端** (`task dev` 重起) — 新 RPC 不随前端热重载生效, 只刷页面会一直 method not found。
