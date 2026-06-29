package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"yotta/pkg/winutil"
)

// Runner Container 运行入口。main.go 注入：把 RunOnce 转成"enqueue 单 target run"。
// StopAll 取消当前 worker run + 清队列。
type Runner interface {
	RunOnce(id string) error
	StopAll() error
}

// ChangeListener 容器 CRUD 后回调，给 hotkey registry / schedule daemon 重新注册用。
type ChangeListener func()

// Service wails3 RPC 入口。
type Service struct {
	store    *Store
	runner   Runner
	onChange ChangeListener

	// hasTemplate / hasClip: 注入式查全局 asset 库某 GUID 是否存在 (validator 存在性检查用).
	// main.go 注入 (查 asset.Store). nil → 校验时跳过该 kind 存在性检查.
	hasTemplate func(guid string) bool
	hasClip     func(guid string) bool

	// postDelete 容器删除完成后回调 (main.go 注入: 匿名子图 GC — 锁序 Container → Subgraph).
	postDelete func()
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

// SetAssetExistence 注入资产存在性查询 (main.go 用全局 asset.Store).
// 校验时 CheckTemplate/ClickTemplate/WaitTemplate 引用的 GUID 不存在 → TEMPLATE_NOT_FOUND;
// PlayClip 引用的 GUID 不存在 → CLIP_NOT_FOUND.
func (s *Service) SetAssetExistence(hasTemplate, hasClip func(guid string) bool) {
	s.hasTemplate = hasTemplate
	s.hasClip = hasClip
}

// SetRunner 启动期 main.go 注入。Runner=nil 时 Run/Stop 返 error。
func (s *Service) SetRunner(r Runner) { s.runner = r }

// SetOnChange 启动期 main.go 注入。CRUD 后调一次（保存成功才调）。
func (s *Service) SetOnChange(f ChangeListener) { s.onChange = f }

// SetPostDelete 注入容器删除后的回调 (main.go: 匿名子图 GC).
func (s *Service) SetPostDelete(f func()) { s.postDelete = f }

func (s *Service) emitChange() {
	if s.onChange != nil {
		s.onChange()
	}
}

func (s *Service) List() []Container {
	return s.store.List()
}

// AINodesUsingConnection 返回引用某 AI 连接 ID 的 AI 节点(跨全部容器主图)。
// 前端删 AI 连接前调, 列出受影响节点让用户确认(不静默失效)。
func (s *Service) AINodesUsingConnection(connectionID string) []AIConnectionRef {
	return ScanAIConnectionRefs(s.store.List(), connectionID)
}

// Reload 从磁盘重读单个容器, 替换内存缓存, 返回最新内容。
// 给前端编辑器「重载」按钮用: MCP / 外部进程改盘后, 主 app 内存不知情, 靠这个重读。
// 不触发 emitChange —— 重载不是 CRUD, 不需重注册 hotkey/schedule。
func (s *Service) Reload(id string) (Container, error) {
	return s.store.Reload(id)
}

func (s *Service) Get(id string) (Container, error) {
	c, ok := s.store.Get(id)
	if !ok {
		return Container{}, fmt.Errorf("container %q not found", id)
	}
	return c, nil
}

// Create 新建一个空 Container：自带 Start → Stop 骨架，用户直接在中间插节点。
// 这种"前置骨架"对新手更友好（看到容器有明确起止），但 Stop 不是必须 — 删了也能跑。
func (s *Service) Create(name string) (Container, error) {
	startID := uuid.NewString()
	stopID := uuid.NewString()
	winTargetID := uuid.NewString()
	c := Container{
		SchemaVersion:  CurrentSchemaVersion,
		ID:             uuid.NewString(),
		Name:           name,
		InputBackend:   "postmessage",
		CaptureBackend: "auto",
		ScaleTolerance: DefaultScaleTolerance,
		Graph: Graph{
			ID:      uuid.NewString(),
			Version: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: startID, Kind: "Start", X: 100, Y: 120, Config: map[string]any{}, CreatedAt: time.Now().UTC()},
				{ID: winTargetID, Kind: "Win32WindowTarget", X: 100, Y: 260, Config: map[string]any{
					"Title": name,
				}, CreatedAt: time.Now().UTC()},
				{ID: stopID, Kind: "Stop", X: 500, Y: 260, Config: map[string]any{}, CreatedAt: time.Now().UTC()},
			},
			Edges: []GraphEdge{
				{From: startID + ".Done", To: winTargetID + ".In"},
				{From: winTargetID + ".Done", To: stopID + ".In"},
			},
		},
	}
	if err := s.store.Save(&c); err != nil {
		return Container{}, err
	}
	saved, _ := s.store.Get(c.ID)
	s.emitChange()
	return saved, nil
}

// Update patchJSON partial JSON merge 到现 Container。
func (s *Service) Update(id string, patchJSON string) error {
	c, ok := s.store.Get(id)
	if !ok {
		return fmt.Errorf("container %q not found", id)
	}
	if err := json.Unmarshal([]byte(patchJSON), &c); err != nil {
		return fmt.Errorf("parse patchJSON: %w", err)
	}
	c.ID = id // 防 patch 改 ID 触发 path traversal
	if err := s.store.Save(&c); err != nil {
		return err
	}
	s.emitChange()
	return nil
}

// ClearAllHotkeys 清空所有容器的热键绑定（容器 / 蓝图保留，仅去掉 hotkey）。
// 「快捷键」页「清空容器热键」用。返回清掉的数量。
// emitChange → containerHotkeyBinder.Refresh 把这些容器热键从注册中心反注册。
func (s *Service) ClearAllHotkeys() (int, error) {
	n := 0
	for _, c := range s.store.List() {
		if strings.TrimSpace(c.Hotkey) == "" {
			continue
		}
		c.Hotkey = ""
		if err := s.store.Save(&c); err != nil {
			return n, err
		}
		n++
	}
	if n > 0 {
		s.emitChange()
	}
	return n, nil
}

func (s *Service) Delete(id string) error {
	if err := s.store.Delete(id); err != nil {
		return err
	}
	if s.postDelete != nil {
		s.postDelete()
	}
	s.emitChange()
	return nil
}

// ValidateContainerByID 给前端用: 拿持久化版本跑校验, 返结构化 ValidationError 列表
// (不抛 error). 前端 "检查" 按钮 + 试运行前主动跑都走这条. 区分于 Validate() 包装函数:
// 那个返 *ValidationFailure 聚合所有 error 但 caller 需 errors.As 解包; 这个直接返列表 +
// warning, UI 不用解包.
func (s *Service) ValidateContainerByID(id string) []ValidationError {
	c, ok := s.store.Get(id)
	if !ok {
		return []ValidationError{
			{Severity: SeverityError, Code: "CONTAINER_NOT_FOUND", Params: map[string]any{"id": id}},
		}
	}
	sgs := s.store.subgraphsFor(&c)
	if s.hasTemplate != nil || s.hasClip != nil {
		return ValidateContainerWithDeps(&c, sgs, s.hasTemplate, s.hasClip)
	}
	return ValidateContainer(&c, sgs)
}

// Run 立即跑一次（前端 ▶ 按钮）。manual source 入 ExecutionQueue 单 target run。
func (s *Service) Run(id string) error {
	if _, ok := s.store.Get(id); !ok {
		return fmt.Errorf("container %q not found", id)
	}
	if s.runner == nil {
		return fmt.Errorf("runner not injected")
	}
	return s.runner.RunOnce(id)
}

// StopAll UI 全局停按钮（跟 Ctrl+Shift+F9 同语义）。
func (s *Service) StopAll() error {
	if s.runner == nil {
		return fmt.Errorf("runner not injected")
	}
	return s.runner.StopAll()
}

// (子图 RPC 已全局化 — 见 service_subgraph.go 的 SubgraphService, 不再按容器路由。)

// DeleteMany 批量删除容器。部分失败也继续；任一失败返 error。
func (s *Service) DeleteMany(ids []string) error {
	var errs []string
	for _, id := range ids {
		if err := s.store.Delete(id); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", id, err))
		}
	}
	// 匿名子图 GC: 批量末尾跑一次 (GC 幂等).
	if s.postDelete != nil {
		s.postDelete()
	}
	s.emitChange()
	if len(errs) > 0 {
		return fmt.Errorf("部分失败：%s", strings.Join(errs, "; "))
	}
	return nil
}

// SyncLocalMouseCalibration RPC 暴露给前端: 批量把所有本地容器主图的 MouseCalibration 节点
// counts360 改成 newCounts.
func (s *Service) SyncLocalMouseCalibration(newCounts int) (SyncMouseCalibrationResult, error) {
	res, err := SyncLocalMouseCalibration(s.store, newCounts)
	if err == nil || len(res.Updated) > 0 {
		s.emitChange()
	}
	return res, err
}

// ResolveWindow 按 containerID 的 Win32WindowTarget 节点解析目标窗口. 制作工具(截模板/取色/HUD)用.
// 容器不存在 / 无 Win32WindowTarget / 窗口没开 → error.
func (s *Service) ResolveWindow(containerID string) (winutil.WindowHandle, error) {
	c, ok := s.store.Get(containerID)
	if !ok {
		return winutil.WindowHandle{}, fmt.Errorf("container %q not found", containerID)
	}
	return ResolveWin32WindowTarget(context.Background(), &c, 3*time.Second, 500*time.Millisecond)
}

// ResolveWindowForNode 按节点最近上游 Win32WindowTarget 解析目标窗口.
// nodeID 为空 → 回落主 Win32WindowTarget（同 ResolveWindow）.
// 容器不存在 / 无 Win32WindowTarget / 窗口没开 → error.
func (s *Service) ResolveWindowForNode(containerID, nodeID string) (winutil.WindowHandle, error) {
	c, ok := s.store.Get(containerID)
	if !ok {
		return winutil.WindowHandle{}, fmt.Errorf("container %q not found", containerID)
	}
	wt := win32WindowTargetForNode(&c, nodeID)
	if wt == nil {
		return winutil.WindowHandle{}, ErrNoWin32WindowTarget
	}
	return winutil.ResolveWindow(context.Background(), ReadWin32WindowTargetMatchSpec(wt), 3*time.Second, 500*time.Millisecond)
}

// ResolveEditorTargetKindForNode resolves the user-visible target kind editor
// tooling should use for screenshots, point picking, rectangle picking, and
// color sampling. No explicit target keeps the historical Windows default.
func (s *Service) ResolveEditorTargetKindForNode(containerID, nodeID string) (string, error) {
	c, ok := s.store.Get(containerID)
	if !ok {
		return "", fmt.Errorf("container %q not found", containerID)
	}
	kind, _ := editorTargetKindForNode(&c, nodeID)
	return kind, nil
}

// CaptureBackendFor 返容器配置的截图后端名 (auto/gdi/wgc/mock). 容器不存在 → "auto".
func (s *Service) CaptureBackendFor(containerID string) string {
	c, ok := s.store.Get(containerID)
	if !ok {
		return "auto"
	}
	return ReadWin32WindowTargetCaptureBackend(&c)
}
