package winutil

import (
	"errors"
	"regexp"
	"strings"

	"github.com/yottaapp/yotta/internal/automation/target"
)

type WindowHandle = target.WindowHandle
type MatchSpec = target.WindowMatchSpec

var (
	ErrWindowNotFound     = errors.New("窗口未找到")
	ErrWindowStillPresent = errors.New("窗口仍存在")
)

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

func CompileTitle(spec MatchSpec) (*regexp.Regexp, error) {
	if spec.TitleMatch != "regex" || spec.Title == "" {
		return nil, nil
	}
	return regexp.Compile(spec.Title)
}
