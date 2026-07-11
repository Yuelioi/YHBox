package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/container/runtime"
	"github.com/yottaapp/yotta/internal/services/execution"
	"github.com/yottaapp/yotta/pkg/winutil"
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
	Registry    node.RegistryReader
}

type Server struct {
	deps  Deps
	runMu sync.Mutex // 串行化 run_node, 防 AI 并行调用交错输入
}

func NewServer(deps Deps) *Server {
	if deps.Store != nil {
		// Store owns the contract generation used to validate and persist.
		deps.Registry = deps.Store.RegistrySnapshot()
	} else if deps.Registry != nil {
		deps.Registry = node.SnapshotRegistry(deps.Registry)
	} else {
		deps.Registry = node.DefaultRegistrySnapshot()
	}
	return &Server{deps: deps}
}

// Register 把全部工具挂到 MCPServer (authoring 复用迁入的纯函数; execution 新增).
func (s *Server) Register(m *server.MCPServer) {
	// --- authoring (只读/写图, 不受 arm 闸; save_container 受 arm 闸) ---
	m.AddTool(mcp.NewTool("list_nodes", mcp.WithDescription("List all Yotta node kinds with pins/types/required/defaults/category/capability flags. The building blocks; run_node executes one target/window action against a supplied window.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(string(listNodesJSONWithRegistry(s.deps.Registry))), nil
		})
	m.AddTool(mcp.NewTool("get_graph_schema", mcp.WithDescription("Yotta container-graph JSON schema + validated examples. Read before generating a container.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(string(graphSchemaJSON())), nil
		})
	m.AddTool(mcp.NewTool("validate_container", mcp.WithDescription("Validate a container graph JSON. Returns []ValidationError (empty=clean)."),
		mcp.WithString("container", mcp.Required(), mcp.Description("Container graph JSON."))),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw, err := req.RequireString("container")
			if err != nil || raw == "" {
				return mcp.NewToolResultError("missing 'container'"), nil
			}
			out, _ := validateContainerJSONWithRegistry(s.deps.Registry, []byte(raw))
			return mcp.NewToolResultText(string(out)), nil
		})
	m.AddTool(mcp.NewTool("save_container", mcp.WithDescription("Validate + persist a container graph (rejects on error-level issues). Server assigns id. Requires MCP armed."),
		mcp.WithString("container", mcp.Required(), mcp.Description("Container graph JSON."))),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if s.deps.Armed == nil || !s.deps.Armed() {
				return mcp.NewToolResultError("NOT_ARMED: 去设置页打开 arm 开关"), nil
			}
			raw, err := req.RequireString("container")
			if err != nil {
				return mcp.NewToolResultError("missing 'container'"), nil
			}
			res, saveErrs := saveContainerWithRegistry(s.deps.Store, s.deps.Registry, []byte(raw))
			if saveErrs != nil {
				return mcp.NewToolResultError(string(saveErrs)), nil
			}
			b, _ := json.MarshalIndent(res, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		})

	// --- window (只读, 不受 arm 闸) ---
	m.AddTool(mcp.NewTool("list_windows", mcp.WithDescription("List all top-level visible windows: {hwnd,title,class,processName,pid,clientW,clientH}. Pick a target, pass its hwnd to run_node.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			b, _ := json.MarshalIndent(winutil.EnumTopWindows(), "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		})
	m.AddTool(mcp.NewTool("find_window", mcp.WithDescription("Resolve the first top-level window matching title/class/processName. Returns its handle (hwnd + metadata)."),
		mcp.WithString("title", mcp.Description("Window title (exact unless titleMatch=regex).")),
		mcp.WithString("class", mcp.Description("Window class (exact).")),
		mcp.WithString("processName", mcp.Description("Process exe basename (case-insensitive).")),
		mcp.WithString("titleMatch", mcp.Description("'exact' (default) or 'regex'."))),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			spec := winutil.MatchSpec{
				Title:       req.GetString("title", ""),
				Class:       req.GetString("class", ""),
				ProcessName: req.GetString("processName", ""),
				TitleMatch:  req.GetString("titleMatch", "exact"),
			}
			wh, err := winutil.ResolveWindow(ctx, spec, 3*time.Second, 200*time.Millisecond)
			if err != nil {
				return mcp.NewToolResultError("WINDOW_NOT_FOUND: " + err.Error()), nil
			}
			b, _ := json.MarshalIndent(wh, "", "  ")
			return mcp.NewToolResultText(string(b)), nil
		})

	// --- run_node (受 arm + busy 闸) ---
	m.AddTool(mcp.NewTool("run_node", mcp.WithDescription("Execute ONE action node (kind from list_nodes, NeedsTarget or NeedsWindow) against a supplied Win32 window once. params = {pinName: literal}. Returns {ok,firedOutput,data,error}; image outputs returned as an image block. Requires MCP armed. Use this to probe (Capture to see, DetectColor for coords, ClickAt to test), then bake findings into save_container."),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Node kind, e.g. ClickAt / Capture / DetectColor.")),
		mcp.WithString("window", mcp.Required(), mcp.Description("Target window hwnd (decimal uintptr from find_window/list_windows).")),
		mcp.WithObject("params", mcp.Description("Input pin literals {pinName: value}."))),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			kind, err := req.RequireString("kind")
			if err != nil {
				return mcp.NewToolResultError("missing 'kind'"), nil
			}
			hwndStr, _ := req.RequireString("window")
			hwnd64, perr := strconv.ParseUint(hwndStr, 10, 64)
			if perr != nil {
				return mcp.NewToolResultError("invalid 'window' (expect decimal uintptr)"), nil
			}
			params := req.GetArguments()["params"]
			pm, _ := params.(map[string]any)
			res, img := s.runNode(ctx, kind, pm, uintptr(hwnd64))
			b, _ := json.MarshalIndent(res, "", "  ")
			if img != nil {
				mime := "image/png"
				if img.Format == "jpeg" {
					mime = "image/jpeg"
				}
				return mcp.NewToolResultImage(string(b), base64.StdEncoding.EncodeToString(img.Data), mime), nil
			}
			return mcp.NewToolResultText(string(b)), nil
		})
}
