# plans/ — INDEX

<!-- AUTO:plans -->
- [2026-06-15-output-auto-capture-part1-backend.md](2026-06-15-output-auto-capture-part1-backend.md) — active — "Spec C 实现计划 Part 1 (后端自动捕获)。① 模板三件套 Found 补成 Bool Data 字段(两出口都 Set); ② config.capture{字段:变量名} 存储 + 路径① fire-time 自动捕获(ContainerRunner.applyCaptures 钩在 routeResult 成功路径 result.OutputData + 失败路径 {Error,Code}, 用 r.bundle.Vars.SetScoped auto); ③ 路径② ctx.CaptureOutput(field,value)(Ctx 接口 + ctxImpl.captureBindings, newCtx 从 config[capture] 解析)+ Loop/ForEach 改用它 + Index/Item 声明为 Body 出口 Data 字段; ④ 删 13 文件 27 个 Semantic:capture 输入 + 11 fire-time 节点 node.Capture 调用 + node.Capture 助手 + capture_test/spec_capture_test, grep+vet 零残留; ⑤ validator 校验 config.capture var-ref + BindableFields 后端单一来源。"
<!-- /AUTO -->
