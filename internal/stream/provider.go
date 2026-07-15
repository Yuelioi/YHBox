// Package stream implements bounded byte streams exclusively as Resource
// Broker providers. It does not expose sessions or channels as workflow values.
package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/resource"
)

const (
	ProviderID = "yotta.stream"
	Kind       = "yotta/stream"

	OperationSend    = "stream/send"
	OperationReceive = "stream/receive"
	OperationFinish  = "stream/finish"
	OperationCancel  = "stream/cancel"

	maxCancelReasonBytes = 1024
)

var (
	ErrCanceled = errors.New("stream canceled")
	ErrFinished = errors.New("stream producer already finished")
)

type Config struct {
	Capacity      int `json:"capacity"`
	MaxChunkBytes int `json:"maxChunkBytes"`
}

type Limits struct {
	MaxCapacity   int
	MaxChunkBytes int
}

type Provider struct{ limits Limits }

func NewProvider(limits Limits) (*Provider, error) {
	if limits.MaxCapacity <= 0 || limits.MaxChunkBytes <= 0 {
		return nil, errors.New("stream provider limits must be positive")
	}
	return &Provider{limits: limits}, nil
}

func (p *Provider) Open(_ context.Context, request resource.OpenRequest) (any, error) {
	if request.Kind != Kind {
		return nil, fmt.Errorf("stream provider cannot open kind %q", request.Kind)
	}
	config, err := decodeConfig(request.Config)
	if err != nil {
		return nil, err
	}
	if config.Capacity <= 0 || config.Capacity > p.limits.MaxCapacity {
		return nil, fmt.Errorf("stream capacity must be within 1..%d", p.limits.MaxCapacity)
	}
	if config.MaxChunkBytes <= 0 || config.MaxChunkBytes > p.limits.MaxChunkBytes {
		return nil, fmt.Errorf("stream chunk budget must be within 1..%d", p.limits.MaxChunkBytes)
	}
	return newSession(config), nil
}

func (p *Provider) Invoke(ctx context.Context, object any, operation string, payload []byte) ([]byte, error) {
	session, ok := object.(*session)
	if !ok || session == nil {
		return nil, errors.New("invalid stream provider object")
	}
	switch operation {
	case OperationSend:
		if err := session.send(ctx, payload); err != nil {
			return nil, err
		}
		return nil, nil
	case OperationReceive:
		if len(payload) != 0 {
			return nil, errors.New("stream receive does not accept a payload")
		}
		return session.receive(ctx)
	case OperationFinish:
		if len(payload) != 0 {
			return nil, errors.New("stream finish does not accept a payload")
		}
		return nil, session.finish()
	case OperationCancel:
		if len(payload) > maxCancelReasonBytes || !utf8.Valid(payload) {
			return nil, errors.New("stream cancel reason must be bounded UTF-8")
		}
		return nil, session.cancel(string(payload))
	default:
		return nil, fmt.Errorf("unknown stream operation %q", operation)
	}
}

func (p *Provider) Close(_ context.Context, object any) error {
	session, ok := object.(*session)
	if !ok || session == nil {
		return errors.New("invalid stream provider object")
	}
	_ = session.cancel("resource lease closed")
	return nil
}

type terminalState uint8

const (
	stateOpen terminalState = iota
	stateFinished
	stateCanceled
)

type session struct {
	config Config
	mu     sync.Mutex
	queue  [][]byte
	state  terminalState
	reason string
	change chan struct{}
}

func newSession(config Config) *session {
	return &session{config: config, change: make(chan struct{})}
}

func (s *session) send(ctx context.Context, payload []byte) error {
	if len(payload) > s.config.MaxChunkBytes {
		return errors.New("stream chunk exceeds byte budget")
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.mu.Lock()
		switch s.state {
		case stateCanceled:
			err := s.cancelError()
			s.mu.Unlock()
			return err
		case stateFinished:
			s.mu.Unlock()
			return ErrFinished
		}
		if len(s.queue) < s.config.Capacity {
			s.queue = append(s.queue, append([]byte(nil), payload...))
			s.notifyLocked()
			s.mu.Unlock()
			return nil
		}
		change := s.change
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-change:
		}
	}
}

func (s *session) receive(ctx context.Context) ([]byte, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		s.mu.Lock()
		if s.state == stateCanceled {
			err := s.cancelError()
			s.mu.Unlock()
			return nil, err
		}
		if len(s.queue) > 0 {
			chunk := s.queue[0]
			copy(s.queue, s.queue[1:])
			s.queue[len(s.queue)-1] = nil
			s.queue = s.queue[:len(s.queue)-1]
			s.notifyLocked()
			s.mu.Unlock()
			return chunk, nil
		}
		if s.state == stateFinished {
			s.mu.Unlock()
			return nil, io.EOF
		}
		change := s.change
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-change:
		}
	}
}

func (s *session) finish() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.state {
	case stateCanceled:
		return s.cancelError()
	case stateFinished:
		return ErrFinished
	default:
		s.state = stateFinished
		s.notifyLocked()
		return nil
	}
}

func (s *session) cancel(reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == stateCanceled {
		return s.cancelError()
	}
	s.state = stateCanceled
	s.reason = reason
	for i := range s.queue {
		s.queue[i] = nil
	}
	s.queue = nil
	s.notifyLocked()
	return nil
}

func (s *session) cancelError() error {
	if s.reason == "" {
		return ErrCanceled
	}
	return fmt.Errorf("%w: %s", ErrCanceled, s.reason)
}

func (s *session) notifyLocked() {
	close(s.change)
	s.change = make(chan struct{})
}

func decodeConfig(raw []byte) (Config, error) {
	if len(raw) == 0 || len(raw) > 1024 {
		return Config{}, errors.New("stream config is required and bounded")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode stream config: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("stream config contains trailing JSON")
	}
	return config, nil
}
