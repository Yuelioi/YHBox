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
	"regexp"
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
	TargetKind                 string                  `json:"targetKind"`
	AdapterKind                string                  `json:"adapterKind"`
	ApplicationIdentityKind    string                  `json:"applicationIdentityKind"`
	Application                appcontrol.ProfileDraft `json:"application"`
	WindowTitle                string                  `json:"windowTitle"`
	WindowClass                string                  `json:"windowClass"`
	InputBackend               string                  `json:"inputBackend"`
	CaptureBackend             string                  `json:"captureBackend"`
	MouseCounts360             int64                   `json:"mouseCounts360"`
	ResolveTimeoutMilliseconds int64                   `json:"resolveTimeoutMilliseconds"`
	ADBSerial                  string                  `json:"adbSerial,omitempty"`
	ADBProduct                 string                  `json:"adbProduct,omitempty"`
	ADBModel                   string                  `json:"adbModel,omitempty"`
	ADBDevice                  string                  `json:"adbDevice,omitempty"`
	AndroidPackage             string                  `json:"androidPackage,omitempty"`
}

type profileState struct {
	digest      artifact.Digest
	document    ProfileDraft
	application appcontrol.Profile
	bytes       []byte
}

type Profile struct{ state *profileState }

func SealProfile(draft ProfileDraft) (Profile, error) {
	if draft.TargetKind == "" {
		draft.TargetKind = TargetKindDesktopWindow
	}
	if draft.AdapterKind == "" {
		draft.AdapterKind = AdapterKindWin32
	}
	if draft.ApplicationIdentityKind == "" {
		if draft.AdapterKind == AdapterKindAndroidADB {
			draft.ApplicationIdentityKind = IdentityKindADBDevice
		} else {
			draft.ApplicationIdentityKind = IdentityKindWindowsExecutable
		}
	}
	if draft.AdapterKind == AdapterKindAndroidADB {
		return sealAndroidProfile(draft)
	}
	if draft.TargetKind != TargetKindDesktopWindow || draft.AdapterKind != AdapterKindWin32 && draft.AdapterKind != AdapterKindTest {
		return Profile{}, fmt.Errorf("automation adapter %q does not implement target kind %q", draft.AdapterKind, draft.TargetKind)
	}
	if draft.ApplicationIdentityKind != IdentityKindWindowsExecutable {
		return Profile{}, fmt.Errorf("automation adapter %q does not implement application identity %q", draft.AdapterKind, draft.ApplicationIdentityKind)
	}
	if draft.ADBSerial != "" || draft.ADBProduct != "" || draft.ADBModel != "" || draft.ADBDevice != "" || draft.AndroidPackage != "" {
		return Profile{}, errors.New("desktop automation profile contains Android identity")
	}
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
	if draft.MouseCounts360 < 0 || draft.MouseCounts360 > 10_000_000 {
		return Profile{}, errors.New("automation target mouse calibration is invalid")
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

var adbIdentityPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var androidPackagePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z][A-Za-z0-9_]*)+$`)

func sealAndroidProfile(draft ProfileDraft) (Profile, error) {
	if draft.TargetKind != TargetKindAndroidDevice || draft.ApplicationIdentityKind != IdentityKindADBDevice {
		return Profile{}, fmt.Errorf("automation adapter %q does not implement target/identity %q/%q", draft.AdapterKind, draft.TargetKind, draft.ApplicationIdentityKind)
	}
	if !adbIdentityPattern.MatchString(draft.ADBSerial) || !adbIdentityPattern.MatchString(draft.ADBProduct) ||
		!adbIdentityPattern.MatchString(draft.ADBModel) || !adbIdentityPattern.MatchString(draft.ADBDevice) ||
		!androidPackagePattern.MatchString(draft.AndroidPackage) {
		return Profile{}, errors.New("android ADB identity or package is invalid")
	}
	if draft.ResolveTimeoutMilliseconds < 100 || draft.ResolveTimeoutMilliseconds > MaxResolveTimeoutMilliseconds {
		return Profile{}, errors.New("automation target resolve timeout is invalid")
	}
	if draft.Application.Executable != "" || draft.Application.ExecutableDigest != "" || len(draft.Application.Arguments) != 0 ||
		draft.WindowTitle != "" || draft.WindowClass != "" || draft.InputBackend != "" || draft.CaptureBackend != "" || draft.MouseCounts360 != 0 {
		return Profile{}, errors.New("android automation profile contains desktop identity")
	}
	raw, err := artifact.Marshal(draft)
	if err != nil {
		return Profile{}, err
	}
	digest, err := artifact.Sum(profileDigestDomain, raw)
	if err != nil {
		return Profile{}, err
	}
	return Profile{state: &profileState{digest: digest, document: draft, bytes: raw}}, nil
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

func (p Profile) TargetKind() string {
	if !p.Valid() {
		return ""
	}
	return p.state.document.TargetKind
}

func (p Profile) AdapterKind() string {
	if !p.Valid() {
		return ""
	}
	return p.state.document.AdapterKind
}

func VerifyProfile(profile Profile) error {
	if !profile.Valid() {
		return errors.New("automation target profile is invalid")
	}
	if profile.AdapterKind() == AdapterKindAndroidADB {
		return nil
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
