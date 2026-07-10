package container

// 该文件仅 anonymous import 所有 node packages, 让 validator / data-graph / literal /
// pin-type 测试可以通过 nodepkg.Get(kind) 查到注册. 没这步, 测试里裸节点会被 validator
// 报 INVALID_PIN / 等. 业务代码本身从 main.go 触发同一个 all 包.
import _ "github.com/yottaapp/yotta/internal/nodes/all"
