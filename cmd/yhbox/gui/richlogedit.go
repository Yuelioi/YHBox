package gui

// RichLogEdit 是裹了 Win32 RichEdit50W 的 walk Widget。
// walk 没有原生 RichEdit；我们用 walk.InitWidget 把 RichEdit50W 当成
// walk 树里的一员，layout/Visible/SetFont 自动可用。

import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

// win 包没定义的 Win32 消息/常量集中放这。
const (
	emGetFirstVisibleLine = 0x00CE
	emLineScroll          = 0x00B6
	wmVScroll             = 0x0115
	sbBottom              = 7
	ecLeftMargin          = 0x0001
	ecRightMargin         = 0x0002
)

// 加载 Msftedit.dll 拿到 "RichEdit50W" 窗口类。仅加载一次。
var richEditOnce sync.Once

func ensureRichEditDLL() {
	richEditOnce.Do(func() {
		syscall.NewLazyDLL("Msftedit.dll").Load()
	})
}

// RichLogEdit 行为约等于多行只读 TextEdit + 支持每段单独 SetTextColor。
type RichLogEdit struct {
	walk.WidgetBase
}

// NewRichLogEdit 在 parent 下创建一个 RichEdit50W 控件。
func NewRichLogEdit(parent walk.Container) (*RichLogEdit, error) {
	ensureRichEditDLL()
	re := new(RichLogEdit)
	err := walk.InitWidget(
		re, parent, "RichEdit50W",
		win.WS_CHILD|win.WS_VISIBLE|win.WS_VSCROLL|
			win.ES_MULTILINE|win.ES_READONLY|win.ES_AUTOVSCROLL,
		0,
	)
	if err != nil {
		return nil, err
	}
	re.applyTightParaFormat()
	re.SetWrapText(false) // 默认不换行；调用方按 settings 调整
	// 白色背景跟其他 GroupBox 一致（默认 ES_READONLY 是 COLOR_3DFACE 灰）
	win.SendMessage(re.Handle(), win.EM_SETBKGNDCOLOR, 0, uintptr(win.RGB(255, 255, 255)))
	// 文本左右内边距 4px (EC_LEFTMARGIN|EC_RIGHTMARGIN, MAKELONG(left,right))
	textMargins := uintptr(4) | (uintptr(4) << 16)
	win.SendMessage(re.Handle(), win.EM_SETMARGINS, ecLeftMargin|ecRightMargin, textMargins)
	// 剥 WS_EX_CLIENTEDGE（walk 默认给 Edit 类加的凹陷边框）；SWP_FRAMECHANGED 必须
	hwnd := re.Handle()
	exStyle := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, exStyle&^win.WS_EX_CLIENTEDGE)
	win.SetWindowPos(hwnd, 0, 0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_FRAMECHANGED)
	return re, nil
}

// paraFormat2 是 PARAFORMAT2 的 Go 镜像（win 包只有 1.0）。字段必须严格按 MSFTEDIT 顺序。
type paraFormat2 struct {
	CbSize           uint32
	DwMask           uint32
	WNumbering       uint16
	WEffects         uint16
	DxStartIndent    int32
	DxRightIndent    int32
	DxOffset         int32
	WAlignment       uint16
	CTabCount        int16
	RgxTabs          [win.MAX_TAB_STOPS]int32
	DySpaceBefore    int32
	DySpaceAfter     int32
	DyLineSpacing    int32
	SStyle           int16
	BLineSpacingRule byte
	BOutlineLevel    byte
	WShadingWeight   uint16
	WShadingStyle    uint16
	WNumberingStart  uint16
	WNumberingStyle  uint16
	WNumberingTab    uint16
	WBorderSpace     uint16
	WBorderWidth     uint16
	WBorders         uint16
}

// Windows PARAFORMAT2 是 188 字节。Go 对齐规则碰巧产出同样大小，靠编译期断言锁死，
// 改字段顺序或类型导致大小漂移时立刻炸编译，避免运行期 msftedit 静默拒收 cbSize。
const paraFormat2ExpectedSize = 188

var (
	_ [paraFormat2ExpectedSize - unsafe.Sizeof(paraFormat2{})]byte
	_ [unsafe.Sizeof(paraFormat2{}) - paraFormat2ExpectedSize]byte
)

// applyTightParaFormat 段落行距锁 Exactly 220 twips (~1.2 倍 9pt)。
// AppendColored 用 \v 软换行 → 整篇文档只有 1 段，初始化设一次即可。
func (e *RichLogEdit) applyTightParaFormat() {
	pf := paraFormat2{
		DwMask:           win.PFM_SPACEBEFORE | win.PFM_SPACEAFTER | win.PFM_LINESPACING,
		DyLineSpacing:    220,
		BLineSpacingRule: 4, // Exactly
	}
	pf.CbSize = uint32(unsafe.Sizeof(pf))
	win.SendMessage(e.Handle(), win.EM_SETPARAFORMAT, 0, uintptr(unsafe.Pointer(&pf)))
}

// SetWrapText 切换自动换行。
//   - wrap=true：EM_SETTARGETDEVICE(NULL, 0) → 文字按控件宽度自动换行
//   - wrap=false：EM_SETTARGETDEVICE(NULL, 1) → 超宽的行横向溢出被右边缘 clip
//     （没 ES_AUTOHSCROLL，所以不会冒出水平滚动条；等价 CSS `nowrap; overflow:hidden`）
//
// 注：RichEdit 不支持按行 ellipsis "..."，关换行时只是单纯 clip。
func (e *RichLogEdit) SetWrapText(wrap bool) {
	lineWidth := uintptr(1)
	if wrap {
		lineWidth = 0
	}
	win.SendMessage(e.Handle(), win.EM_SETTARGETDEVICE, 0, lineWidth)
}

// SetReadOnly 切换只读。
func (e *RichLogEdit) SetReadOnly(ro bool) {
	param := uintptr(0)
	if ro {
		param = 1
	}
	e.SendMessage(win.EM_SETREADONLY, param, 0)
}

// richLogLayoutItem 贪心 LayoutItem。MinSize 留 ~4 行 Consolas 9pt 高度，给日志区一个
// 视觉底线，避免切到矮 tab 时 shrinkToFit 把日志压成一条缝。
// walk 内置 NewGreedyLayoutItem 是 50×50 DIP，宽度太死也不够 4 行；自己写一个更合适。
type richLogLayoutItem struct {
	walk.LayoutItemBase
}

// richLogMinHeight96dpi：4 行 × 11pt 行距 (~14.7 px/行 @96dpi) + 一点呼吸空间 ≈ 64 px。
// GroupBox 自己还会再吃 ~15px 内边距，所以日志区实际最小高度 ~80 px，能稳定看 4 行。
const richLogMinHeight96dpi = 64

func (*richLogLayoutItem) LayoutFlags() walk.LayoutFlags {
	return walk.ShrinkableHorz | walk.GrowableHorz | walk.GreedyHorz |
		walk.ShrinkableVert | walk.GrowableVert | walk.GreedyVert
}
func (li *richLogLayoutItem) IdealSize() walk.Size { return walk.Size{Width: 100, Height: 100} }
func (li *richLogLayoutItem) MinSize() walk.Size {
	dpi := 96
	if ctx := li.Context(); ctx != nil {
		dpi = ctx.DPI()
	}
	return walk.Size{Width: 0, Height: walk.IntFrom96DPI(richLogMinHeight96dpi, dpi)}
}

// CreateLayoutItem 让 walk 把本控件当贪心 widget；最小 0×0，可被极致压扁。
func (e *RichLogEdit) CreateLayoutItem(ctx *walk.LayoutContext) walk.LayoutItem {
	return new(richLogLayoutItem)
}

// Clear 清空所有文本。
func (e *RichLogEdit) Clear() {
	empty, _ := syscall.UTF16PtrFromString("")
	e.SendMessage(win.WM_SETTEXT, 0, uintptr(unsafe.Pointer(empty)))
}

// 强制每段文本用 Consolas 9pt（walk.SetFont 对 RichEdit50W 不可靠）
var richLogFaceName = utf16Face("Consolas")

// utf16Face 把字体名转成 GDI 要求的 NUL-terminated UTF-16 数组。
// 最长 LF_FACESIZE-1 个有效字符，末位强制留 0 当 NUL 终止符。
func utf16Face(name string) [win.LF_FACESIZE]uint16 {
	var arr [win.LF_FACESIZE]uint16
	utf16, _ := syscall.UTF16FromString(name) // 末尾自带 NUL
	maxChars := win.LF_FACESIZE - 1           // 给 NUL 留位
	if len(utf16) > maxChars {
		utf16 = utf16[:maxChars]
	}
	copy(arr[:], utf16)
	return arr
}

// cfTemplate 缓存 CHARFORMAT 模板：face name / size / mask 都固定，只 CrTextColor 变。
// AppendColored 每次只改颜色字段，省 60 字节栈拷贝和重复字段写入。
var cfTemplate = func() win.CHARFORMAT {
	cf := win.CHARFORMAT{
		DwMask:     win.CFM_COLOR | win.CFM_FACE | win.CFM_SIZE | win.CFM_CHARSET,
		YHeight:    9 * 20, // 9pt
		SzFaceName: richLogFaceName,
	}
	cf.CbSize = uint32(unsafe.Sizeof(cf))
	return cf
}()

// AppendColored 在末尾插入一段带颜色的文本。color=0 用默认色（黑）。
// 不会自动换行——调用方在 text 末尾自己加 \r 或 \v。
func (e *RichLogEdit) AppendColored(text string, color walk.Color) {
	if text == "" {
		return
	}
	hwnd := e.Handle()
	// caret 移到末尾
	end := uintptr(0xFFFFFFFF)
	win.SendMessage(hwnd, win.EM_SETSEL, end, end)
	// 字符格式：复用模板，只改颜色
	cf := cfTemplate
	cf.CrTextColor = win.COLORREF(color)
	win.SendMessage(hwnd, win.EM_SETCHARFORMAT, win.SCF_SELECTION, uintptr(unsafe.Pointer(&cf)))
	// 插入文本
	utf16, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	win.SendMessage(hwnd, win.EM_REPLACESEL, 0, uintptr(unsafe.Pointer(utf16)))
}

// LineCount 当前文本行数（含末尾空行）。
func (e *RichLogEdit) LineCount() int {
	return int(int32(e.SendMessage(win.EM_GETLINECOUNT, 0, 0)))
}

// DeleteFirstLines 删除最前面 n 行（n>0；n>=LineCount 则全清）。
// 用 EM_LINEINDEX 拿第 n 行起始 char 位置，SETSEL(0,pos)+REPLACESEL("") 一刀切。
func (e *RichLogEdit) DeleteFirstLines(n int) {
	if n <= 0 {
		return
	}
	hwnd := e.Handle()
	charIdx := int32(e.SendMessage(win.EM_LINEINDEX, uintptr(n), 0))
	if charIdx < 0 {
		e.Clear() // n >= 总行数
		return
	}
	win.SendMessage(hwnd, win.EM_SETSEL, 0, uintptr(charIdx))
	empty, _ := syscall.UTF16PtrFromString("")
	win.SendMessage(hwnd, win.EM_REPLACESEL, 0, uintptr(unsafe.Pointer(empty)))
}

// ScrollToEnd 用 WM_VSCROLL(SB_BOTTOM) 绝对滚到底，与焦点状态无关。
func (e *RichLogEdit) ScrollToEnd() {
	win.SendMessage(e.Handle(), wmVScroll, sbBottom, 0)
}

// FirstVisibleLine 当前最上面可见行的 0-based index。给"保留视野"用。
func (e *RichLogEdit) FirstVisibleLine() int {
	return int(int32(e.SendMessage(emGetFirstVisibleLine, 0, 0)))
}

// LineScrollTo 让指定行成为最上面可见行（绝对滚动到该行）。
// 没有 RichEdit 直接 API；先 LINESCROLL 到 0 再 LINESCROLL line 行。
func (e *RichLogEdit) LineScrollTo(line int) {
	hwnd := e.Handle()
	// 大幅向上滚到顶（uintptr 不能直接装负数常量，先 int32 再 uintptr）
	bigUp := int32(-1 << 20)
	win.SendMessage(hwnd, emLineScroll, 0, uintptr(bigUp))
	if line > 0 {
		win.SendMessage(hwnd, emLineScroll, 0, uintptr(int32(line)))
	}
}
