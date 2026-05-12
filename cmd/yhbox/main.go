// cmd/yhbox 异环工具箱主入口。双击 exe 启动 GUI。
package main

import (
	"fmt"
	"os"

	"yhbox/cmd/yhbox/gui"
	"yhbox/pkg/platform"
)

var version = "1.0.0"

func main() {
	platform.EnsureAdmin()

	if err := gui.Run(version); err != nil {
		fmt.Fprintf(os.Stderr, "GUI 启动失败: %v\n", err)
		os.Exit(1)
	}
}
