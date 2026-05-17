package inputclip

// Service wails3 RPC 入口. 瘦皮层转发到 Store + emit changes.
type Service struct {
	store *Store
	emit  func(name string, data any)
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetEmit(emit func(name string, data any)) { s.emit = emit }

// Store 暴露给 wire (ContainerRuntime 需要直接读 clip 数据用 store)
func (s *Service) Store() *Store { return s.store }

func (s *Service) emitChanged() {
	if s.emit != nil {
		s.emit("clip:changed", map[string]any{})
	}
}

// List 列所有 clip 摘要 (不带 events, 列表视图用).
func (s *Service) List() []ClipSummary {
	return s.store.List()
}

// Get 拿单个 clip 完整数据 (含 events).
func (s *Service) Get(id string) (*InputClip, error) {
	return s.store.Get(id)
}

// Save 写盘 (新建或覆盖).
func (s *Service) Save(clip *InputClip) error {
	if err := s.store.Save(clip); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

// Delete 删 clip.
func (s *Service) Delete(id string) error {
	if err := s.store.Delete(id); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}

// Resolve 满足 container/runtime.ClipResolver 接口. PlayClip 节点用.
func (s *Service) Resolve(id string) (*InputClip, bool) {
	clip, err := s.store.Get(id)
	if err != nil {
		return nil, false
	}
	return clip, true
}

// Update 改 metadata (label / description / tags) — 不改 events.
func (s *Service) Update(id string, label, description string, tags []string) error {
	clip, err := s.store.Get(id)
	if err != nil {
		return err
	}
	clip.Label = label
	clip.Description = description
	clip.Tags = tags
	if err := s.store.Save(clip); err != nil {
		return err
	}
	s.emitChanged()
	return nil
}
