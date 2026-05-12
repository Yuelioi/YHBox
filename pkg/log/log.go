// Package log 简单的多 sink 日志器。每条消息写入全部 sink。
// 早期 v1 不再做 ANSI 颜色（GUI 用 TextEdit 不渲染 ANSI；终端用户已不存在）。
package log

import (
	"fmt"
	"io"
	"time"
)

// Scope 是日志标签，留作向后兼容（颜色字段不再使用，纯字符串 tag）。
type Scope struct {
	Name  string
	Color string // 保留字段防破坏调用方，实际不用
}

var (
	SYSTEM = Scope{Name: "SYSTEM"}
	READY  = Scope{Name: "READY"}
	SETUP  = Scope{Name: "SETUP"}
	CLICK  = Scope{Name: "CLICK"}
	RESULT = Scope{Name: "RESULT"}
	MISS   = Scope{Name: "MISS"}
	STOP   = Scope{Name: "STOP"}
	PAUSE  = Scope{Name: "PAUSE"}
)

// Logger 把每条消息写入所有 sink。Sink 顺序无关，并发安全由调用方保证
// （walk.Synchronize 在 GUI sink 内部已处理）。
type Logger struct {
	sinks []io.Writer
}

// New 创建 Logger。传入零个或多个 sink。
func New(sinks ...io.Writer) *Logger {
	return &Logger{sinks: sinks}
}

// Log 用预定义 Scope 写一条带时间戳的日志。
func (l *Logger) Log(s Scope, format string, args ...any) {
	ts := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s\n", ts, s.Name, msg)
	for _, w := range l.sinks {
		w.Write([]byte(line))
	}
}

// Printf 兼容 logState 等动态 tag 调用。format 串里已带 [TAG]。
func (l *Logger) Printf(format string, args ...any) {
	ts := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s %s\n", ts, msg)
	for _, w := range l.sinks {
		w.Write([]byte(line))
	}
}
