// Package appcontrol provides explicitly installed desktop application lifecycle
// authority. It is not a generic process runner or plugin host.
package appcontrol

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MaxExecutableBytes  = int64(1 << 30)
	MaxArguments        = 64
	MaxArgumentBytes    = 4096
	profileDigestDomain = "yotta/installed-application-profile/v1"
)

var deniedEntrypoints = map[string]struct{}{
	"cmd.exe": {}, "powershell.exe": {}, "pwsh.exe": {}, "wscript.exe": {},
	"cscript.exe": {}, "mshta.exe": {}, "rundll32.exe": {}, "regsvr32.exe": {},
}

type ProfileDraft struct {
	Executable string   `json:"executable"`
	Arguments  []string `json:"arguments"`
}

type ExecutableInspection struct {
	Executable string          `json:"executable"`
	Digest     artifact.Digest `json:"digest"`
	Size       int64           `json:"size"`
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
	if len(draft.Arguments) > MaxArguments {
		return Profile{}, errors.New("installed application argument list exceeds budget")
	}
	arguments := append([]string(nil), draft.Arguments...)
	if arguments == nil {
		arguments = []string{}
	}
	for _, argument := range arguments {
		if len(argument) > MaxArgumentBytes || !utf8.ValidString(argument) || strings.ContainsRune(argument, 0) {
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

func InspectExecutable(path string) (ExecutableInspection, error) {
	normalized, err := normalizeExecutablePath(path)
	if err != nil {
		return ExecutableInspection{}, err
	}
	file, err := os.Open(normalized)
	if err != nil {
		return ExecutableInspection{}, fmt.Errorf("open installed application executable: %w", err)
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > MaxExecutableBytes {
		return ExecutableInspection{}, errors.New("installed application executable is not a bounded regular file")
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, MaxExecutableBytes+1))
	if err != nil {
		return ExecutableInspection{}, err
	}
	after, err := file.Stat()
	if err != nil || written != before.Size() || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return ExecutableInspection{}, errors.New("installed application executable changed while hashing")
	}
	digest, err := artifact.ParseDigest("sha256:" + hex.EncodeToString(hash.Sum(nil)))
	if err != nil {
		return ExecutableInspection{}, err
	}
	return ExecutableInspection{Executable: normalized, Digest: digest, Size: written}, nil
}

func VerifyProfile(profile Profile) error {
	if !profile.Valid() {
		return errors.New("installed application profile is invalid")
	}
	inspection, err := InspectExecutable(profile.Machine().Executable)
	if err != nil {
		return err
	}
	hostExecutable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve Yotta host executable: %w", err)
	}
	hostInfo, err := os.Stat(hostExecutable)
	if err != nil {
		return fmt.Errorf("inspect Yotta host executable: %w", err)
	}
	configuredInfo, err := os.Stat(inspection.Executable)
	if err != nil {
		return fmt.Errorf("inspect installed application executable: %w", err)
	}
	if os.SameFile(hostInfo, configuredInfo) {
		return errors.New("yotta host executable cannot be installed as a workflow application")
	}
	return nil
}

func normalizeExecutablePath(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || len(value) > 32_767 || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return "", errors.New("installed application executable path is invalid")
	}
	cleaned := filepath.Clean(value)
	if !filepath.IsAbs(cleaned) || !strings.EqualFold(filepath.Ext(cleaned), ".exe") {
		return "", errors.New("installed application executable must be an absolute .exe path")
	}
	if _, denied := deniedEntrypoints[strings.ToLower(filepath.Base(cleaned))]; denied {
		return "", errors.New("shell and script host executables cannot be installed as applications")
	}
	return cleaned, nil
}
