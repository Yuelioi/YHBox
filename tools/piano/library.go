package piano

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gitlab.com/gomidi/midi/v2/smf"
)

//go:embed midi
var midiFS embed.FS

// LibraryEntry 是内置 MIDI 曲目。Display 给 GUI 下拉显示。
type LibraryEntry struct {
	Display string // "流行-青花瓷"
	Asset   string // "动漫-86 Avid.mid" — 相对 midi/ 子目录的文件名
}

// BuiltinLibrary 列出 embed 进 exe 的所有 MIDI，按文件名排序。
func BuiltinLibrary() []LibraryEntry {
	var out []LibraryEntry
	_ = fs.WalkDir(midiFS, "midi", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".mid") {
			return nil
		}
		asset := strings.TrimPrefix(path, "midi/")
		display := strings.TrimSuffix(asset, ".mid")
		out = append(out, LibraryEntry{Display: display, Asset: asset})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Display < out[j].Display })
	return out
}

// LoadEmbeddedMIDI 按 LibraryEntry.Asset 读 embed 字节流并解析。
func LoadEmbeddedMIDI(asset string) (*MIDIData, error) {
	data, err := fs.ReadFile(midiFS, "midi/"+asset)
	if err != nil {
		return nil, fmt.Errorf("读内置 MIDI: %w", err)
	}
	s, err := smf.ReadFrom(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("解析 MIDI: %w", err)
	}
	return parseSMF(s)
}
