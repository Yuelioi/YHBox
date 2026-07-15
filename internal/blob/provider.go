package blob

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/resource"
)

const (
	ProviderID = "blob"
	KindReader = "blob/reader"
	KindWriter = "blob/writer"

	OperationReadRange = "read-range"
	OperationAppend    = "append"
	OperationCommit    = "commit"
	OperationCancel    = "cancel"
)

type ProviderLimits struct {
	MaxChunkBytes int
	QueueCapacity int
}

type Provider struct {
	store  *Store
	limits ProviderLimits
}

type ReadConfig struct {
	Blob BlobRef `json:"blob"`
}

type WriteConfig struct {
	MediaType string `json:"mediaType"`
}

type RangeRequest struct {
	Offset int64 `json:"offset"`
	Length int64 `json:"length"`
}

type readerState struct {
	store *Store
	file  *os.File
	ref   BlobRef
}

type writeResult struct {
	ref BlobRef
	pin *blobPin
	err error
}

type writerState struct {
	mu          sync.Mutex
	chunks      chan []byte
	done        chan struct{}
	lifetime    context.Context
	cancel      context.CancelFunc
	inputClosed bool
	result      writeResult
}

type chunkReader struct {
	ctx     context.Context
	chunks  <-chan []byte
	current []byte
}

func NewProvider(store *Store, limits ProviderLimits) (*Provider, error) {
	if store == nil || limits.MaxChunkBytes <= 0 || limits.QueueCapacity <= 0 {
		return nil, errors.New("blob provider requires a store and positive chunk limits")
	}
	return &Provider{store: store, limits: limits}, nil
}

func (p *Provider) Open(ctx context.Context, request resource.ProviderOpenRequest) (any, error) {
	switch request.Kind {
	case KindReader:
		var config ReadConfig
		if err := decodeExact(request.Config, &config); err != nil || config.Blob.Validate() != nil {
			return nil, errors.New("invalid blob reader config")
		}
		state, err := p.openReader(ctx, config.Blob)
		if err != nil {
			return nil, err
		}
		return state, nil
	case KindWriter:
		var config WriteConfig
		if err := decodeExact(request.Config, &config); err != nil {
			return nil, errors.New("invalid blob writer config")
		}
		probe := BlobRef{MediaType: config.MediaType, Digest: zeroDigest(), Size: 0}
		if err := probe.Validate(); err != nil {
			return nil, err
		}
		lifetime, cancel := context.WithCancel(context.Background())
		state := &writerState{
			chunks: make(chan []byte, p.limits.QueueCapacity), done: make(chan struct{}), lifetime: lifetime, cancel: cancel,
		}
		go func() {
			ref, pin, err := p.store.putPinned(lifetime, config.MediaType, &chunkReader{ctx: lifetime, chunks: state.chunks})
			state.result = writeResult{ref: ref, pin: pin, err: err}
			close(state.done)
		}()
		return state, nil
	default:
		return nil, fmt.Errorf("blob provider cannot open kind %q", request.Kind)
	}
}

func (p *Provider) Invoke(ctx context.Context, object any, operation string, payload []byte) ([]byte, error) {
	switch state := object.(type) {
	case *readerState:
		if operation != OperationReadRange {
			return nil, errors.New("blob reader operation is not supported")
		}
		var request RangeRequest
		if err := decodeExact(payload, &request); err != nil || request.Length > int64(p.limits.MaxChunkBytes) {
			return nil, errors.New("invalid blob range request")
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if request.Offset < 0 || request.Length < 0 || request.Offset > state.ref.Size || request.Length > state.ref.Size-request.Offset {
			return nil, errors.New("blob range is outside the object")
		}
		result := make([]byte, int(request.Length))
		if _, err := state.file.ReadAt(result, request.Offset); err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		return result, nil
	case *writerState:
		switch operation {
		case OperationAppend:
			if len(payload) == 0 || len(payload) > p.limits.MaxChunkBytes {
				return nil, errors.New("blob append exceeds chunk limits")
			}
			return nil, state.append(ctx, payload)
		case OperationCommit:
			if len(payload) != 0 {
				return nil, errors.New("blob commit payload must be empty")
			}
			ref, err := state.commit(ctx)
			if err != nil {
				return nil, err
			}
			return artifact.Marshal(ref)
		case OperationCancel:
			state.cancel()
			return nil, nil
		default:
			return nil, errors.New("blob writer operation is not supported")
		}
	default:
		return nil, errors.New("invalid blob provider object")
	}
}

func (p *Provider) Close(ctx context.Context, object any) error {
	switch state := object.(type) {
	case *readerState:
		err := state.file.Close()
		state.store.mu.RUnlock()
		return err
	case *writerState:
		state.cancel()
		select {
		case <-state.done:
			state.result.pin.release()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	default:
		return errors.New("invalid blob provider object")
	}
}

func (p *Provider) openReader(ctx context.Context, ref BlobRef) (*readerState, error) {
	p.store.mu.RLock()
	file, err := os.Open(filepath.Join(p.store.root, objectName(ref.Digest)))
	if err != nil {
		p.store.mu.RUnlock()
		return nil, fmt.Errorf("open blob: %w", err)
	}
	if err := verifyOpenFile(ctx, file, ref); err != nil {
		_ = file.Close()
		p.store.mu.RUnlock()
		return nil, err
	}
	return &readerState{store: p.store, file: file, ref: ref}, nil
}

func (s *writerState) append(ctx context.Context, payload []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inputClosed {
		return errors.New("blob writer input is closed")
	}
	chunk := append([]byte(nil), payload...)
	select {
	case s.chunks <- chunk:
		return nil
	case <-s.done:
		return s.resultError()
	case <-s.lifetime.Done():
		return s.lifetime.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *writerState) commit(ctx context.Context) (BlobRef, error) {
	s.mu.Lock()
	if !s.inputClosed {
		s.inputClosed = true
		close(s.chunks)
	}
	s.mu.Unlock()
	select {
	case <-s.done:
		return s.result.ref, s.result.err
	case <-ctx.Done():
		return BlobRef{}, ctx.Err()
	}
}

func (s *writerState) resultError() error {
	if s.result.err != nil {
		return s.result.err
	}
	return errors.New("blob writer is complete")
}

func (r *chunkReader) Read(destination []byte) (int, error) {
	for len(r.current) == 0 {
		select {
		case <-r.ctx.Done():
			return 0, r.ctx.Err()
		case chunk, ok := <-r.chunks:
			if !ok {
				return 0, io.EOF
			}
			r.current = chunk
		}
	}
	written := copy(destination, r.current)
	r.current = r.current[written:]
	return written, nil
}

func decodeExact(raw []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("JSON contains trailing values")
	}
	return nil
}

func zeroDigest() artifact.Digest {
	return artifact.Digest("sha256:" + strings.Repeat("0", 64))
}
