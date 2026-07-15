package winutil

import (
	"errors"
	"regexp"
	"strings"

	"github.com/yottaapp/yotta/internal/automation/target"
)

// WindowHandle is retained as an adapter-facing alias of the target contract.
type WindowHandle = target.WindowHandle

// MatchSpec is retained as an adapter-facing alias of the target contract.
type MatchSpec = target.WindowMatchSpec

var (
	// ErrWindowNotFound lets callers classify an exhausted resolve timeout with errors.Is.
	ErrWindowNotFound = errors.New("窗口未找到")
	// ErrWindowStillPresent lets callers classify an exhausted wait timeout with errors.Is.
	ErrWindowStillPresent = errors.New("窗口仍存在")
	// ErrWindowAmbiguous reports that an exact selector resolved multiple
	// visible top-level windows. Callers must narrow the installed selector.
	ErrWindowAmbiguous = errors.New("窗口目标不唯一")
)

// IsEmptyMatch rejects selectors that are blank or effectively match every title.
func IsEmptyMatch(spec MatchSpec) bool {
	hasAny := spec.Title != "" || spec.Class != "" || spec.ProcessName != ""
	if !hasAny {
		return true
	}
	if spec.TitleMatch == "regex" && spec.Class == "" && spec.ProcessName == "" {
		title := strings.TrimSpace(spec.Title)
		return title == ".*" || title == ".+" || title == "^.*$" || title == "^.+$"
	}
	return false
}

// CompileTitle compiles regex title selectors and is a no-op for exact matching.
func CompileTitle(spec MatchSpec) (*regexp.Regexp, error) {
	if spec.TitleMatch != "regex" || spec.Title == "" {
		return nil, nil
	}
	return regexp.Compile(spec.Title)
}
