package pluginconformance

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yottaapp/yotta/internal/pluginprotocol"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestGeneratedVectorsAreCanonicalForEveryGuestTransport(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "contracts", "plugin", "v1", "conformance", "vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Protocol string `json:"protocol"`
		Vectors  []struct {
			Name, Direction, FrameBase64, SHA256 string
		} `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if document.Protocol != pluginprotocol.Protocol || len(document.Vectors) < 4 {
		t.Fatalf("vector document = %#v", document)
	}
	for _, vector := range document.Vectors {
		payload, err := base64.StdEncoding.DecodeString(vector.FrameBase64)
		if err != nil {
			t.Fatalf("%s: %v", vector.Name, err)
		}
		digest := sha256.Sum256(payload)
		if hex.EncodeToString(digest[:]) != vector.SHA256 {
			t.Fatalf("%s: digest mismatch", vector.Name)
		}
		frame, err := pluginprotocol.UnmarshalFrame(payload)
		if err != nil {
			t.Fatalf("%s: %v", vector.Name, err)
		}
		opened, err := pluginprotocol.MarshalFrame(frame)
		if err != nil || !bytes.Equal(opened, payload) {
			t.Fatalf("%s: vector is not canonical", vector.Name)
		}
	}
}

func TestSharedDecoderRejectsProtocolMismatchUnknownFieldsAndOversize(t *testing.T) {
	wrongProtocol, err := (proto.MarshalOptions{Deterministic: true}).Marshal(&pluginprotocol.Frame{
		Protocol: "yotta.plugin/0", Sequence: 1,
		Payload: &pluginprotocol.Frame_Cancel{Cancel: &pluginprotocol.Cancel{Reason: "user_cancelled"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pluginprotocol.UnmarshalFrame(wrongProtocol); err == nil {
		t.Fatal("accepted a mismatched protocol version")
	}
	valid, err := pluginprotocol.MarshalFrame(&pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 1,
		Payload: &pluginprotocol.Frame_Cancel{Cancel: &pluginprotocol.Cancel{Reason: "user_cancelled"}}})
	if err != nil {
		t.Fatal(err)
	}
	unknown := protowire.AppendTag(append([]byte(nil), valid...), 99, protowire.VarintType)
	unknown = protowire.AppendVarint(unknown, 1)
	if _, err := pluginprotocol.UnmarshalFrame(unknown); err == nil {
		t.Fatal("accepted an unknown Protobuf field")
	}
	if _, err := pluginprotocol.UnmarshalFrame(make([]byte, pluginprotocol.MaxFrameBytes+1)); err == nil {
		t.Fatal("accepted an oversized frame")
	}
}
