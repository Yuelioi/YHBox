package inputclip

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MediaType                = "application/vnd.yotta.input-clip"
	formatVersion            = uint32(2)
	MaxEncodedInputClipBytes = 40 << 20
	MaxInputClipEvents       = 1_000_000
	MaxInputClipDurationUs   = uint64(24 * 60 * 60 * 1_000_000)
	maxHeaderBytes           = 64 << 10
	chunkPayloadBytes        = 4080
	eventsPerChunk           = chunkPayloadBytes / 32
	chunkBytes               = 4096
	chunkHeaderBytes         = 16
)

type headerJSON struct {
	DurationUs uint64   `json:"durationUs"`
	Meta       ClipMeta `json:"meta"`
	EventCount uint32   `json:"eventCount"`
	ChunkCount uint32   `json:"chunkCount"`
}

// Encode writes the canonical InputClip v2 carrier. Presentation metadata and
// the mutable asset GUID stay in the asset record, so identical event streams
// have one content identity regardless of label or library placement.
func Encode(w io.Writer, clip *InputClip) error {
	if w == nil || clip == nil {
		return errors.New("input clip encoder requires a destination and clip")
	}
	if err := validateClip(clip); err != nil {
		return err
	}
	chunkCount := chunkCountFor(len(clip.Events))
	header, err := artifact.Marshal(headerJSON{
		DurationUs: clip.DurationUs, Meta: clip.Meta, EventCount: uint32(len(clip.Events)), ChunkCount: chunkCount,
	})
	if err != nil {
		return fmt.Errorf("encode input clip header: %w", err)
	}
	if len(header) > maxHeaderBytes || encodedSize(len(header), chunkCount) > MaxEncodedInputClipBytes {
		return errors.New("input clip exceeds byte budget")
	}
	if _, err := w.Write([]byte("ICLP")); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, formatVersion); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(header))); err != nil {
		return err
	}
	if _, err := w.Write(header); err != nil {
		return err
	}
	offsets := make([]uint32, 0, chunkCount)
	offset := uint32(12 + len(header))
	for index := uint32(0); index < chunkCount; index++ {
		offsets = append(offsets, offset)
		start := int(index) * eventsPerChunk
		end := min(start+eventsPerChunk, len(clip.Events))
		if err := writeChunk(w, clip.Events[start:end]); err != nil {
			return fmt.Errorf("encode input clip chunk %d: %w", index, err)
		}
		offset += chunkBytes
	}
	if err := binary.Write(w, binary.LittleEndian, chunkCount); err != nil {
		return err
	}
	for _, offset := range offsets {
		if err := binary.Write(w, binary.LittleEndian, offset); err != nil {
			return err
		}
	}
	return nil
}

func Decode(source io.Reader) (*InputClip, error) {
	if source == nil {
		return nil, errors.New("input clip decoder requires a source")
	}
	data, err := io.ReadAll(io.LimitReader(source, MaxEncodedInputClipBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read input clip: %w", err)
	}
	if len(data) > MaxEncodedInputClipBytes || len(data) < 16 || string(data[:4]) != "ICLP" {
		return nil, errors.New("input clip carrier is invalid")
	}
	if version := binary.LittleEndian.Uint32(data[4:8]); version != formatVersion {
		return nil, fmt.Errorf("input clip version %d is unsupported", version)
	}
	headerLength := int(binary.LittleEndian.Uint32(data[8:12]))
	if headerLength <= 0 || headerLength > maxHeaderBytes || 12+headerLength > len(data) {
		return nil, errors.New("input clip header is invalid")
	}
	headerRaw := data[12 : 12+headerLength]
	canonicalHeader, err := artifact.Canonicalize(headerRaw)
	if err != nil || !bytes.Equal(canonicalHeader, headerRaw) {
		return nil, errors.New("input clip header is not canonical")
	}
	var header headerJSON
	decoder := json.NewDecoder(bytes.NewReader(headerRaw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&header); err != nil {
		return nil, fmt.Errorf("decode input clip header: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("input clip header contains trailing JSON")
	}
	if header.EventCount > MaxInputClipEvents || header.ChunkCount != chunkCountFor(int(header.EventCount)) || encodedSize(headerLength, header.ChunkCount) != len(data) {
		return nil, errors.New("input clip structure is invalid")
	}
	clip := &InputClip{DurationUs: header.DurationUs, Meta: header.Meta, Events: make([]Event, 0, header.EventCount)}
	chunkStart := 12 + headerLength
	remaining := int(header.EventCount)
	for index := uint32(0); index < header.ChunkCount; index++ {
		base := chunkStart + int(index)*chunkBytes
		expectedEvents := min(eventsPerChunk, remaining)
		if int(binary.LittleEndian.Uint32(data[base:base+4])) != expectedEvents || !allZero(data[base+4:base+chunkHeaderBytes]) {
			return nil, fmt.Errorf("input clip chunk %d header is invalid", index)
		}
		for eventIndex := 0; eventIndex < expectedEvents; eventIndex++ {
			position := base + chunkHeaderBytes + eventIndex*32
			clip.Events = append(clip.Events, readEvent(data[position:position+32]))
		}
		padding := data[base+chunkHeaderBytes+expectedEvents*32 : base+chunkBytes]
		if !allZero(padding) {
			return nil, fmt.Errorf("input clip chunk %d padding is invalid", index)
		}
		remaining -= expectedEvents
	}
	footer := chunkStart + int(header.ChunkCount)*chunkBytes
	if binary.LittleEndian.Uint32(data[footer:footer+4]) != header.ChunkCount {
		return nil, errors.New("input clip footer count is invalid")
	}
	for index := uint32(0); index < header.ChunkCount; index++ {
		offset := binary.LittleEndian.Uint32(data[footer+4+int(index)*4 : footer+8+int(index)*4])
		if offset != uint32(chunkStart)+index*chunkBytes {
			return nil, errors.New("input clip footer offset is invalid")
		}
	}
	if err := validateClip(clip); err != nil {
		return nil, err
	}
	return clip, nil
}

func validateClip(clip *InputClip) error {
	if clip.DurationUs > MaxInputClipDurationUs || len(clip.Events) > MaxInputClipEvents {
		return errors.New("input clip exceeds duration or event budget")
	}
	if clip.Meta.MouseMode != "absolute" && clip.Meta.MouseMode != "relative" && clip.Meta.MouseMode != "mixed" {
		return errors.New("input clip mouse mode is invalid")
	}
	width, height := clip.Meta.BaseResolution[0], clip.Meta.BaseResolution[1]
	if width <= 0 || width > 100_000 || height <= 0 || height > 100_000 || clip.Meta.MouseCounts360 < 0 || clip.Meta.MouseCounts360 > 10_000_000 || clip.Meta.StopHotkeyVK > 255 {
		return errors.New("input clip metadata is invalid")
	}
	if len(clip.Events) == 0 {
		if clip.DurationUs != 0 {
			return errors.New("empty input clip has a duration")
		}
		return nil
	}
	for index, event := range clip.Events {
		if event.TUs > MaxInputClipDurationUs || index == 0 && event.TUs != 0 || index > 0 && !clip.Events[index-1].Less(event) {
			return fmt.Errorf("input clip event %d ordering is invalid", index)
		}
		if err := validateEvent(event, clip.Meta); err != nil {
			return fmt.Errorf("input clip event %d: %w", index, err)
		}
	}
	if clip.DurationUs != clip.Events[len(clip.Events)-1].TUs {
		return errors.New("input clip duration does not match its final event")
	}
	return nil
}

func validateEvent(event Event, meta ClipMeta) error {
	switch event.Type {
	case EventTypeKeyDown, EventTypeKeyUp:
		if event.A <= 0 || event.A > 255 {
			return errors.New("virtual key is invalid")
		}
	case EventTypeMouseBtnDown, EventTypeMouseBtnUp:
		if event.A < 0 || event.A > 2 || !validClipPoint(event.B, event.C, meta.BaseResolution) {
			return errors.New("mouse button event is invalid")
		}
	case EventTypeMouseMove:
		if event.A != 0 || !validClipPoint(event.B, event.C, meta.BaseResolution) {
			return errors.New("mouse move event is invalid")
		}
	case EventTypeRawDelta:
		if event.A != 0 || meta.MouseCounts360 <= 0 {
			return errors.New("relative mouse event lacks calibration")
		}
	case EventTypeScroll:
		if event.A == 0 || !validClipPoint(event.B, event.C, meta.BaseResolution) {
			return errors.New("scroll event is invalid")
		}
	default:
		return errors.New("event type is invalid")
	}
	return nil
}

func validClipPoint(x, y int32, resolution [2]int) bool {
	return x >= 0 && y >= 0 && int64(x) < int64(resolution[0]) && int64(y) < int64(resolution[1])
}

func chunkCountFor(events int) uint32 {
	if events == 0 {
		return 0
	}
	return uint32((events + eventsPerChunk - 1) / eventsPerChunk)
}

func encodedSize(headerLength int, chunks uint32) int {
	return 12 + headerLength + int(chunks)*chunkBytes + 4 + int(chunks)*4
}

func writeChunk(w io.Writer, events []Event) error {
	var chunk [chunkBytes]byte
	binary.LittleEndian.PutUint32(chunk[:4], uint32(len(events)))
	for index, event := range events {
		writeEvent(chunk[chunkHeaderBytes+index*32:chunkHeaderBytes+(index+1)*32], event)
	}
	_, err := w.Write(chunk[:])
	return err
}

func writeEvent(destination []byte, event Event) {
	binary.LittleEndian.PutUint64(destination[0:8], event.TUs)
	binary.LittleEndian.PutUint32(destination[8:12], event.Seq)
	destination[12] = byte(event.Type)
	binary.LittleEndian.PutUint32(destination[16:20], uint32(event.A))
	binary.LittleEndian.PutUint32(destination[20:24], uint32(event.B))
	binary.LittleEndian.PutUint32(destination[24:28], uint32(event.C))
}

func readEvent(source []byte) Event {
	return Event{
		TUs: binary.LittleEndian.Uint64(source[0:8]), Seq: binary.LittleEndian.Uint32(source[8:12]), Type: EventType(source[12]),
		A: int32(binary.LittleEndian.Uint32(source[16:20])), B: int32(binary.LittleEndian.Uint32(source[20:24])), C: int32(binary.LittleEndian.Uint32(source[24:28])),
	}
}

func allZero(data []byte) bool {
	for _, value := range data {
		if value != 0 {
			return false
		}
	}
	return true
}
