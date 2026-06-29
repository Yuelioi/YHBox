# ⚠ StopApp 的 taskkill /IM 目标只能是 image name

SUMMARY: Windows `taskkill /IM` 不接受完整 exe 路径；StopApp 接到完整路径时必须先折成文件名，否则会报“无效查询/invalid query”。
READ WHEN: 改 StopApp / KillProcess / RunProgram→StopApp 流程；排查 StopApp 对完整 exe 路径失败；设计按进程名、PID 或路径结束程序的节点语义时。
RECHECK WHEN: StopApp 改成 path-specific 进程过滤、改用 PowerShell/WMI/Win32 API 枚举进程，或 taskkill 参数变化时。

---

`RunProgram.Target` 常是完整路径，例如 `E:\adobe\Adobe After Effects 2022\Support Files\AfterFX.exe`。如果用户把同一个值接到 StopApp，底层不能直接执行 `taskkill /IM <full-path>`，因为 `/IM` 只接受 image name 或通配 image name。

当前约定：

- 纯数字按 PID：`taskkill /F /PID <pid>`。
- 其他 target 如果含路径分隔符，先取 basename，再按 image name：`taskkill /F /IM AfterFX.exe`。
- 这不是按完整路径精确过滤；同名进程都会匹配。需要 path-specific 语义时，不能继续只靠 `/IM`。
