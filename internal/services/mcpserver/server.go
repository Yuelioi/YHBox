package mcpserver

import (
	"sync"

	"yotta/internal/services/container"
	"yotta/internal/services/container/runtime"
	"yotta/internal/services/execution"
)

// Deps 是 main.go 装配时注入的 GUI 常驻标准件 (与 runFunc 用的同一批).
type Deps struct {
	Store       *container.Store
	InputBus    *execution.InputBus
	Matcher     runtime.TemplateMatcher
	Game        runtime.GameProvider
	Clip        runtime.ClipResolver
	MouseCounts func() int  // 取 settings.ActiveMouseCounts360, live
	Armed       func() bool // 取 settings.MCP.Armed, live
	Busy        func() bool // worker.IsRunning
}

type Server struct {
	deps  Deps
	runMu sync.Mutex // 串行化 run_node, 防 AI 并行调用交错输入
}

func NewServer(deps Deps) *Server { return &Server{deps: deps} }
