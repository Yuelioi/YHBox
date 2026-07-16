// Package installed provides exact, explicitly installed automation targets.
// Workflow code binds a logical slot and never receives a native handle,
// executable path, PID, process name, or input backend selector.
package installed

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MaxResolveTimeoutMilliseconds = int64(10_000)
	profileDigestDomain           = "yotta/installed-automation-target-profile/v1"
)

type ProfileDraft struct {
	Application                appcontrol.ProfileDraft `json:"application"`
	WindowTitle                string                  `json:"windowTitle"`
	WindowClass                string                  `json:"windowClass"`
	InputBackend               string                  `json:"inputBackend"`
	CaptureBackend             string                  `json:"captureBackend"`
	ResolveTimeoutMilliseconds int64                   `json:"resolveTimeoutMilliseconds"`
}

type profileState struct {
	digest      artifact.Digest
	document    ProfileDraft
	application appcontrol.Profile
	bytes       []byte
}

type Profile struct{ state *profileState }

func SealProfile(draft ProfileDraft) (Profile, error) {
	if len(draft.Application.Arguments) != 0 {
		return Profile{}, errors.New("automation target application arguments must be empty")
	}
	application, err := appcontrol.SealProfile(draft.Application)
	if err != nil {
		return Profile{}, fmt.Errorf("seal automation target application identity: %w", err)
	}
	if !validSelector(draft.WindowTitle) || !validSelector(draft.WindowClass) {
		return Profile{}, errors.New("automation target window selector is invalid")
	}
	if draft.InputBackend != "sendinput" && draft.InputBackend != "postmessage" {
		return Profile{}, errors.New("automation target input backend is invalid")
	}
	if draft.CaptureBackend != "gdi" && draft.CaptureBackend != "wgc" {
		return Profile{}, errors.New("automation target capture backend is invalid")
	}
	if draft.ResolveTimeoutMilliseconds < 100 || draft.ResolveTimeoutMilliseconds > MaxResolveTimeoutMilliseconds {
		return Profile{}, errors.New("automation target resolve timeout is invalid")
	}
	draft.Application = application.Machine()
	raw, err := artifact.Marshal(draft)
	if err != nil {
		return Profile{}, err
	}
	digest, err := artifact.Sum(profileDigestDomain, raw)
	if err != nil {
		return Profile{}, err
	}
	return Profile{state: &profileState{digest: digest, document: draft, application: application, bytes: raw}}, nil
}

func OpenProfile(raw []byte, digest artifact.Digest) (Profile, error) {
	if !digest.Valid() || len(raw) == 0 || len(raw) > 512<<10 {
		return Profile{}, errors.New("invalid automation target profile artifact")
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Profile{}, errors.New("automation target profile is not canonical")
	}
	var draft ProfileDraft
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&draft); err != nil {
		return Profile{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Profile{}, errors.New("automation target profile contains trailing values")
	}
	sealed, err := SealProfile(draft)
	if err != nil || sealed.Digest() != digest || !bytes.Equal(sealed.Bytes(), raw) {
		return Profile{}, errors.New("automation target profile digest mismatch")
	}
	return sealed, nil
}

func (p Profile) Valid() bool { return p.state != nil && p.state.digest.Valid() }
func (p Profile) Digest() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.digest
}
func (p Profile) Bytes() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.state.bytes...)
}
func (p Profile) Machine() ProfileDraft {
	if !p.Valid() {
		return ProfileDraft{}
	}
	document := p.state.document
	document.Application.Arguments = append([]string(nil), document.Application.Arguments...)
	return document
}
func (p Profile) Application() appcontrol.Profile {
	if !p.Valid() {
		return appcontrol.Profile{}
	}
	return p.state.application
}

func VerifyProfile(profile Profile) error {
	if !profile.Valid() {
		return errors.New("automation target profile is invalid")
	}
	return appcontrol.VerifyProfile(profile.Application())
}

func validSelector(value string) bool {
	if len(value) > 512 || !utf8.ValidString(value) || strings.ContainsRune(value, 0) || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) || character == '\u061c' || character == '\u200e' || character == '\u200f' ||
			character >= '\u202a' && character <= '\u202e' || character >= '\u2066' && character <= '\u2069' {
			return false
		}
	}
	return true
}
