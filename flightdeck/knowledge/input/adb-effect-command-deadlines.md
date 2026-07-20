# ADB effect command 必须有 Adapter-owned deadline

Workflow Run 的 context 可能没有 deadline。target resolver 自己的超时只覆盖设备发现和 identity/size 解析；如果后续 `adb shell input`、`exec-out screencap`、`monkey`、`am force-stop` 等 effect 直接继承 Run context，ADB server、transport 或设备卡住时，节点和整次 Run 都可能永久停在 RUNNING。

所有生产 ADB 调用必须经过同一个 Adapter-owned bounded runner：

- 每条子进程叠加有限 deadline（当前默认 10 秒），同时自然继承更短的调用方 deadline/cancellation。
- discovery、health、controller、capture、application operation 与 InputClip playback 都使用该 runner，不能只给 resolve 路径加超时。
- input/application 等真实副作用失败后直接返回 provider/journal；不得自动重试，因为首条命令可能已经在设备端生效，重试会造成重复点击、按键或启动/停止。
- emulator reconnect 是显式 discovery 行为，也必须受调用 context 和单命令上限约束。
- 回归测试使用会一直阻塞到 `ctx.Done()` 的 ADB runner，从无 deadline 的调用开始，断言 effect 在 Adapter 上限内返回 `context.DeadlineExceeded`。只用快速 fake 或真实热设备无法证明这一点。

真实纵向 smoke 仍必须从 Source → Compiler → Admission → installed provider → journal 运行；controller-only 绿灯不能证明 Run 生命周期会收敛。
