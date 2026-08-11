package noderuntime

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workspacefs"
)

const imageFileChunkBytes = 64 << 10

func fileLoadImage(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		action := nodeadapter.AdapterAction{
			EffectID: nodes.FileLoadImageEffectID, Action: "filesystem.image-loaded", SummaryCode: "filesystem.load-image",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, workspacefs.CodeReadFailed, runErr))
		}()
		path, err := filePath(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		maximum, err := imageFileBudget(invocation.Config)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		pathDigest, err := artifact.Sum("yotta/workspace-file-path/v1", []byte(path))
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		action.Facts["path_digest"] = pathDigest.String()

		files := invocation.Sessions["workspace-files"]
		blobs := invocation.Sessions["blob-write"]
		if files == nil || blobs == nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, errors.New("image load capability session is missing"))
		}
		fileHandle, err := files.Open(ctx, []string{workspacefs.OperationReadRange, workspacefs.OperationStat}, []byte(`{}`))
		if err != nil {
			return nodeadapter.AdapterResult{}, mapFileFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, files.Drop(context.WithoutCancel(ctx), fileHandle)) }()
		statPayload, err := artifact.Marshal(workspacefs.StatRequest{Path: path})
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		rawMetadata, err := files.Invoke(ctx, fileHandle, workspacefs.OperationStat, statPayload)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapFileFailure(err)
		}
		metadata, err := workspacefs.OpenMetadata(rawMetadata)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapFileFailure(err)
		}
		if metadata.IsDirectory {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeIsDirectory, errors.New("image path is a directory"))
		}
		if metadata.Size <= 0 {
			return nodeadapter.AdapterResult{}, fileFailure(nodes.VisionImageInvalidCode, errors.New("image file is empty"))
		}
		if metadata.Size > int64(maximum) {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeBudgetExceeded, errors.New("image file exceeds its load budget"))
		}
		content := make([]byte, 0, int(metadata.Size))
		for offset := int64(0); offset < metadata.Size; {
			length := min(int64(imageFileChunkBytes), metadata.Size-offset)
			payload, err := artifact.Marshal(workspacefs.ReadRangeRequest{Path: path, Offset: offset, Length: length})
			if err != nil {
				return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
			}
			chunk, err := files.Invoke(ctx, fileHandle, workspacefs.OperationReadRange, payload)
			if err != nil {
				return nodeadapter.AdapterResult{}, mapFileFailure(err)
			}
			if int64(len(chunk)) != length {
				return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, errors.New("workspace file range length is invalid"))
			}
			content = append(content, chunk...)
			offset += length
		}
		if _, err := decodeVisionPNG(content); err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(nodes.VisionImageInvalidCode, errors.New("load image requires a valid bounded PNG"))
		}

		writeConfig, err := artifact.Marshal(blob.WriteConfig{MediaType: "image/png"})
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		writer, err := blobs.Open(ctx, []string{blob.OperationAppend, blob.OperationCancel, blob.OperationCommit}, writeConfig)
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		committed := false
		defer func() {
			if !committed {
				_, _ = blobs.Invoke(context.WithoutCancel(ctx), writer, blob.OperationCancel, nil)
			}
			runErr = errors.Join(runErr, blobs.Drop(context.WithoutCancel(ctx), writer))
		}()
		for offset := 0; offset < len(content); offset += imageFileChunkBytes {
			if _, err := blobs.Invoke(ctx, writer, blob.OperationAppend, content[offset:min(len(content), offset+imageFileChunkBytes)]); err != nil {
				return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
			}
		}
		rawRef, err := blobs.Invoke(ctx, writer, blob.OperationCommit, nil)
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		var ref blob.BlobRef
		if err := json.Unmarshal(rawRef, &ref); err != nil || ref.Validate() != nil || ref.MediaType != "image/png" {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, errors.Join(err, ref.Validate()))
		}
		imageType, ok := invocation.OutputTypes["image"]
		if !ok {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, errors.New("image output type is unresolved"))
		}
		imageValue, err := datatype.SealBlobRef(builtins.Catalog, imageType, ref)
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		metadataValue, err := sealFileOutputs(builtins, invocation, map[string]json.RawMessage{"metadata": rawMetadata})
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		committed = true
		action.Counters["bytes"] = ref.Size
		return nodeadapter.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"image": imageValue, "metadata": metadataValue["metadata"]}, ExecOutputs: []string{"completed"}}, nil
	}
}

func fileSaveImage(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		action := nodeadapter.AdapterAction{
			EffectID: nodes.FileSaveImageEffectID, Action: "filesystem.image-saved", SummaryCode: "filesystem.save-image",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, action, workspacefs.CodeWriteFailed, runErr))
		}()
		path, err := filePath(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		pathDigest, err := artifact.Sum("yotta/workspace-file-path/v1", []byte(path))
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		action.Facts["path_digest"] = pathDigest.String()
		ref, err := visionBlobInput(invocation, "image")
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(nodes.VisionImageInvalidCode, err)
		}
		content, err := readVisionBlob(ctx, invocation, ref)
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeReadFailed, err)
		}
		files := invocation.Sessions["workspace-files"]
		if files == nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, errors.New("image save capability session is missing"))
		}
		config, err := artifact.Marshal(workspacefs.WriteConfig{Path: path, Overwrite: configBool(invocation.Config, "overwrite")})
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		writer, err := files.Open(ctx, []string{workspacefs.OperationWriteAppend, workspacefs.OperationWriteCancel, workspacefs.OperationWriteCommit}, config)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapFileFailure(err)
		}
		committed := false
		defer func() {
			if !committed {
				_, _ = files.Invoke(context.WithoutCancel(ctx), writer, workspacefs.OperationWriteCancel, nil)
			}
			runErr = errors.Join(runErr, files.Drop(context.WithoutCancel(ctx), writer))
		}()
		for offset := 0; offset < len(content); offset += imageFileChunkBytes {
			if _, err := files.Invoke(ctx, writer, workspacefs.OperationWriteAppend, content[offset:min(len(content), offset+imageFileChunkBytes)]); err != nil {
				return nodeadapter.AdapterResult{}, mapFileFailure(err)
			}
		}
		rawMetadata, err := files.Invoke(ctx, writer, workspacefs.OperationWriteCommit, nil)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapFileFailure(err)
		}
		if _, err := workspacefs.OpenMetadata(rawMetadata); err != nil {
			return nodeadapter.AdapterResult{}, mapFileFailure(err)
		}
		sealed, err := sealFileOutputs(builtins, invocation, map[string]json.RawMessage{"metadata": rawMetadata})
		if err != nil {
			return nodeadapter.AdapterResult{}, fileFailure(workspacefs.CodeContractViolation, err)
		}
		committed = true
		action.Counters["bytes"] = ref.Size
		return nodeadapter.AdapterResult{Outputs: sealed, ExecOutputs: []string{"completed"}}, nil
	}
}

func imageFileBudget(config map[string]any) (int, error) {
	value, exists := config["maxBytes"]
	if !exists {
		return nodes.DefaultImageFileBytes, nil
	}
	maximum, err := configInt64(value)
	if err != nil || maximum < 1 || maximum > nodes.DefaultImageFileBytes {
		return 0, fileFailure(workspacefs.CodeContractViolation, errors.New("image file budget is invalid"))
	}
	return int(maximum), nil
}

func configBool(config map[string]any, key string) bool {
	value, _ := config[key].(bool)
	return value
}
