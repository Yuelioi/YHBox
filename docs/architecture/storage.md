# Storage consistency

Settings 使用 immutable snapshot：更新流程是 clone、mutate/merge、validate、atomic swap、atomic save。同目录临时文件写入并 sync，rename 后同步父目录；失败保留旧文件。

Container 是多文件 package。提交顺序固定为 `package.json`、`graph.json`、`installation.json`，最后写 `yotta-lock.json` 作为 commit record。Lock 包含 portable 文件、installation 与 dependency closure 的 hash。加载时验证 lock；新旧混合 generation 被标记 incompatible，不作为正常容器运行。

Store 的内存 cache 只在磁盘 generation 完整提交后更新。导出在读锁内复制同一 generation，避免保存并发造成跨代 zip。

