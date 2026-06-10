package container

// 该文件仅 anonymous import 所有 node packages, 让 validator / data-graph / literal /
// pin-type 测试可以通过 nodepkg.Get(kind) 查到注册. 没这步, 测试里裸节点会被 validator
// 报 INVALID_PIN / 等. 业务代码本身从 main.go 引各节点包.
import (
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/collection" // Split/Join/List* 列表节点
	_ "yotta/internal/nodes/random" // RandomInt/RandomFloat/RandomBool
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
)
