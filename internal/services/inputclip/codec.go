package inputclip

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// 文件格式:
// [Magic "ICLP" 4B][Version uint32 LE 4B]
// [HeaderJSONLen uint32 LE 4B][HeaderJSON ...]
// [Chunk 0: ChunkLen uint32 LE 4B][events ... padding to 4KB]
// [Chunk 1: 同上]
// ...
// [Footer: ChunkCount uint32 LE][ChunkOffsets []uint32 LE]

const (
	formatVersion     = uint32(1)
	chunkPayloadBytes = 4080                    // 4096 - 16 字节 chunk header (LenU32 + reserved)
	eventsPerChunk    = chunkPayloadBytes / 32  // = 127, 32 字节一个 Event
	chunkHeaderBytes  = 16
)

// headerJSON 写盘的 header (events 不在这里, 走 chunks)
type headerJSON struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	DurationUs  uint64   `json:"durationUs"`
	CreatedAt   string   `json:"createdAt"`
	Meta        ClipMeta `json:"meta"`
	EventCount  uint32   `json:"eventCount"`
	ChunkCount  uint32   `json:"chunkCount"`
}

// Encode 写 InputClip 到 w. binary chunk stream format.
func Encode(w io.Writer, clip *InputClip) error {
	// 1) Magic + Version
	if _, err := w.Write([]byte("ICLP")); err != nil {
		return fmt.Errorf("magic: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, formatVersion); err != nil {
		return fmt.Errorf("version: %w", err)
	}

	// 2) HeaderJSON
	chunkCount := uint32((len(clip.Events) + eventsPerChunk - 1) / eventsPerChunk)
	if len(clip.Events) == 0 {
		chunkCount = 0
	}
	hdr := headerJSON{
		ID: clip.ID, Label: clip.Label, Description: clip.Description, Category: clip.Category, Tags: clip.Tags,
		DurationUs: clip.DurationUs, CreatedAt: clip.CreatedAt, Meta: clip.Meta,
		EventCount: uint32(len(clip.Events)), ChunkCount: chunkCount,
	}
	hjson, err := json.Marshal(hdr)
	if err != nil {
		return fmt.Errorf("header json: %w", err)
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(hjson))); err != nil {
		return fmt.Errorf("header len: %w", err)
	}
	if _, err := w.Write(hjson); err != nil {
		return fmt.Errorf("header body: %w", err)
	}

	// 3) Chunks
	chunkOffsets := make([]uint32, 0, chunkCount)
	// Header 段总长度: Magic(4) + Version(4) + HeaderLen(4) + Header(N)
	currentOffset := uint32(12 + len(hjson))
	for i := 0; i < int(chunkCount); i++ {
		chunkOffsets = append(chunkOffsets, currentOffset)
		start := i * eventsPerChunk
		end := start + eventsPerChunk
		if end > len(clip.Events) {
			end = len(clip.Events)
		}
		events := clip.Events[start:end]
		if err := writeChunk(w, events); err != nil {
			return fmt.Errorf("chunk %d: %w", i, err)
		}
		currentOffset += 4096
	}

	// 4) Footer: ChunkCount + offset table
	if err := binary.Write(w, binary.LittleEndian, chunkCount); err != nil {
		return fmt.Errorf("footer count: %w", err)
	}
	for _, off := range chunkOffsets {
		if err := binary.Write(w, binary.LittleEndian, off); err != nil {
			return fmt.Errorf("footer offset: %w", err)
		}
	}
	return nil
}

func writeChunk(w io.Writer, events []Event) error {
	// Chunk header: Len uint32 LE + reserved 12 字节 = 16 字节
	if err := binary.Write(w, binary.LittleEndian, uint32(len(events))); err != nil {
		return err
	}
	if _, err := w.Write(make([]byte, 12)); err != nil { // reserved
		return err
	}
	// Events: 每个 32 字节, 显式按字段写 (避免 _ padding 字段被 binary.Write 当 unexported 出错)
	for _, ev := range events {
		if err := writeEvent(w, ev); err != nil {
			return err
		}
	}
	// Padding 到 4KB
	used := 16 + len(events)*32
	if used < 4096 {
		if _, err := w.Write(make([]byte, 4096-used)); err != nil {
			return err
		}
	}
	return nil
}

// writeEvent 显式按字段写 32 字节布局 — 不依赖 binary.Write 对 struct 的反射.
// 布局: TUs(8) + Seq(4) + Type(1) + pad(3) + A(4) + B(4) + C(4) + pad(4) = 32
func writeEvent(w io.Writer, ev Event) error {
	var buf [32]byte
	binary.LittleEndian.PutUint64(buf[0:8], ev.TUs)
	binary.LittleEndian.PutUint32(buf[8:12], ev.Seq)
	buf[12] = byte(ev.Type)
	// buf[13:16] padding 0
	binary.LittleEndian.PutUint32(buf[16:20], uint32(ev.A))
	binary.LittleEndian.PutUint32(buf[20:24], uint32(ev.B))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(ev.C))
	// buf[28:32] padding 0
	_, err := w.Write(buf[:])
	return err
}

// readEvent 从 32 字节 buf 解出 Event.
func readEvent(buf []byte) Event {
	return Event{
		TUs:  binary.LittleEndian.Uint64(buf[0:8]),
		Seq:  binary.LittleEndian.Uint32(buf[8:12]),
		Type: EventType(buf[12]),
		A:    int32(binary.LittleEndian.Uint32(buf[16:20])),
		B:    int32(binary.LittleEndian.Uint32(buf[20:24])),
		C:    int32(binary.LittleEndian.Uint32(buf[24:28])),
	}
}

// Decode 读 InputClip from r.
func Decode(r io.Reader) (*InputClip, error) {
	// 读全部到 bytes (binary.Read 需要 Reader, 没 Seeker)
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("file too short %d", len(data))
	}
	if string(data[:4]) != "ICLP" {
		return nil, fmt.Errorf("magic 错: %q", data[:4])
	}
	version := binary.LittleEndian.Uint32(data[4:8])
	if version != formatVersion {
		return nil, fmt.Errorf("version %d 不支持 (当前 %d)", version, formatVersion)
	}
	hdrLen := binary.LittleEndian.Uint32(data[8:12])
	if uint32(len(data)) < 12+hdrLen {
		return nil, fmt.Errorf("header truncated")
	}
	var hdr headerJSON
	if err := json.Unmarshal(data[12:12+hdrLen], &hdr); err != nil {
		return nil, fmt.Errorf("header json: %w", err)
	}
	clip := &InputClip{
		ID: hdr.ID, Label: hdr.Label, Description: hdr.Description, Category: hdr.Category, Tags: hdr.Tags,
		DurationUs: hdr.DurationUs, CreatedAt: hdr.CreatedAt, Meta: hdr.Meta,
	}
	// Chunks
	chunkStart := 12 + int(hdrLen)
	clip.Events = make([]Event, 0, hdr.EventCount)
	for i := 0; i < int(hdr.ChunkCount); i++ {
		base := chunkStart + i*4096
		if base+16 > len(data) {
			return nil, fmt.Errorf("chunk %d truncated", i)
		}
		evLen := binary.LittleEndian.Uint32(data[base : base+4])
		// reserved 12 字节跳过
		for j := uint32(0); j < evLen; j++ {
			pos := base + 16 + int(j)*32
			if pos+32 > len(data) {
				return nil, fmt.Errorf("chunk %d event %d truncated", i, j)
			}
			clip.Events = append(clip.Events, readEvent(data[pos:pos+32]))
		}
	}
	return clip, nil
}
