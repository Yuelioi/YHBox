package node

import (
	"errors"
	"fmt"
	"testing"
)

func TestFailf_CodeAndChain(t *testing.T) {
	base := errors.New("disk full")
	err := Failf(CodeWriteFailed, base, "Screenshot write %s", "a.png")

	var ne *NodeError
	if !errors.As(err, &ne) {
		t.Fatal("expected *NodeError")
	}
	if ne.ErrCode() != CodeWriteFailed {
		t.Errorf("code = %q, want %q", ne.ErrCode(), CodeWriteFailed)
	}
	if ne.Error() != "Screenshot write a.png" {
		t.Errorf("msg = %q", ne.Error())
	}
	if !errors.Is(err, base) { // 显式 cause 保链
		t.Error("error chain lost: errors.Is(base) == false")
	}
	var coded Coded
	if !errors.As(err, &coded) {
		t.Error("NodeError must satisfy Coded")
	}
}

func TestFailf_NilCause(t *testing.T) {
	err := Failf(CodeLaunchFailed, nil, "application %q", "example.app")
	if errors.Unwrap(err) != nil {
		t.Error("nil cause should Unwrap to nil")
	}
}

func TestAllErrCodesRegistered(t *testing.T) {
	for _, c := range []ErrCode{
		CodeError, CodeLaunchFailed, CodeCaptureFailed, CodeWriteFailed,
		CodeNotFound, CodeTimeout, CodePlaybackFailed, CodeSendFailed, CodeThrown,
	} {
		if _, ok := ErrorCodes[c]; !ok {
			t.Errorf("code %q not in ErrorCodes registry", c)
		}
	}
	_ = fmt.Sprint
}
