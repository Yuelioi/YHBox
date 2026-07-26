package blob

import (
	"context"
	"io"
	"testing"
)

type boundedZeroReader struct {
	remaining  int64
	maxRequest int
}

func (reader *boundedZeroReader) Read(buffer []byte) (int, error) {
	if len(buffer) > reader.maxRequest {
		reader.maxRequest = len(buffer)
	}
	if reader.remaining == 0 {
		return 0, io.EOF
	}
	read := int64(len(buffer))
	if read > reader.remaining {
		read = reader.remaining
	}
	reader.remaining -= read
	return int(read), nil
}

func TestCopyBoundedStreamsTwoHundredFiftySixMiBWithFixedMemory(t *testing.T) {
	const maximum = int64(256 << 20)
	source := &boundedZeroReader{remaining: maximum}
	written, err := copyBounded(context.Background(), io.Discard, source, maximum)
	if err != nil || written != maximum {
		t.Fatalf("copyBounded() = %d, %v", written, err)
	}
	if source.maxRequest != 64<<10 {
		t.Fatalf("maximum read buffer = %d, want %d", source.maxRequest, 64<<10)
	}

	oversized := &boundedZeroReader{remaining: maximum + 1}
	written, err = copyBounded(context.Background(), io.Discard, oversized, maximum)
	if err == nil || written != maximum {
		t.Fatalf("oversized copyBounded() = %d, %v", written, err)
	}
}
