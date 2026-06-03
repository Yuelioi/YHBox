package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	jsonpatch "github.com/evanphx/json-patch/v5"

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

// WindowOpener 用于开/聚焦容器编辑器独立子窗口。adapter 在 main.go / wire_container.go 里注入。
// 同 containerID 多次调用 → focus 已有窗口、不开第二个。
type WindowOpener interface {
	OpenEditor(containerID string) error
}

// Service wails3 RPC 入口。
type Service struct {
	store    *Store
	runner   Runner
	onChange ChangeListener
	windows  WindowOpener
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

// SetRunner 启动期 main.go 注入。Runner=nil 时 Run/Stop 返 error。
func (s *Service) SetRunner(r Runner) { s.runner = r }

// SetOnChange 启动期 main.go 注入。CRUD 后调一次（保存成功才调）。
func (s *Service) SetOnChange(f ChangeListener) { s.onChange = f }

// SetWindowOpener main.go 启动时注入。前端 OpenEditorWindow 调用前必须先注入；
// 没注入时 OpenEditorWindow 返清晰错误。
func (s *Service) SetWindowOpener(w WindowOpener) {
	s.windows = w
}

// OpenEditorWindow 给前端 RPC 用：打开 containerID 对应的容器编辑器独立窗口。
// 同 containerID 多次调用走 focus 现有窗口，不会开第二个。
func (s *Service) OpenEditorWindow(containerID string) error {
	if _, ok := s.store.Get(containerID); !ok {
		return fmt.Errorf("container %q not found", containerID)
	}
	if s.windows == nil {
		return fmt.Errorf("WindowOpener 未注入（main.go 启动期 SetWindowOpener 调用了吗？）")
	}
	return s.windows.OpenEditor(containerID)
}

// OpenInWindow 让前端从嵌入式编辑器一键开独立子窗口编辑同一容器.
// 内部走 WindowOpener (main.go 注入 containerWindowAdapter).
// 同 containerID 重复调 → windowAdapter focus 已有窗口, 不重复开.
func (s *Service) OpenInWindow(containerID string) error {
	if s.windows == nil {
		return fmt.Errorf("window opener 未注入")
	}
	return s.windows.OpenEditor(containerID)
}

func (s *Service) emitChange() {
	if s.onChange != nil {
		s.onChange()
	}
}

func (s *Service) List() []Container {
	return s.store.List()
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
		SchemaVersion: CurrentSchemaVersion,
		ID:            uuid.NewString(),
		Name:          name,
		Graph: Graph{
			ID:      uuid.NewString(),
			Version: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: startID, Kind: "Start", X: 100, Y: 120, Config: map[string]any{}, CreatedAt: time.Now().UTC()},
				{ID: stopID, Kind: "Stop", X: 500, Y: 120, Config: map[string]any{}, CreatedAt: time.Now().UTC()},
				// 新建容器自带 WindowTarget 占位 (匿名 title), 用户必须改成实际窗口标题
				// 才能 Save (validator 拒空 match). 这里给 placeholder 让首次创建不报错.
				{ID: winTargetID, Kind: "WindowTarget", X: 100, Y: 260, Config: map[string]any{
					"Title": name,
				}, CreatedAt: time.Now().UTC()},
			},
			Edges: []GraphEdge{
				{From: startID + ".Done", To: stopID + ".In"},
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
	return ValidateContainer(&c)
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

// === Subgraph RPC（暴露给前端） ===

// ListSubgraphs 列容器的全部子图。
func (s *Service) ListSubgraphs(containerID string) []Subgraph {
	return s.store.ListSubgraphs(containerID)
}

// GetSubgraph 拿单个子图。
func (s *Service) GetSubgraph(containerID, sgID string) (Subgraph, error) {
	sg, ok := s.store.GetSubgraph(containerID, sgID)
	if !ok {
		return Subgraph{}, fmt.Errorf("subgraph %q not found in container %q", sgID, containerID)
	}
	return sg, nil
}

// CreateSubgraph 新建 subgraph，自动分配 UUID id。
// 预填 SubgraphInput + SubgraphOutput 节点：v2 validator 要求每个 sg 必须有 ≥1 SubgraphOutput，
// 且 SubgraphOutput.config.declID 必须挂在 OutputPins 的某个 ID 上（runtime 才能 dispatch 父图）。
// 不预填 → 用户每建一个子图都会撞 EMPTY_SUBGRAPH_OUTPUT。
func (s *Service) CreateSubgraph(containerID, label string) (Subgraph, error) {
	declID := uuid.NewString()
	now := time.Now().UTC()
	sg := Subgraph{
		ID:    "sg-" + uuid.NewString()[:8],
		Label: label,
		Graph: Graph{
			ID:      uuid.NewString(),
			Version: GraphSchemaVersion,
			Nodes: []GraphNode{
				{ID: "in", Kind: "SubgraphInput", X: 80, Y: 160, CreatedAt: now},
				{ID: "out", Kind: "SubgraphOutput", X: 420, Y: 160,
					Config: map[string]any{"DeclID": declID}, CreatedAt: now},
			},
		},
		OutputPins: []SubgraphOutputDecl{{ID: declID, Name: "done"}},
		CreatedAt:  now,
	}
	if err := s.store.SaveSubgraph(containerID, &sg); err != nil {
		return Subgraph{}, err
	}
	s.emitChange()
	return sg, nil
}

// UpdateSubgraph 接 RFC7386 merge patch。
func (s *Service) UpdateSubgraph(containerID, sgID, patchJSON string) error {
	cur, ok := s.store.GetSubgraph(containerID, sgID)
	if !ok {
		return fmt.Errorf("subgraph %q not found", sgID)
	}
	curJSON, err := json.Marshal(cur)
	if err != nil {
		return err
	}
	merged, err := jsonpatch.MergePatch(curJSON, []byte(patchJSON))
	if err != nil {
		return fmt.Errorf("merge patch: %w", err)
	}
	var next Subgraph
	if err := json.Unmarshal(merged, &next); err != nil {
		return err
	}
	next.ID = sgID // 防 patch 改 id
	if err := s.store.SaveSubgraph(containerID, &next); err != nil {
		return err
	}
	s.emitChange()
	return nil
}

// DeleteSubgraph 删除子图文件 + 内存。
func (s *Service) DeleteSubgraph(containerID, sgID string) error {
	if err := s.store.DeleteSubgraph(containerID, sgID); err != nil {
		return err
	}
	s.emitChange()
	return nil
}

// DeleteMany 批量删除容器。部分失败也继续；任一失败返 error。
func (s *Service) DeleteMany(ids []string) error {
	var errs []string
	for _, id := range ids {
		if err := s.store.Delete(id); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", id, err))
		}
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

// ResolveWindow 按 containerID 的 WindowTarget 节点解析目标窗口. 制作工具(截模板/取色/HUD)用.
// 容器不存在 / 无 WindowTarget / 窗口没开 → error.
func (s *Service) ResolveWindow(containerID string) (winutil.WindowHandle, error) {
	c, ok := s.store.Get(containerID)
	if !ok {
		return winutil.WindowHandle{}, fmt.Errorf("container %q not found", containerID)
	}
	return ResolveWindowTarget(context.Background(), &c, 3*time.Second, 500*time.Millisecond)
}

// CaptureBackendFor 返容器配置的截图后端名 (auto/gdi/wgc/mock). 容器不存在 → "auto".
func (s *Service) CaptureBackendFor(containerID string) string {
	c, ok := s.store.Get(containerID)
	if !ok {
		return "auto"
	}
	return ReadWindowTargetCaptureBackend(&c)
}

// ScaleToleranceFor 读容器 WindowTarget 节点的模板缩放容差. 镜像 CaptureBackendFor.
// templateMatcherAdapter 按此决定 miss 精确分辨率时缩放兜底的允许范围.
func (s *Service) ScaleToleranceFor(containerID string) float64 {
	c, ok := s.store.Get(containerID)
	if !ok {
		return DefaultScaleTolerance
	}
	return ReadWindowTargetScaleTolerance(&c)
}
