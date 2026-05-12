package piano

import (
	"fmt"
	"math"
	"sort"
	"time"

	"gitlab.com/gomidi/midi/v2/smf"
)

// NoteEvent 一个绝对时间戳的 NoteOn 事件。
type NoteEvent struct {
	At      time.Duration // 自曲首起的绝对时间
	Key     uint8         // MIDI 音号
	Channel uint8         // MIDI channel 0..15 (9 = drums)
}

// TrackInfo 一个 MIDI track 的统计 + 事件。
type TrackInfo struct {
	Name      string // MIDI MetaTrackName 提取的轨道名（无名 = ""）
	Events    []NoteEvent
	NoteCount int
	AvgPitch  float64 // 用于挑主旋律轨：越高越像主旋律
	HasDrums  bool    // 有任何事件落在 channel 9
}

// MIDIData 整曲解析结果。
type MIDIData struct {
	Tracks []TrackInfo
}

// AllEvents 合所有 track 的非鼓事件，按时间排序。
func (d *MIDIData) AllEvents() []NoteEvent {
	var all []NoteEvent
	for _, t := range d.Tracks {
		for _, e := range t.Events {
			if e.Channel == 9 { // 跳过鼓事件
				continue
			}
			all = append(all, e)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].At < all[j].At })
	return all
}

// SmartEvents 智能选轨：丢掉鼓和稀疏轨，剩下的按"平均音高"挑最高的（通常是主旋律）。
// 稀疏阈值 = max(10, 总音符数的 5%)，过滤掉只有 1-2 个音的标记 / 控制轨。
// 没合适候选时退化到 AllEvents。
func (d *MIDIData) SmartEvents() []NoteEvent {
	totalCount := 0
	for _, t := range d.Tracks {
		totalCount += t.NoteCount
	}
	threshold := totalCount / 20
	if threshold < 10 {
		threshold = 10
	}

	bestIdx, bestScore := -1, -math.MaxFloat64
	for i, t := range d.Tracks {
		if t.HasDrums || t.NoteCount < threshold {
			continue
		}
		// 通过稀疏过滤后，平均音高最高的最像主旋律
		if t.AvgPitch > bestScore {
			bestIdx, bestScore = i, t.AvgPitch
		}
	}
	if bestIdx < 0 {
		return d.AllEvents()
	}
	return d.Tracks[bestIdx].Events
}

// AutoTranspose 算把整曲整体平移多少个八度能让最多音落进 [midiLowest, midiHighest]。
// 用这个值给所有音统一加偏移 → 保持音程关系，旋律线轮廓不变。
// 不平移就能完全覆盖时返回 0；相同覆盖率优先选 |offset| 更小的。
func AutoTranspose(events []NoteEvent) int {
	if len(events) == 0 {
		return 0
	}
	bestOffset, bestFit := 0, -1
	for off := -4; off <= 4; off++ {
		fit := 0
		for _, ev := range events {
			n := int(ev.Key) + off*12
			if n >= midiLowest && n <= midiHighest {
				fit++
			}
		}
		if fit > bestFit || (fit == bestFit && abs(off) < abs(bestOffset)) {
			bestOffset, bestFit = off, fit
		}
	}
	return bestOffset
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// LoadMIDI 读 .mid 文件，返回 per-track 信息 + 事件。
func LoadMIDI(path string) (*MIDIData, error) {
	s, err := smf.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读 MIDI: %w", err)
	}
	return parseSMF(s)
}

// parseSMF 是 LoadMIDI / LoadEmbeddedMIDI 共用的解析逻辑。
// 处理 SMF1 多 track；TempoChanges 跨 track 应用（gomidi 已在 SMF 层聚合）。
func parseSMF(s *smf.SMF) (*MIDIData, error) {
	mt, ok := s.TimeFormat.(smf.MetricTicks)
	if !ok {
		return nil, fmt.Errorf("不支持的 TimeFormat: %T（仅支持 MetricTicks）", s.TimeFormat)
	}
	tempos := s.TempoChanges()

	data := &MIDIData{}
	for _, track := range s.Tracks {
		var t TrackInfo
		var absTicks int64
		var pitchSum int
		for _, ev := range track {
			absTicks += int64(ev.Delta)
			// 提取轨道名（通常在 track 起始的 meta 事件里）
			if t.Name == "" {
				var name string
				if ev.Message.GetMetaTrackName(&name) && name != "" {
					t.Name = name
				}
			}
			var ch, key, vel uint8
			if ev.Message.GetNoteStart(&ch, &key, &vel) && vel > 0 {
				t.Events = append(t.Events, NoteEvent{
					At:      ticksToDuration(mt, tempos, absTicks),
					Key:     key,
					Channel: ch,
				})
				pitchSum += int(key)
				if ch == 9 {
					t.HasDrums = true
				}
			}
		}
		t.NoteCount = len(t.Events)
		if t.NoteCount > 0 {
			t.AvgPitch = float64(pitchSum) / float64(t.NoteCount)
		}
		data.Tracks = append(data.Tracks, t)
	}
	return data, nil
}

// TrackEvents 返回指定 track 的事件（跳过 channel 9 鼓点）。idx 越界返回 nil。
func (d *MIDIData) TrackEvents(idx int) []NoteEvent {
	if idx < 0 || idx >= len(d.Tracks) {
		return nil
	}
	var out []NoteEvent
	for _, e := range d.Tracks[idx].Events {
		if e.Channel == 9 {
			continue
		}
		out = append(out, e)
	}
	return out
}

// ticksToDuration 按 TempoChanges 分段累加每个 tempo 区间的时长。
func ticksToDuration(mt smf.MetricTicks, tempos smf.TempoChanges, absTicks int64) time.Duration {
	var total time.Duration
	var cursor int64
	for i, tc := range tempos {
		var segEnd int64
		if i+1 < len(tempos) {
			segEnd = tempos[i+1].AbsTicks
		} else {
			segEnd = absTicks
		}
		if segEnd > absTicks {
			segEnd = absTicks
		}
		if segEnd > cursor {
			total += mt.Duration(tc.BPM, uint32(segEnd-cursor))
		}
		cursor = segEnd
		if cursor >= absTicks {
			break
		}
	}
	// 无 tempo 事件或 cursor 没走到末尾 → 用默认 120 bpm
	if cursor < absTicks {
		total += mt.Duration(120, uint32(absTicks-cursor))
	}
	return total
}
