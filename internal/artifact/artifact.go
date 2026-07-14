// Package artifact provides the canonical encoding and content identities used
// by durable Yotta contracts.
package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"unicode/utf8"

	jsoncanonicalizer "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

const Algorithm = "sha256"

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var domainPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9/_-]*/v[1-9][0-9]*(?:\.(?:0|[1-9][0-9]*))*$`)
var maxSafeInteger = big.NewInt(9_007_199_254_740_991)

// Digest is the strict textual form of a content identity.
type Digest string

func (d Digest) String() string { return string(d) }
func (d Digest) Valid() bool    { return digestPattern.MatchString(string(d)) }

func ParseDigest(value string) (Digest, error) {
	digest := Digest(value)
	if !digest.Valid() {
		return "", fmt.Errorf("invalid content digest %q", value)
	}
	return digest, nil
}

// Sum hashes canonical bytes with a versioned domain separator.
func Sum(domain string, canonical []byte) (Digest, error) {
	if !domainPattern.MatchString(domain) {
		return "", fmt.Errorf("invalid artifact hash domain %q", domain)
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte(domain))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(canonical)
	return Digest(Algorithm + ":" + hex.EncodeToString(hash.Sum(nil))), nil
}

// Canonicalize transforms one JSON value according to RFC 8785 JCS.
func Canonicalize(raw []byte) ([]byte, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("canonical JSON must be valid UTF-8")
	}
	if err := validateNumbers(raw); err != nil {
		return nil, err
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalize JSON: %w", err)
	}
	return canonical, nil
}

func validateNumbers(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("decode canonical JSON input: %w", err)
		}
		number, ok := token.(json.Number)
		if !ok {
			continue
		}
		floatValue, err := strconv.ParseFloat(number.String(), 64)
		if err != nil || math.IsInf(floatValue, 0) {
			return fmt.Errorf("JSON number %q is outside binary64", number)
		}
		if floatValue == 0 && math.Signbit(floatValue) {
			return fmt.Errorf("JSON number %q is negative zero", number)
		}
		rational, ok := new(big.Rat).SetString(number.String())
		if !ok {
			return fmt.Errorf("invalid JSON number %q", number)
		}
		if floatValue == 0 && rational.Sign() != 0 {
			return fmt.Errorf("JSON number %q underflows binary64", number)
		}
		if math.Trunc(floatValue) == floatValue && math.Abs(floatValue) > float64(maxSafeInteger.Int64()) {
			return fmt.Errorf("JSON number %q rounds outside the interoperable safe integer range", number)
		}
		if rational.IsInt() && new(big.Int).Abs(rational.Num()).Cmp(maxSafeInteger) > 0 {
			return fmt.Errorf("JSON integer %q exceeds the interoperable safe range", number)
		}
	}
}

// Marshal serializes a contract value and returns its RFC 8785 form.
func Marshal(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal contract: %w", err)
	}
	return Canonicalize(raw)
}
