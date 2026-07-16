package pluginprotocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	MaxWasmModuleBytes = 64 << 20
	MinWasmMemoryPages = 1
	MaxWasmMemoryPages = 4_096
)

var wasmBootstrapMagic = [4]byte{'Y', 'W', 'M', '1'}

type WasmBootstrap struct {
	MemoryLimitPages uint32
	Module           []byte
}

// WriteWasmBootstrap transfers the exact module bytes before framed protocol
// traffic begins. The digest detects pipe corruption and confused framing.
func WriteWasmBootstrap(writer io.Writer, bootstrap WasmBootstrap) error {
	if writer == nil {
		return errors.New("Wasm bootstrap writer is required")
	}
	if err := validateWasmBootstrap(bootstrap); err != nil {
		return err
	}
	digest := sha256.Sum256(bootstrap.Module)
	var header [48]byte
	copy(header[:4], wasmBootstrapMagic[:])
	binary.BigEndian.PutUint32(header[4:8], bootstrap.MemoryLimitPages)
	binary.BigEndian.PutUint64(header[8:16], uint64(len(bootstrap.Module)))
	copy(header[16:], digest[:])
	if err := writeAll(writer, header[:]); err != nil {
		return fmt.Errorf("write Wasm bootstrap header: %w", err)
	}
	if err := writeAll(writer, bootstrap.Module); err != nil {
		return fmt.Errorf("write Wasm bootstrap module: %w", err)
	}
	return nil
}

func ReadWasmBootstrap(reader io.Reader) (WasmBootstrap, error) {
	if reader == nil {
		return WasmBootstrap{}, errors.New("Wasm bootstrap reader is required")
	}
	var header [48]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return WasmBootstrap{}, fmt.Errorf("read Wasm bootstrap header: %w", err)
	}
	if !bytes.Equal(header[:4], wasmBootstrapMagic[:]) {
		return WasmBootstrap{}, errors.New("Wasm bootstrap magic is invalid")
	}
	pages := binary.BigEndian.Uint32(header[4:8])
	size := binary.BigEndian.Uint64(header[8:16])
	if pages < MinWasmMemoryPages || pages > MaxWasmMemoryPages || size == 0 || size > MaxWasmModuleBytes {
		return WasmBootstrap{}, errors.New("Wasm bootstrap budget is invalid")
	}
	module := make([]byte, int(size))
	if _, err := io.ReadFull(reader, module); err != nil {
		return WasmBootstrap{}, fmt.Errorf("read Wasm bootstrap module: %w", err)
	}
	digest := sha256.Sum256(module)
	if !bytes.Equal(header[16:], digest[:]) {
		return WasmBootstrap{}, errors.New("Wasm bootstrap digest mismatch")
	}
	return WasmBootstrap{MemoryLimitPages: pages, Module: module}, nil
}

func validateWasmBootstrap(bootstrap WasmBootstrap) error {
	if bootstrap.MemoryLimitPages < MinWasmMemoryPages || bootstrap.MemoryLimitPages > MaxWasmMemoryPages ||
		len(bootstrap.Module) == 0 || len(bootstrap.Module) > MaxWasmModuleBytes {
		return errors.New("Wasm bootstrap budget is invalid")
	}
	return nil
}
