//go:build !windows

package backends

import "yhbox/internal/services/inputclip"

// SendInputBackend 非 Windows 平台 stub. 不真做任何 OS 调用.
type SendInputBackend struct{}

// NewSendInputBackend stub 构造. 签名跟 windows 一致 (hwndGetter 在 stub 忽略).
func NewSendInputBackend(_ func() uintptr) IInputBackend { return &SendInputBackend{} }

func (b *SendInputBackend) Send(ev inputclip.Event) error          { return nil }
func (b *SendInputBackend) SendBatch(evs []inputclip.Event) error  { return nil }
func (b *SendInputBackend) ReleaseHeld() error                     { return nil }
func (b *SendInputBackend) Name() string                           { return "sendinput-stub" }
