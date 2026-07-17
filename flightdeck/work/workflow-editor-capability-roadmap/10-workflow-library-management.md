---
slice: "10"
title: 工作流库管理
status: completed
---

# Slice 10：工作流库删除、批量操作与分页

## Outcome / Question

工作流首页成为可扩展的工作流库，而不是只有 Run/Edit 的演示列表；用户可以安全删除、批量选择、搜索、排序和分页管理工作流。

## Completion criterion

- workflow service/store 提供明确的 delete contract；删除前检查 schedule、launcher item、正在运行/排队实例和其他持久引用。
- 产品明确引用处理：阻止删除并列出引用，或让用户显式选择级联；历史 Run journal 默认保留并显示 source 已删除，不篡改审计记录。
- 单项删除与批量删除都使用 Nuxt UI 确认，显示将受影响的对象；部分失败返回逐项结果，不能伪装成全成功。
- 列表查询支持 search、sort、page、pageSize 和 total；分页不是先把所有 source 拉到前端再切片。
- 页面支持全选当前页、跨页选择状态的明确语义、批量操作栏、空结果与加载/失败恢复。
- Run/Edit/Delete 的键盘与无障碍名称完整；成功删除原地更新列表，不发成功 toast。
- 删除与批量动作不绕过 source revision/CAS 和并发修改保护。

## Blocked by

Stage 3 批量验收；删除引用策略需要在实现前定案。

## Verification

先做 store/service 引用与分页定向测试、前端列表状态测试；Stage 4 完成后统一运行 task check、task build、Windows WebView 大数据列表与删除 smoke。

## Out of scope

云端协作、回收站/版本历史 UI、跨设备同步、无限滚动、删除历史 Run journal。

## Result

Completed。SourceStore 新增 revision+sourceHash CAS 删除，Application 阻止删除 queued/running Source，WorkflowService 提供后端 QuerySources(search/sort/page/pageSize/total)、PreviewDeleteSources 和逐项 DeleteSources。产品策略为引用阻止而非静默级联：Schedule、launcher item 与 active Run 都在确认前显示；批量只删除无引用项并原地报告 blocked/failed，历史 Run journal 不删。首页已加入搜索、排序、每页数量、翻页、当前页全选、明确跨页选择、单项/批量 Nuxt UI 确认、空结果与失败重试；Run/Edit/Delete 均有命名无障碍标签，成功不 toast。相关 4 个 Go 包、17 个前端测试、1546-key i18n、typecheck 通过；正式大数据 WebView 与删除 smoke 留到 Stage 4 门禁。
