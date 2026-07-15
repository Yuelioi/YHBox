// Package nodes31runtime contains installed built-in adapters for Node Contract 3.1.
package nodes31runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/stream"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const conversionChunkBytes = 64 << 10

func Installed(builtins nodes31.Builtins) (map[string]compiler.InstalledAdapter, error) {
	concat, err := trustedEntry(builtins, nodes31.ConcatNodeID)
	if err != nil {
		return nil, err
	}
	toStream, err := trustedEntry(builtins, nodes31.BlobToStreamNodeID)
	if err != nil {
		return nil, err
	}
	toBlob, err := trustedEntry(builtins, nodes31.StreamToBlobNodeID)
	if err != nil {
		return nil, err
	}
	return map[string]compiler.InstalledAdapter{
		concat.Implementation.Entrypoint:   {Implementation: concat.Implementation, Run: concatAdapter(builtins)},
		toStream.Implementation.Entrypoint: {Implementation: toStream.Implementation, Run: blobToStream(builtins)},
		toBlob.Implementation.Entrypoint:   {Implementation: toBlob.Implementation, Run: streamToBlob(builtins)},
	}, nil
}

func trustedEntry(builtins nodes31.Builtins, nodeTypeID string) (nodecatalog.Entry, error) {
	entry, ok := builtins.Catalog.Lookup(nodeTypeID)
	if !ok {
		return nodecatalog.Entry{}, fmt.Errorf("built-in implementation lock for %q is missing", nodeTypeID)
	}
	trusted, err := nodes31.BuiltinImplementationLock(nodeTypeID)
	if err != nil {
		return nodecatalog.Entry{}, err
	}
	if entry.Implementation != trusted {
		return nodecatalog.Entry{}, fmt.Errorf("built-in implementation lock for %q does not match this build", nodeTypeID)
	}
	return entry, nil
}

func concatAdapter(builtins nodes31.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (map[string]datatype.ValueEnvelope, error) {
		a, aOK := invocation.Inputs["a"]
		b, bOK := invocation.Inputs["b"]
		if !aOK || !bOK || len(a.InlineJSON()) == 0 || len(b.InlineJSON()) == 0 {
			return nil, errors.New("concat requires inline inputs a and b")
		}
		outputs, err := nodes31.Concat(ctx, map[string]json.RawMessage{"a": a.InlineJSON(), "b": b.InlineJSON()})
		if err != nil {
			return nil, err
		}
		result, err := datatype.SealInlineJSON(builtins.Catalog, a.Type(), outputs["result"])
		if err != nil {
			return nil, err
		}
		return map[string]datatype.ValueEnvelope{"result": result}, nil
	}
}

func blobToStream(builtins nodes31.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (map[string]datatype.ValueEnvelope, error) {
		input, ok := invocation.Inputs["blob"]
		if !ok {
			return nil, errors.New("blob-to-stream input is missing")
		}
		ref, ok := input.BlobRef()
		if !ok {
			return nil, errors.New("blob-to-stream input is not a BlobRef")
		}
		blobSession, streamSession := invocation.Sessions["blob-read"], invocation.Sessions["stream"]
		if blobSession == nil || streamSession == nil {
			return nil, errors.New("blob-to-stream capability session is missing")
		}
		readConfig, err := artifact.Marshal(blob.ReadConfig{Blob: ref})
		if err != nil {
			return nil, err
		}
		reader, err := blobSession.Open(ctx, []string{blob.OperationReadRange}, readConfig)
		if err != nil {
			return nil, err
		}
		streamConfig, err := artifact.Marshal(stream.Config{Capacity: 4, MaxChunkBytes: conversionChunkBytes})
		if err != nil {
			_ = blobSession.Drop(context.Background(), reader)
			return nil, err
		}
		streamHandle, err := streamSession.Open(ctx, []string{
			stream.OperationCancel, stream.OperationFinish, stream.OperationReceive, stream.OperationSend,
		}, streamConfig)
		if err != nil {
			_ = blobSession.Drop(context.Background(), reader)
			return nil, err
		}
		if err := invocation.Spawn(func(taskCtx context.Context) (taskErr error) {
			defer func() { taskErr = errors.Join(taskErr, blobSession.Drop(context.Background(), reader)) }()
			defer func() {
				if taskErr != nil {
					_, _ = streamSession.Invoke(context.Background(), streamHandle, stream.OperationCancel, []byte("blob producer failed"))
				}
			}()
			for offset := int64(0); offset < ref.Size; {
				length := min(int64(conversionChunkBytes), ref.Size-offset)
				request, err := artifact.Marshal(blob.RangeRequest{Offset: offset, Length: length})
				if err != nil {
					return err
				}
				chunk, err := blobSession.Invoke(taskCtx, reader, blob.OperationReadRange, request)
				if err != nil {
					return err
				}
				if _, err := streamSession.Invoke(taskCtx, streamHandle, stream.OperationSend, chunk); err != nil {
					return err
				}
				offset += int64(len(chunk))
			}
			_, err := streamSession.Invoke(taskCtx, streamHandle, stream.OperationFinish, nil)
			return err
		}); err != nil {
			_ = blobSession.Drop(context.Background(), reader)
			_ = streamSession.Drop(context.Background(), streamHandle)
			return nil, err
		}
		envelope, err := datatype.SealStreamRef(builtins.Catalog, input.Type(), streamHandle)
		if err != nil {
			return nil, err
		}
		return map[string]datatype.ValueEnvelope{"stream": envelope}, nil
	}
}

func streamToBlob(builtins nodes31.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (map[string]datatype.ValueEnvelope, error) {
		input, ok := invocation.Inputs["stream"]
		if !ok {
			return nil, errors.New("stream-to-blob input is missing")
		}
		streamHandle, ok := input.StreamRef()
		if !ok {
			return nil, errors.New("stream-to-blob input is not a StreamRef")
		}
		mediaType, ok := invocation.Config["mediaType"].(string)
		if !ok {
			return nil, errors.New("stream-to-blob media type is missing")
		}
		streamSession, blobSession := invocation.Sessions["stream"], invocation.Sessions["blob-write"]
		if streamSession == nil || blobSession == nil {
			return nil, errors.New("stream-to-blob capability session is missing")
		}
		writeConfig, err := artifact.Marshal(blob.WriteConfig{MediaType: mediaType})
		if err != nil {
			return nil, err
		}
		writer, err := blobSession.Open(ctx, []string{blob.OperationAppend, blob.OperationCancel, blob.OperationCommit}, writeConfig)
		if err != nil {
			return nil, err
		}
		committed := false
		defer func() {
			if !committed {
				_, _ = blobSession.Invoke(context.Background(), writer, blob.OperationCancel, nil)
				_ = blobSession.Drop(context.Background(), writer)
			}
		}()
		for {
			chunk, err := streamSession.Invoke(ctx, streamHandle, stream.OperationReceive, nil)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, err
			}
			if len(chunk) == 0 {
				continue
			}
			if _, err := blobSession.Invoke(ctx, writer, blob.OperationAppend, chunk); err != nil {
				return nil, err
			}
		}
		rawRef, err := blobSession.Invoke(ctx, writer, blob.OperationCommit, nil)
		if err != nil {
			return nil, err
		}
		var ref blob.BlobRef
		if err := json.Unmarshal(rawRef, &ref); err != nil {
			return nil, fmt.Errorf("decode committed BlobRef: %w", err)
		}
		if err := ref.Validate(); err != nil {
			return nil, fmt.Errorf("validate committed BlobRef: %w", err)
		}
		committed = true
		envelope, err := datatype.SealBlobRef(builtins.Catalog, input.Type(), ref)
		if err != nil {
			return nil, err
		}
		return map[string]datatype.ValueEnvelope{"blob": envelope}, nil
	}
}
