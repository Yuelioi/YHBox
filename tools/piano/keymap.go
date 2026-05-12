package piano

// Modifier 修饰键。
type Modifier int

const (
	ModNone Modifier = iota
	ModShift
)

// KeyAction 一次按键动作。
type KeyAction struct {
	Key string   // input.Tap 用的键名（小写字母 / esc 等）
	Mod Modifier // 修饰键
}

// MIDI 中音 1 (C, 中央 C) = MIDI 60。游戏 3 八度：
//
//	低音 1 = MIDI 48 (C3)
//	中音 1 = MIDI 60 (C4)
//	高音 1 = MIDI 72 (C5)
//
// 范围 [midiLowest, midiHighest]（含），共 36 个半音。
const (
	midiLowest  = 48
	midiHighest = 83
)

// 7 个自然音在八度内的半音偏移（C 大调白键）。
var naturalSemitones = [7]int{0, 2, 4, 5, 7, 9, 11}

// octaveKeys[octave][naturalIndex] = 该自然音对应的物理键。
// octave: 0=低音 1=中音 2=高音；naturalIndex: 0..6 对应 1..7 (C D E F G A B)。
var octaveKeys = [3][7]string{
	{"z", "x", "c", "v", "b", "n", "m"}, // 低音
	{"a", "s", "d", "f", "g", "h", "j"}, // 中音
	{"q", "w", "e", "r", "t", "y", "u"}, // 高音
}

// MapNote 把 MIDI 音号映射成一个键位动作。返回 ok=false 表示该模式下无法弹出。
//
// 36 键策略：自然音 → 直接按；升半音 → Shift + 下方自然音。
// 游戏同时支持 Ctrl+key=低半音，但跟 Shift+下方自然音等价（C# = Shift+C = Ctrl+D），
// 算法只用 Shift 让多音和弦能在同一 modifier 组里齐按下，减少 Shift/Ctrl 切换。
//
// 21 键策略：只接受自然音；升半音返回 false。
func MapNote(midi int, mode Mode) (KeyAction, bool) {
	if midi < midiLowest || midi > midiHighest {
		return KeyAction{}, false
	}
	rel := midi - midiLowest // 0..35
	octave := rel / 12       // 0..2
	semitone := rel % 12     // 0..11

	for natIdx, st := range naturalSemitones {
		if st == semitone {
			return KeyAction{Key: octaveKeys[octave][natIdx], Mod: ModNone}, true
		}
	}

	if mode == Keys21 {
		return KeyAction{}, false
	}

	// 升半音：从下方自然音 + Shift
	for natIdx, st := range naturalSemitones {
		if st+1 == semitone {
			return KeyAction{Key: octaveKeys[octave][natIdx], Mod: ModShift}, true
		}
	}
	return KeyAction{}, false
}

// MapNoteWithOctaveShift 试 1 次 ±12 兜底。超范围太多（>1 八度）直接放弃，
// 防止 per-note 大幅八度跳让旋律线乱掉——丢音好过乱跳。
// 用户配置的 OctaveOffset 已经叠加在 midi 上；这里是额外兜底。
func MapNoteWithOctaveShift(midi int, mode Mode, maxShift int) (KeyAction, bool) {
	if a, ok := MapNote(midi, mode); ok {
		return a, true
	}
	// 太低 → 上抬 1 八度；太高 → 下压 1 八度。只试 1 次，避免乱跳。
	if midi < midiLowest {
		if a, ok := MapNote(midi+12, mode); ok {
			return a, true
		}
	} else if midi > midiHighest {
		if a, ok := MapNote(midi-12, mode); ok {
			return a, true
		}
	}
	return KeyAction{}, false
}
