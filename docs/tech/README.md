# YHBox 技术文档

面向**维护者 / 二次开发者**。用户向说明在项目根的 [README.md](../../README.md) 和 app 内"帮助"页。

## 文档索引

### 总体

- **[architecture.md](architecture.md)** — 项目脉络、跨 bot 共享机制、踩坑通用项
- **[development.md](development.md)** — 加新分辨率 / 加新语言 / 加新 bot / 调参流程

### 每个 bot 的技术细节

- **[fish.md](fish.md)** — 自动钓鱼，9 个 state，完整状态机 + 时序
- **[rhythm.md](rhythm.md)** — 自动音游，亮度检测算法 + ok-nte 来源
- **[cook.md](cook.md)** — 锤子连点（前台运行的原因）
- **[battle.md](battle.md)** — 全局热键切队伍
- **[piano.md](piano.md)** — MIDI 解析 + 时序按键

## 阅读顺序建议

**新人上手**：architecture.md → 想动哪个 bot 看哪个 → 想加新东西看 development.md。

**调参 / 调试**：development.md §调参。

**踩了坑**：architecture.md §8 "不要犯的错" + 对应 bot 的"已知坑"段。
