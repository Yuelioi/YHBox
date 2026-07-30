// Package httpegress runs HTTP requests against configured base URLs.
package httpegress

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
)

const profileDigestDomain = "yotta/http-egress-profile/v1"

type ProfileDraft struct {
	Origin              string `json:"origin"`
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
	baseURL, err := canonicalBaseURL(draft.Origin)
	if err != nil {
		return Profile{}, err
	}
	if draft.ResponseByteLimit <= 0 || draft.TimeoutMilliseconds <= 0 {
		return Profile{}, errors.New("HTTP target response limit and timeout must be positive")
	}
	draft.Origin = baseURL
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

func canonicalBaseURL(value string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", errors.New("HTTP egress origin is invalid")
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.Fragment != "" {
		return "", errors.New("configured HTTP base URL must be an absolute http or https URL")
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" || strings.ContainsAny(hostname, "\x00\r\n") {
		return "", errors.New("HTTP egress origin host is invalid")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	port := parsed.Port()
	if parsed.Scheme == "http" && port == "80" || parsed.Scheme == "https" && port == "443" {
		port = ""
	}
	if port != "" {
		parsed.Host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		parsed.Host = "[" + hostname + "]"
	} else {
		parsed.Host = hostname
	}
	if parsed.Path == "/" && parsed.RawQuery == "" {
		parsed.Path = ""
	}
	return parsed.String(), nil
}
