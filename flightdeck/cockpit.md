# Cockpit — YHFish

**Last updated**: 2026-06-10 by 月离 (modal + HUD 风格统一定稿。`common/BaseModal.vue` = **纯黑平铺**外壳(经多轮试错收敛: 包裹面板/凹陷井/emerald 边都试过, 用户选回原版纯黑), 11 modal 统一; 输入校准 + 录制 HUD 统一成**彩色状态面板**(border-{色}/40+bg-{色}/10)。试错 commit 已 squash 干净(8→`2b1ebef`+`3d96c3e`+`34faaef`)。门绿。)
**Active focus**: **modal + HUD 风格统一完成(真机已过)**。modal=纯黑平铺(BaseModal+11), 校准/录制 HUD=彩色状态面板。约定已落 checklist([ui.md](checklists/ui.md) BaseModal · [standalone-window-style.md](checklists/standalone-window-style.md) HUD 状态面板)。**整批待 push**。资产子系统已归档(→ [docs/asset-subsystem.md](docs/asset-subsystem.md))。

## 进行中

- [ ] **modal + HUD 风格统一(定稿, 待真机最终扫一眼)**。① **BaseModal**(`common/BaseModal.vue`)= **纯黑平铺**(bg-default + header border-b + body 平铺 px-5 py-4 + footer border-t; props open/title/icon/iconColor/size md..5xl/showClose)。多轮试错(包裹面板→emerald 边→凹陷井→回纯黑)后用户定**纯黑平铺**。11 modal 已统一(NewVar/PromoteToVar/ContainerSettings/DeleteVarConfirm/FindReferences/ValidationErrorPanel/ContainerHelp/NodeExplorer/LibraryExplorer + ConfirmDialog 保 API 重皮 + ImportToContainerDialog 多步)。② **校准 + 录制 HUD** 统一成**彩色状态面板**(border-{色}/40+bg-{色}/10, 按状态语义上色)。门绿。**剩(可选, 看用户)**: 其它 frameless HUD(截屏/鼠标检测 MouseHUDView)是否也上彩色面板、搜索面板(NodeSearch/CommandPalette)+ 大复合(TemplatePicker/TemplateManager)未转。

## 下一步

**push 整批**(真机已过)—— 资产子系统 + modal/HUD 风格统一这一大波(`0d744de`→本次 landing)还在本地没 push, 用户拍了就 `git push`。之后候选(看用户): 其它 frameless HUD(截屏 ScreenPicker / 鼠标检测 MouseHUDView)是否也上彩色状态面板 / 搜索面板(CommandPalette·NodeSearch)+ 大复合(TemplatePicker·TemplateManager)modal 是否收 / 子图切换 smoke(Hanging)/ idea 池(cv-perception · editor-footgun · misc-tools)。已知预存失败(非回归): runtime 缺 fish fixture, 见 [build.md](checklists/build.md)。

## Hanging tasks

- [ ] **子图系统问题（大部分已根治，剩真机 smoke 复验）**。修了三条真 bug，前两条同症状 "(子图未找到)" 不同根因：① 库导入绕过容器 Store 内存缓存 → [incident](incidents/2026-06-09-import-bypasses-container-store-cache.md)（library SetContainerReloader + 回归测试）；② keep-alive 多容器编辑器共享全局单例子图 store 切回污染 → [incident](incidents/2026-06-09-keepalive-singleton-subgraph-store-stale.md)。② 已**根治**（commit 20e25a9）：store 状态按容器隔离（subgraphsByContainer / editorPathByContainer keyed by containerID），activeContainerID 降级成前台指针，对外 API 不变、单测 59 绿；顺带消除未落盘子图编辑/层级切换丢失 + id 碰撞 mergeSubgraphs 取错版本。先前的 onActivated 补丁(9ccccbf)已被根治替换。**待用户真机 smoke 复验**：容器2 折叠子图 → 切容器3 → 切回容器2，子图节点应仍正常 + 分享成功。（原记的 2 个预存 vue-tsc 红已在 2026-06-10 import-strategy 收口随资产子系统清零。）
- [ ] 无阻塞待办。（原积压已路由：编辑器 footgun → [editor-footgun-backlog](specs/editor-footgun-backlog.md)；bindings/测试 fixture/AlwaysOnTop/通道B smoke → [checklists/build.md](checklists/build.md)；删符号全仓 grep → [checklists/code-style.md](checklists/code-style.md)；i18n residue → [misc-tools-backlog](specs/misc-tools-backlog.md)；诊断探针目录已删。）
