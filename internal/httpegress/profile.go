// Package httpegress provides explicitly installed, origin-bound HTTP access
// for workflow nodes. A workflow can select an installation and supply a
// relative path, but it cannot choose a scheme, host, proxy, or redirect.
package httpegress

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	MaxResponseBytes       = 256 << 10
	MaxTimeoutMilliseconds = 60_000
	profileDigestDomain    = "yotta/http-egress-profile/v1"
)

type ProfileDraft struct {
	Origin              string `json:"origin"`
	AllowPrivateNetwork bool   `json:"allowPrivateNetwork"`
	ResponseByteLimit   int64  `json:"responseByteLimit"`
	TimeoutMilliseconds int64  `json:"timeoutMilliseconds"`
}

type profileState struct {
	digest   artifact.Digest
	document ProfileDraft
	bytes    []byte
}

type Profile struct{ state *profileState }

func SealProfile(draft ProfileDraft) (Profile, error) {
	origin, err := canonicalOrigin(draft.Origin)
	if err != nil {
		return Profile{}, err
	}
	if strings.HasPrefix(origin, "http://") && !draft.AllowPrivateNetwork {
		return Profile{}, errors.New("public HTTP egress origins require TLS")
	}
	if draft.ResponseByteLimit <= 0 || draft.ResponseByteLimit > MaxResponseBytes ||
		draft.TimeoutMilliseconds < 100 || draft.TimeoutMilliseconds > MaxTimeoutMilliseconds {
		return Profile{}, errors.New("HTTP egress profile budgets are invalid")
	}
	draft.Origin = origin
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
	if !digest.Valid() || len(raw) == 0 || len(raw) > 16<<10 {
		return Profile{}, errors.New("invalid HTTP egress profile artifact")
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Profile{}, errors.New("HTTP egress profile is not canonical")
	}
	var draft ProfileDraft
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&draft); err != nil {
		return Profile{}, err
	}
	sealed, err := SealProfile(draft)
	if err != nil || sealed.Digest() != digest || !bytes.Equal(sealed.Bytes(), raw) {
		return Profile{}, errors.New("HTTP egress profile digest mismatch")
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
	return p.state.document
}

func canonicalOrigin(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value || len(value) > 2048 {
		return "", errors.New("HTTP egress origin is invalid")
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil ||
		parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", errors.New("HTTP egress origin must contain only scheme and authority")
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" || strings.ContainsAny(hostname, "\x00\r\n") {
		return "", errors.New("HTTP egress origin host is invalid")
	}
	port := parsed.Port()
	if port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return "", errors.New("HTTP egress origin port is invalid")
		}
	}
	if port == "" || parsed.Scheme == "https" && port == "443" || parsed.Scheme == "http" && port == "80" {
		parsed.Host = hostname
		if strings.Contains(hostname, ":") {
			parsed.Host = "[" + hostname + "]"
		}
	} else {
		parsed.Host = net.JoinHostPort(hostname, port)
	}
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.ForceQuery = false
	return parsed.String(), nil
}
