//go:build !windows

package backends

import "yotta/internal/services/inputclip"

// HybridBackend 非 Windows 平台 stub. 不真做任何 OS 调用.
type HybridBackend struct{}

// NewHybridBackend stub 构造. 签名跟 windows 一致 (hwndGetter 在 stub 忽略).
func NewHybridBackend(_ func() uintptr) IInputBackend { return &HybridBackend{} }

func (b *HybridBackend) Send(ev inputclip.Event) error          { return nil }
func (b *HybridBackend) SendBatch(evs []inputclip.Event) error  { return nil }
func (b *HybridBackend) ReleaseHeld() error                     { return nil }
func (b *HybridBackend) Name() string                           { return "hybrid-stub" }
