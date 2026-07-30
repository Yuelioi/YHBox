// Package appcontrol provides configured desktop application lifecycle targets.
package appcontrol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	profileDigestDomain = "yotta/installed-application-profile/v1"
)

type ProfileDraft struct {
	Executable string   `json:"executable"`
	Arguments  []string `json:"arguments"`
}

type profileState struct {
	digest   artifact.Digest
	document ProfileDraft
	bytes    []byte
}

type Profile struct{ state *profileState }

func SealProfile(draft ProfileDraft) (Profile, error) {
	path, err := normalizeExecutablePath(draft.Executable)
	if err != nil {
		return Profile{}, err
	}
	arguments := append([]string(nil), draft.Arguments...)
	if arguments == nil {
		arguments = []string{}
	}
	for _, argument := range arguments {
		if !utf8.ValidString(argument) || strings.ContainsRune(argument, 0) {
			return Profile{}, errors.New("installed application argument is invalid")
		}
	}
	draft.Executable, draft.Arguments = path, arguments
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
		return Profile{}, errors.New("invalid installed application profile artifact")
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Profile{}, errors.New("installed application profile is not canonical")
	}
	var draft ProfileDraft
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&draft); err != nil {
		return Profile{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Profile{}, errors.New("installed application profile contains trailing values")
	}
	sealed, err := SealProfile(draft)
	if err != nil || sealed.Digest() != digest || !bytes.Equal(sealed.Bytes(), raw) {
		return Profile{}, errors.New("installed application profile digest mismatch")
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
	document.Arguments = append([]string(nil), document.Arguments...)
	return document
}

func normalizeExecutablePath(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return "", errors.New("installed application executable path is invalid")
	}
	cleaned := filepath.Clean(value)
	if !filepath.IsAbs(cleaned) {
		return "", errors.New("configured application path must be absolute")
	}
	return cleaned, nil
}
