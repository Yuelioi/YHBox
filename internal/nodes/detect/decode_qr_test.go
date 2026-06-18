// internal/nodes/detect/decode_qr_test.go
package detect

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func runDecodeQR(t *testing.T, vision *mockVision) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&DecodeQR{})
	rn, _ := node.Get("DecodeQR")
	return node.RunNode(context.Background(), rn, nil, map[string]any{}, nil, withVision(vision), false)
}

func TestDecodeQR_FoundFirstOfMany(t *testing.T) {
	vision := &mockVision{qrResults: []node.QRResult{
		{Text: "first", Points: []node.Point{{X: 0.1, Y: 0.1}}},
		{Text: "second", Points: []node.Point{{X: 0.5, Y: 0.5}}},
	}}
	r := runDecodeQR(t, vision)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != dqOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	if r.OutputData[dqDataText] != "first" {
		t.Errorf("Text = %v, want first", r.OutputData[dqDataText])
	}
	if r.OutputData[dqDataCount] != 2 {
		t.Errorf("Count = %v, want 2", r.OutputData[dqDataCount])
	}
}

func TestDecodeQR_NotFoundWhenEmpty(t *testing.T) {
	r := runDecodeQR(t, &mockVision{qrResults: nil})
	if r.ExitName != dqOutNotFound {
		t.Fatalf("exit = %q, want NotFound", r.ExitName)
	}
	if r.OutputData[dqDataCount] != 0 {
		t.Errorf("Count = %v, want 0", r.OutputData[dqDataCount])
	}
}

func TestDecodeQR_ErrorPropagates(t *testing.T) {
	r := runDecodeQR(t, &mockVision{qrErr: errors.New("capture failed")})
	if r.Error == nil {
		t.Error("expected error propagation")
	}
}
