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
	"github.com/yottaapp/yotta/internal/automation/browsercdp"
)

const (
	profileDigestDomain    = "yotta/installed-automation-target-profile/v2"
	maxProfilePayloadBytes = 512 << 10
)

// ProfileDraft is a versioned discriminated envelope. Adapter-native fields
// live in Payload and are decoded only by the registered adapter sealer.
type ProfileDraft struct {
	TargetKind     string          `json:"targetKind"`
	AdapterKind    string          `json:"adapterKind"`
	ProfileVersion string          `json:"profileVersion"`
	Payload        json.RawMessage `json:"payload"`
}

type DesktopProfilePayload struct {
	Application                appcontrol.ProfileDraft `json:"application"`
	WindowTitle                string                  `json:"windowTitle"`
	WindowTitleMatch           string                  `json:"windowTitleMatch"`
	WindowSelection            string                  `json:"windowSelection"`
	WindowClass                string                  `json:"windowClass"`
	InputBackend               string                  `json:"inputBackend"`
	CaptureBackend             string                  `json:"captureBackend"`
	MouseCounts360             int64                   `json:"mouseCounts360"`
	ResolveTimeoutMilliseconds int64                   `json:"resolveTimeoutMilliseconds"`
}

type AndroidProfilePayload struct {
	ADBSerial                  string `json:"adbSerial"`
	ADBProduct                 string `json:"adbProduct"`
	ADBModel                   string `json:"adbModel"`
	ADBDevice                  string `json:"adbDevice"`
	AndroidPackage             string `json:"androidPackage"`
	ResolveTimeoutMilliseconds int64  `json:"resolveTimeoutMilliseconds"`
}

type BrowserProfilePayload struct {
	BrowserEndpoint            string `json:"browserEndpoint"`
	BrowserTargetID            string `json:"browserTargetId"`
	BrowserWebSocketURL        string `json:"browserWebSocketUrl"`
	BrowserTitle               string `json:"browserTitle"`
	BrowserURL                 string `json:"browserUrl"`
	ResolveTimeoutMilliseconds int64  `json:"resolveTimeoutMilliseconds"`
}

type DesktopProfileIntent struct {
	ApplicationSlot            string `json:"applicationSlot"`
	WindowTitle                string `json:"windowTitle"`
	WindowTitleMatch           string `json:"windowTitleMatch"`
	WindowSelection            string `json:"windowSelection"`
	WindowClass                string `json:"windowClass"`
	InputBackend               string `json:"inputBackend"`
	CaptureBackend             string `json:"captureBackend"`
	MouseCounts360             int64  `json:"mouseCounts360"`
	ResolveTimeoutMilliseconds int64  `json:"resolveTimeoutMilliseconds"`
}

type AndroidProfileIntent = AndroidProfilePayload
type BrowserProfileIntent = BrowserProfilePayload
type ApplicationProfileResolver func(string) (appcontrol.ProfileDraft, bool)

type profileIntentCodec struct {
	draft           func(json.RawMessage, ApplicationProfileResolver) (ProfileDraft, error)
	applicationSlot func(json.RawMessage) (string, error)
}

func ProfileDraftFromIntent(targetKind, adapterKind, profileVersion string, payload json.RawMessage, resolve ApplicationProfileResolver) (ProfileDraft, error) {
	return profileDraftFromIntentWithRegistry(targetKind, adapterKind, profileVersion, payload, resolve, defaultAdapterRegistry())
}

func profileDraftFromIntentWithRegistry(targetKind, adapterKind, profileVersion string, payload json.RawMessage, resolve ApplicationProfileResolver, registry adapterRegistry) (ProfileDraft, error) {
	registered, ok := registry.byKind[adapterKind]
	if !ok || registered.targetType.TargetKind != targetKind || registered.targetType.ProfileVersion != profileVersion || registered.intent.draft == nil {
		return ProfileDraft{}, fmt.Errorf("automation adapter %q does not implement target/profile intent %q/v%s", adapterKind, targetKind, profileVersion)
	}
	return registered.intent.draft(payload, resolve)
}

func ApplicationSlotFromIntent(targetKind, adapterKind, profileVersion string, payload json.RawMessage) (string, error) {
	registered, ok := defaultAdapterRegistry().byKind[adapterKind]
	if !ok || registered.targetType.TargetKind != targetKind || registered.targetType.ProfileVersion != profileVersion || registered.intent.applicationSlot == nil {
		return "", fmt.Errorf("automation adapter %q does not implement target/profile intent %q/v%s", adapterKind, targetKind, profileVersion)
	}
	return registered.intent.applicationSlot(payload)
}

func desktopProfileIntentCodec() profileIntentCodec {
	decode := func(raw json.RawMessage) (DesktopProfileIntent, error) {
		var intent DesktopProfileIntent
		if err := decodeProfilePayload(raw, &intent); err != nil {
			return DesktopProfileIntent{}, err
		}
		return intent, nil
	}
	return profileIntentCodec{
		draft: func(raw json.RawMessage, resolve ApplicationProfileResolver) (ProfileDraft, error) {
			intent, err := decode(raw)
			if err != nil {
				return ProfileDraft{}, err
			}
			if resolve == nil {
				return ProfileDraft{}, errors.New("desktop automation profile requires an application resolver")
			}
			application, ok := resolve(intent.ApplicationSlot)
			if !ok {
				return ProfileDraft{}, fmt.Errorf("desktop automation profile references unknown application slot %q", intent.ApplicationSlot)
			}
			application.Arguments = []string{}
			return NewDesktopProfileDraft(DesktopProfilePayload{
				Application: application, WindowTitle: intent.WindowTitle, WindowTitleMatch: intent.WindowTitleMatch,
				WindowSelection: intent.WindowSelection, WindowClass: intent.WindowClass, InputBackend: intent.InputBackend,
				CaptureBackend: intent.CaptureBackend, MouseCounts360: intent.MouseCounts360,
				ResolveTimeoutMilliseconds: intent.ResolveTimeoutMilliseconds,
			}), nil
		},
		applicationSlot: func(raw json.RawMessage) (string, error) {
			intent, err := decode(raw)
			return intent.ApplicationSlot, err
		},
	}
}

func androidProfileIntentCodec() profileIntentCodec {
	return profileIntentCodec{
		draft: func(raw json.RawMessage, _ ApplicationProfileResolver) (ProfileDraft, error) {
			var intent AndroidProfileIntent
			if err := decodeProfilePayload(raw, &intent); err != nil {
				return ProfileDraft{}, err
			}
			return NewAndroidProfileDraft(intent), nil
		},
		applicationSlot: func(json.RawMessage) (string, error) { return "", nil },
	}
}

func browserProfileIntentCodec() profileIntentCodec {
	return profileIntentCodec{
		draft: func(raw json.RawMessage, _ ApplicationProfileResolver) (ProfileDraft, error) {
			var intent BrowserProfileIntent
			if err := decodeProfilePayload(raw, &intent); err != nil {
				return ProfileDraft{}, err
			}
			return NewBrowserProfileDraft(intent), nil
		},
		applicationSlot: func(json.RawMessage) (string, error) { return "", nil },
	}
}

func NewDesktopProfileDraft(payload DesktopProfilePayload) ProfileDraft {
	return newProfileDraft(TargetKindDesktopWindow, AdapterKindWin32, payload)
}

func NewAndroidProfileDraft(payload AndroidProfilePayload) ProfileDraft {
	return newProfileDraft(TargetKindAndroidDevice, AdapterKindAndroidADB, payload)
}

func NewBrowserProfileDraft(payload BrowserProfilePayload) ProfileDraft {
	return newProfileDraft(TargetKindBrowserCDP, AdapterKindBrowserCDP, payload)
}

func newProfileDraft(targetKind, adapterKind string, payload any) ProfileDraft {
	raw, _ := artifact.Marshal(payload)
	return ProfileDraft{TargetKind: targetKind, AdapterKind: adapterKind, ProfileVersion: ProfileVersionV1, Payload: raw}
}

type profileState struct {
	digest      artifact.Digest
	document    ProfileDraft
	payload     any
	application appcontrol.Profile
	resolveMS   int64
	bytes       []byte
}

type Profile struct{ state *profileState }

func SealProfile(draft ProfileDraft) (Profile, error) {
	return sealProfileWithRegistry(draft, defaultAdapterRegistry())
}

func sealProfileWithRegistry(draft ProfileDraft, registry adapterRegistry) (Profile, error) {
	if draft.TargetKind == "" || draft.AdapterKind == "" || draft.ProfileVersion == "" || len(draft.Payload) == 0 {
		return Profile{}, errors.New("automation profile envelope is incomplete")
	}
	registered, ok := registry.byKind[draft.AdapterKind]
	if !ok || registered.targetType.TargetKind != draft.TargetKind || registered.targetType.ProfileVersion != draft.ProfileVersion {
		return Profile{}, fmt.Errorf("automation adapter %q does not implement target/profile %q/v%s", draft.AdapterKind, draft.TargetKind, draft.ProfileVersion)
	}
	return registered.seal(draft)
}

func sealDesktopProfile(draft ProfileDraft) (Profile, error) {
	var payload DesktopProfilePayload
	if err := decodeProfilePayload(draft.Payload, &payload); err != nil {
		return Profile{}, fmt.Errorf("decode desktop automation profile v%s: %w", draft.ProfileVersion, err)
	}
	if len(payload.Application.Arguments) != 0 {
		return Profile{}, errors.New("automation target application arguments must be empty")
	}
	application, err := appcontrol.SealProfile(payload.Application)
	if err != nil {
		return Profile{}, fmt.Errorf("seal automation target application configuration: %w", err)
	}
	if !validSelector(payload.WindowTitle) || !validSelector(payload.WindowClass) {
		return Profile{}, errors.New("automation target window selector is invalid")
	}
	if payload.WindowTitleMatch != "exact" && payload.WindowTitleMatch != "regex" {
		return Profile{}, errors.New("automation target window title match mode is invalid")
	}
	if payload.WindowTitleMatch == "regex" {
		if _, err := regexp.Compile(payload.WindowTitle); err != nil {
			return Profile{}, fmt.Errorf("automation target window title regex is invalid: %w", err)
		}
	}
	if payload.WindowSelection != "unique" && payload.WindowSelection != "topmost" {
		return Profile{}, errors.New("automation target window selection policy is invalid")
	}
	if payload.InputBackend != "sendinput" && payload.InputBackend != "postmessage" {
		return Profile{}, errors.New("automation target input backend is invalid")
	}
	if payload.CaptureBackend != "gdi" && payload.CaptureBackend != "wgc" {
		return Profile{}, errors.New("automation target capture backend is invalid")
	}
	if payload.MouseCounts360 < 0 || payload.MouseCounts360 > 10_000_000 {
		return Profile{}, errors.New("automation target mouse calibration is invalid")
	}
	if !validResolveTimeout(payload.ResolveTimeoutMilliseconds) {
		return Profile{}, errors.New("automation target resolve timeout is invalid")
	}
	payload.Application = application.Machine()
	return sealProfileDocument(draft, payload, application, payload.ResolveTimeoutMilliseconds)
}

var adbValuePattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var androidPackagePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z][A-Za-z0-9_]*)+$`)

func sealAndroidProfile(draft ProfileDraft) (Profile, error) {
	var payload AndroidProfilePayload
	if err := decodeProfilePayload(draft.Payload, &payload); err != nil {
		return Profile{}, fmt.Errorf("decode Android ADB profile v%s: %w", draft.ProfileVersion, err)
	}
	if !adbValuePattern.MatchString(payload.ADBSerial) ||
		payload.ADBProduct != "" && !adbValuePattern.MatchString(payload.ADBProduct) ||
		payload.ADBModel != "" && !adbValuePattern.MatchString(payload.ADBModel) ||
		payload.ADBDevice != "" && !adbValuePattern.MatchString(payload.ADBDevice) ||
		!androidPackagePattern.MatchString(payload.AndroidPackage) {
		return Profile{}, errors.New("android ADB configuration is invalid")
	}
	if !validResolveTimeout(payload.ResolveTimeoutMilliseconds) {
		return Profile{}, errors.New("automation target resolve timeout is invalid")
	}
	return sealProfileDocument(draft, payload, appcontrol.Profile{}, payload.ResolveTimeoutMilliseconds)
}

func sealBrowserProfile(draft ProfileDraft) (Profile, error) {
	var payload BrowserProfilePayload
	if err := decodeProfilePayload(draft.Payload, &payload); err != nil {
		return Profile{}, fmt.Errorf("decode Browser CDP profile v%s: %w", draft.ProfileVersion, err)
	}
	if !validResolveTimeout(payload.ResolveTimeoutMilliseconds) {
		return Profile{}, errors.New("automation target resolve timeout is invalid")
	}
	endpoint, err := browsercdp.CanonicalEndpoint(payload.BrowserEndpoint)
	if err != nil {
		return Profile{}, err
	}
	websocketURL, err := browsercdp.ValidateWebSocketURL(payload.BrowserWebSocketURL, endpoint, payload.BrowserTargetID)
	if err != nil {
		return Profile{}, err
	}
	if !validSelector(payload.BrowserTitle) || !validSelector(payload.BrowserURL) {
		return Profile{}, errors.New("browser automation page metadata is invalid")
	}
	payload.BrowserEndpoint, payload.BrowserWebSocketURL = endpoint, websocketURL
	return sealProfileDocument(draft, payload, appcontrol.Profile{}, payload.ResolveTimeoutMilliseconds)
}

func sealProfileDocument(draft ProfileDraft, payload any, application appcontrol.Profile, resolveMS int64) (Profile, error) {
	rawPayload, err := artifact.Marshal(payload)
	if err != nil {
		return Profile{}, err
	}
	draft.Payload = rawPayload
	raw, err := artifact.Marshal(draft)
	if err != nil {
		return Profile{}, err
	}
	digest, err := artifact.Sum(profileDigestDomain, raw)
	if err != nil {
		return Profile{}, err
	}
	return Profile{state: &profileState{digest: digest, document: draft, payload: payload, application: application, resolveMS: resolveMS, bytes: raw}}, nil
}

func decodeProfilePayload(raw []byte, destination any) error {
	if len(raw) == 0 || len(raw) > maxProfilePayloadBytes || destination == nil {
		return errors.New("automation profile payload is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("automation profile payload contains trailing values")
	}
	return nil
}

func OpenProfile(raw []byte, digest artifact.Digest) (Profile, error) {
	if !digest.Valid() || len(raw) == 0 || len(raw) > maxProfilePayloadBytes {
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

func (profile Profile) Valid() bool { return profile.state != nil && profile.state.digest.Valid() }

func (profile Profile) Digest() artifact.Digest {
	if !profile.Valid() {
		return ""
	}
	return profile.state.digest
}

func (profile Profile) Bytes() []byte {
	if !profile.Valid() {
		return nil
	}
	return append([]byte(nil), profile.state.bytes...)
}

func (profile Profile) Machine() ProfileDraft {
	if !profile.Valid() {
		return ProfileDraft{}
	}
	document := profile.state.document
	document.Payload = append([]byte(nil), document.Payload...)
	return document
}

func (profile Profile) Application() appcontrol.Profile {
	if !profile.Valid() {
		return appcontrol.Profile{}
	}
	return profile.state.application
}

func (profile Profile) TargetKind() string {
	if !profile.Valid() {
		return ""
	}
	return profile.state.document.TargetKind
}

func (profile Profile) AdapterKind() string {
	if !profile.Valid() {
		return ""
	}
	return profile.state.document.AdapterKind
}

func (profile Profile) ResolveTimeoutMilliseconds() int64 {
	if !profile.Valid() {
		return 0
	}
	return profile.state.resolveMS
}

func DesktopProfile(profile Profile) (DesktopProfilePayload, bool) {
	if !profile.Valid() {
		return DesktopProfilePayload{}, false
	}
	payload, ok := profile.state.payload.(DesktopProfilePayload)
	if ok {
		payload.Application.Arguments = append([]string(nil), payload.Application.Arguments...)
	}
	return payload, ok
}

func AndroidProfile(profile Profile) (AndroidProfilePayload, bool) {
	if !profile.Valid() {
		return AndroidProfilePayload{}, false
	}
	payload, ok := profile.state.payload.(AndroidProfilePayload)
	return payload, ok
}

func BrowserProfile(profile Profile) (BrowserProfilePayload, bool) {
	if !profile.Valid() {
		return BrowserProfilePayload{}, false
	}
	payload, ok := profile.state.payload.(BrowserProfilePayload)
	return payload, ok
}

func validResolveTimeout(value int64) bool { return value > 0 }

func validSelector(value string) bool {
	if len(value) > 512 || !utf8.ValidString(value) || strings.ContainsRune(value, 0) || value != "" && strings.TrimSpace(value) == "" {
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
