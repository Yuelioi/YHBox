package llm

import (
	"context"
	"errors"
	"net/http"
)

type ErrKind string

const (
	KindAuth       ErrKind = "auth"
	KindRateLimit  ErrKind = "rateLimit"
	KindQuota      ErrKind = "quota"
	KindUpstream   ErrKind = "upstream"
	KindNetwork    ErrKind = "network"
	KindTimeout    ErrKind = "timeout"
	KindNotFound   ErrKind = "notFound"
	KindBadRequest ErrKind = "badRequest"
	KindUnknown    ErrKind = "unknown"
)

type CodedError struct {
	Kind ErrKind
	Err  error
}

func (e *CodedError) Error() string { return string(e.Kind) + ": " + e.Err.Error() }
func (e *CodedError) Unwrap() error { return e.Err }

// KindOf 剥出错误分类: CodedError 直读; deadline 归 timeout; 其余 unknown。
func KindOf(err error) ErrKind {
	var ce *CodedError
	if errors.As(err, &ce) {
		return ce.Kind
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return KindTimeout
	}
	return KindUnknown
}

// ErrAPIKeyRequired: 官方付费端点缺 key 时 New 早返此错, 免等到真调用才炸。
var ErrAPIKeyRequired = &CodedError{Kind: KindAuth, Err: errors.New("api key required for official endpoint")}

func statusToKind(status int) ErrKind {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return KindAuth
	case status == http.StatusTooManyRequests:
		return KindRateLimit
	case status == http.StatusNotFound:
		return KindNotFound
	case status == http.StatusBadRequest:
		return KindBadRequest
	case status >= 500:
		return KindUpstream
	default:
		return KindUnknown
	}
}
