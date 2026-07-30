# Plan

- [x] 1. 审计网络、应用、自动化三条 Source → Node Contract → Compiler → Admission → Runtime 路径，
  列出专属与共享 capability/grant 接口。
- [x] 2. 定义配置直驱的统一运行 seam，并以回归测试锁定：无授权事实、无 TTL、配置后直接运行。
- [x] 3. 迁移网络节点，删除 Origin/私网/重定向与 capability 门禁。
- [x] 4. 迁移应用节点，删除 application capability、身份授权与 arm 门禁。
- [x] 5. 迁移自动化节点，Target 仅解析配置并选择 adapter，删除 plan/grant/lease 依赖。
- [x] 6. 删除三类生产路径不再使用的安全代码、设置投影、错误码、文案和文档。
- [x] 7. 修复缓存编辑器的 Source revision 检测，完成增量门禁与自动化回归验收。
