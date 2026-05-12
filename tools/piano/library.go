package piano

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gitlab.com/gomidi/midi/v2/smf"

	"yhbox"
)

// LibraryEntry 是内置 MIDI 曲目。Display 给 GUI 下拉显示。
type LibraryEntry struct {
	Display string // "流行-青花瓷"
	Asset   string // "midi/流行-青花瓷.mid" — embed 内路径，不含 assets/ 前缀
}

// BuiltinLibrary 列出 embed 进 exe 的所有 MIDI，按文件名排序。
func BuiltinLibrary() []LibraryEntry {
	var out []LibraryEntry
	for _, a := range yhbox.AssetList() {
		if !strings.HasPrefix(a, "midi/") || !strings.HasSuffix(a, ".mid") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(a, "midi/"), ".mid")
		out = append(out, LibraryEntry{Display: name, Asset: a})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Display < out[j].Display })
	return out
}

// LoadEmbeddedMIDI 按 LibraryEntry.Asset 读 embed 字节流并解析。
func LoadEmbeddedMIDI(asset string) (*MIDIData, error) {
	data, err := yhbox.AssetBytes(asset)
	if err != nil {
		return nil, fmt.Errorf("读内置 MIDI: %w", err)
	}
	s, err := smf.ReadFrom(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解析 MIDI: %w", err)
	}
	return parseSMF(s)
}
