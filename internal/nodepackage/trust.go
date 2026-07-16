package nodepackage

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	TrustPolicyFormat  = "yotta.node-package-trust-policy"
	TrustPolicyVersion = "1"
	SignatureFormat    = "yotta.node-package-signature"
	SignatureVersion   = "1"
	SignatureAlgorithm = "ed25519"

	TrustPublisherSignature = "publisher-signature"

	trustPolicyDigestDomain  = "yotta/node-package-trust-policy/v1"
	publisherKeyDigestDomain = "yotta/node-package-publisher-key/v1"
	signatureDigestDomain    = "yotta/node-package-signature/v1"
	signaturePreimageDomain  = "yotta/node-package-signature-preimage/v1"
	maxTrustPolicyBytes      = 4 << 20
	maxSignatureBytes        = 16 << 10
	maxPublisherAuthorities  = 4_096
	maxPublisherKeys         = 16_384
	maxTrustStatusEntries    = 65_536
	maxSafeRevision          = uint64(9_007_199_254_740_991)
)

type PublisherAuthorityDraft struct {
	Namespace string
	Keys      []ed25519.PublicKey
}

type TrustStatusDraft struct {
	Digest artifact.Digest
	Reason string
}

type TrustPolicyDraft struct {
	Revision             uint64
	PreviousDigest       artifact.Digest
	Publishers           []PublisherAuthorityDraft
	RevokedKeys          []TrustStatusDraft
	RevokedManifests     []TrustStatusDraft
	QuarantinedManifests []TrustStatusDraft
}

type publisherKeyRecord struct {
	KeyID     artifact.Digest `json:"keyId"`
	PublicKey string          `json:"publicKey"`
}

type publisherAuthorityRecord struct {
	Namespace string               `json:"namespace"`
	Keys      []publisherKeyRecord `json:"keys"`
}

type trustStatusRecord struct {
	Digest artifact.Digest `json:"digest"`
	Reason string          `json:"reason"`
}

type trustPolicyDocument struct {
	Format               string                     `json:"format"`
	Version              string                     `json:"version"`
	Revision             uint64                     `json:"revision"`
	PreviousDigest       artifact.Digest            `json:"previousDigest,omitempty"`
	Publishers           []publisherAuthorityRecord `json:"publishers"`
	RevokedKeys          []trustStatusRecord        `json:"revokedKeys"`
	RevokedManifests     []trustStatusRecord        `json:"revokedManifests"`
	QuarantinedManifests []trustStatusRecord        `json:"quarantinedManifests"`
}

type trustPolicyState struct {
	document trustPolicyDocument
	digest   artifact.Digest
	bytes    []byte
	keys     map[artifact.Digest]publisherKey
	status   trustStatus
}

type publisherKey struct {
	namespace string
	publicKey ed25519.PublicKey
}

type trustStatus struct {
	revokedKeys          map[artifact.Digest]string
	revokedManifests     map[artifact.Digest]string
	quarantinedManifests map[artifact.Digest]string
}

type TrustPolicy struct{ state *trustPolicyState }

type SignatureRecord struct {
	Format             string          `json:"format"`
	Version            string          `json:"version"`
	Algorithm          string          `json:"algorithm"`
	PublisherKeyID     artifact.Digest `json:"publisherKeyId"`
	PublisherNamespace string          `json:"publisherNamespace"`
	PackageID          string          `json:"packageId"`
	ManifestDigest     artifact.Digest `json:"manifestDigest"`
	Signature          string          `json:"signature"`
}

type signatureState struct {
	record SignatureRecord
	digest artifact.Digest
	bytes  []byte
}

type SignatureEnvelope struct{ state *signatureState }

type PackageTrust struct {
	Kind           string          `json:"kind"`
	SignerKeyID    artifact.Digest `json:"signerKeyId"`
	EnvelopeDigest artifact.Digest `json:"envelopeDigest"`
	Envelope       SignatureRecord `json:"envelope"`
}

func PublisherKeyID(publicKey ed25519.PublicKey) (artifact.Digest, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return "", errors.New("node package publisher public key is invalid")
	}
	return artifact.Sum(publisherKeyDigestDomain, publicKey)
}

func SealTrustPolicy(draft TrustPolicyDraft) (TrustPolicy, error) {
	if draft.Revision == 0 || draft.Revision > maxSafeRevision {
		return TrustPolicy{}, errors.New("node package trust policy revision is invalid")
	}
	if draft.Revision == 1 && draft.PreviousDigest.Valid() || draft.Revision > 1 && !draft.PreviousDigest.Valid() {
		return TrustPolicy{}, errors.New("node package trust policy predecessor is invalid")
	}
	if len(draft.Publishers) == 0 || len(draft.Publishers) > maxPublisherAuthorities {
		return TrustPolicy{}, errors.New("node package publisher authority count is invalid")
	}
	publishers := make([]publisherAuthorityRecord, 0, len(draft.Publishers))
	totalKeys := 0
	for _, authority := range draft.Publishers {
		if _, err := validateNamespace(authority.Namespace); err != nil {
			return TrustPolicy{}, fmt.Errorf("publisher authority namespace: %w", err)
		}
		if len(authority.Keys) == 0 {
			return TrustPolicy{}, errors.New("node package publisher authority has no keys")
		}
		keys := make([]publisherKeyRecord, 0, len(authority.Keys))
		for _, publicKey := range authority.Keys {
			keyID, err := PublisherKeyID(publicKey)
			if err != nil {
				return TrustPolicy{}, err
			}
			keys = append(keys, publisherKeyRecord{KeyID: keyID, PublicKey: base64.RawStdEncoding.EncodeToString(publicKey)})
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i].KeyID < keys[j].KeyID })
		for index := 1; index < len(keys); index++ {
			if keys[index].KeyID == keys[index-1].KeyID {
				return TrustPolicy{}, errors.New("node package publisher authority contains a duplicate key")
			}
		}
		totalKeys += len(keys)
		publishers = append(publishers, publisherAuthorityRecord{Namespace: authority.Namespace, Keys: keys})
	}
	if totalKeys > maxPublisherKeys {
		return TrustPolicy{}, errors.New("node package publisher key count exceeds its budget")
	}
	sort.Slice(publishers, func(i, j int) bool { return publishers[i].Namespace < publishers[j].Namespace })
	for index := 1; index < len(publishers); index++ {
		if publishers[index].Namespace == publishers[index-1].Namespace {
			return TrustPolicy{}, errors.New("node package publisher namespace has conflicting authorities")
		}
	}
	revokedKeys, err := normalizeTrustStatus(draft.RevokedKeys)
	if err != nil {
		return TrustPolicy{}, fmt.Errorf("revoked publisher keys: %w", err)
	}
	revokedManifests, err := normalizeTrustStatus(draft.RevokedManifests)
	if err != nil {
		return TrustPolicy{}, fmt.Errorf("revoked manifests: %w", err)
	}
	quarantinedManifests, err := normalizeTrustStatus(draft.QuarantinedManifests)
	if err != nil {
		return TrustPolicy{}, fmt.Errorf("quarantined manifests: %w", err)
	}
	if len(revokedKeys)+len(revokedManifests)+len(quarantinedManifests) > maxTrustStatusEntries {
		return TrustPolicy{}, errors.New("node package trust status count exceeds its budget")
	}
	document := trustPolicyDocument{
		Format: TrustPolicyFormat, Version: TrustPolicyVersion, Revision: draft.Revision,
		PreviousDigest: draft.PreviousDigest, Publishers: publishers, RevokedKeys: revokedKeys,
		RevokedManifests: revokedManifests, QuarantinedManifests: quarantinedManifests,
	}
	return sealTrustPolicyDocument(document)
}

func OpenTrustPolicy(raw []byte) (TrustPolicy, error) {
	if len(raw) == 0 || len(raw) > maxTrustPolicyBytes {
		return TrustPolicy{}, errors.New("node package trust policy exceeds its byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, 16, 262_144, maxTrustPolicyBytes); err != nil {
		return TrustPolicy{}, fmt.Errorf("inspect node package trust policy: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return TrustPolicy{}, errors.New("node package trust policy is not canonical JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document trustPolicyDocument
	if err := decoder.Decode(&document); err != nil {
		return TrustPolicy{}, fmt.Errorf("decode node package trust policy: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return TrustPolicy{}, errors.New("node package trust policy contains trailing JSON values")
	}
	if document.Format != TrustPolicyFormat || document.Version != TrustPolicyVersion {
		return TrustPolicy{}, errors.New("unsupported node package trust policy format")
	}
	draft := TrustPolicyDraft{
		Revision: document.Revision, PreviousDigest: document.PreviousDigest,
		RevokedKeys: trustStatusDrafts(document.RevokedKeys), RevokedManifests: trustStatusDrafts(document.RevokedManifests),
		QuarantinedManifests: trustStatusDrafts(document.QuarantinedManifests),
	}
	for _, authority := range document.Publishers {
		draftAuthority := PublisherAuthorityDraft{Namespace: authority.Namespace}
		for _, key := range authority.Keys {
			publicKey, err := base64.RawStdEncoding.DecodeString(key.PublicKey)
			if err != nil {
				return TrustPolicy{}, errors.New("node package trust policy contains an invalid publisher key")
			}
			computed, err := PublisherKeyID(ed25519.PublicKey(publicKey))
			if err != nil || computed != key.KeyID {
				return TrustPolicy{}, errors.New("node package trust policy publisher key identity is invalid")
			}
			draftAuthority.Keys = append(draftAuthority.Keys, ed25519.PublicKey(publicKey))
		}
		draft.Publishers = append(draft.Publishers, draftAuthority)
	}
	sealed, err := SealTrustPolicy(draft)
	if err != nil {
		return TrustPolicy{}, err
	}
	if !bytes.Equal(sealed.Bytes(), raw) {
		return TrustPolicy{}, errors.New("node package trust policy is not normalized")
	}
	return sealed, nil
}

func sealTrustPolicyDocument(document trustPolicyDocument) (TrustPolicy, error) {
	raw, err := artifact.Marshal(document)
	if err != nil {
		return TrustPolicy{}, err
	}
	if len(raw) > maxTrustPolicyBytes {
		return TrustPolicy{}, errors.New("node package trust policy exceeds its byte budget")
	}
	digest, err := artifact.Sum(trustPolicyDigestDomain, raw)
	if err != nil {
		return TrustPolicy{}, err
	}
	keys := make(map[artifact.Digest]publisherKey)
	for _, authority := range document.Publishers {
		for _, record := range authority.Keys {
			publicKey, _ := base64.RawStdEncoding.DecodeString(record.PublicKey)
			if existing, found := keys[record.KeyID]; found && existing.namespace != authority.Namespace {
				return TrustPolicy{}, errors.New("node package publisher key owns conflicting namespaces")
			}
			keys[record.KeyID] = publisherKey{namespace: authority.Namespace, publicKey: ed25519.PublicKey(publicKey)}
		}
	}
	status := trustStatus{
		revokedKeys: statusMap(document.RevokedKeys), revokedManifests: statusMap(document.RevokedManifests),
		quarantinedManifests: statusMap(document.QuarantinedManifests),
	}
	for keyID := range status.revokedKeys {
		if _, found := keys[keyID]; !found {
			return TrustPolicy{}, errors.New("node package trust policy revokes an unknown publisher key")
		}
	}
	return TrustPolicy{state: &trustPolicyState{document: document, digest: digest, bytes: raw, keys: keys, status: status}}, nil
}

func (p TrustPolicy) Valid() bool { return p.state != nil && p.state.digest.Valid() }
func (p TrustPolicy) Digest() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.digest
}
func (p TrustPolicy) Bytes() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.state.bytes...)
}
func (p TrustPolicy) Revision() uint64 {
	if !p.Valid() {
		return 0
	}
	return p.state.document.Revision
}

func SignManifest(manifest Manifest, privateKey ed25519.PrivateKey) (SignatureEnvelope, error) {
	if !manifest.Valid() || len(privateKey) != ed25519.PrivateKeySize {
		return SignatureEnvelope{}, errors.New("node package signature input is invalid")
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	keyID, err := PublisherKeyID(publicKey)
	if err != nil {
		return SignatureEnvelope{}, err
	}
	record := SignatureRecord{
		Format: SignatureFormat, Version: SignatureVersion, Algorithm: SignatureAlgorithm,
		PublisherKeyID: keyID, PublisherNamespace: manifest.PublisherNamespace(), PackageID: manifest.PackageID(),
		ManifestDigest: manifest.Digest(),
	}
	preimage, err := signaturePreimage(record)
	if err != nil {
		return SignatureEnvelope{}, err
	}
	record.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, preimage))
	return sealSignatureRecord(record)
}

func OpenSignatureEnvelope(raw []byte) (SignatureEnvelope, error) {
	if len(raw) == 0 || len(raw) > maxSignatureBytes {
		return SignatureEnvelope{}, errors.New("node package signature envelope exceeds its byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, 4, 64, maxSignatureBytes); err != nil {
		return SignatureEnvelope{}, fmt.Errorf("inspect node package signature envelope: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return SignatureEnvelope{}, errors.New("node package signature envelope is not canonical JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var record SignatureRecord
	if err := decoder.Decode(&record); err != nil {
		return SignatureEnvelope{}, fmt.Errorf("decode node package signature envelope: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return SignatureEnvelope{}, errors.New("node package signature envelope contains trailing JSON values")
	}
	sealed, err := sealSignatureRecord(record)
	if err != nil {
		return SignatureEnvelope{}, err
	}
	if !bytes.Equal(sealed.Bytes(), raw) {
		return SignatureEnvelope{}, errors.New("node package signature envelope is not normalized")
	}
	return sealed, nil
}

func sealSignatureRecord(record SignatureRecord) (SignatureEnvelope, error) {
	if record.Format != SignatureFormat || record.Version != SignatureVersion || record.Algorithm != SignatureAlgorithm ||
		!record.PublisherKeyID.Valid() || !record.ManifestDigest.Valid() || record.PackageID == "" {
		return SignatureEnvelope{}, errors.New("node package signature envelope identity is invalid")
	}
	if _, err := validateNamespace(record.PublisherNamespace); err != nil {
		return SignatureEnvelope{}, err
	}
	signature, err := base64.RawStdEncoding.DecodeString(record.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || base64.RawStdEncoding.EncodeToString(signature) != record.Signature {
		return SignatureEnvelope{}, errors.New("node package signature encoding is invalid")
	}
	raw, err := artifact.Marshal(record)
	if err != nil {
		return SignatureEnvelope{}, err
	}
	digest, err := artifact.Sum(signatureDigestDomain, raw)
	if err != nil {
		return SignatureEnvelope{}, err
	}
	return SignatureEnvelope{state: &signatureState{record: record, digest: digest, bytes: raw}}, nil
}

func (e SignatureEnvelope) Valid() bool { return e.state != nil && e.state.digest.Valid() }
func (e SignatureEnvelope) Digest() artifact.Digest {
	if !e.Valid() {
		return ""
	}
	return e.state.digest
}
func (e SignatureEnvelope) Bytes() []byte {
	if !e.Valid() {
		return nil
	}
	return append([]byte(nil), e.state.bytes...)
}

func VerifySignature(manifest Manifest, envelope SignatureEnvelope, policy TrustPolicy) (PackageTrust, error) {
	if !manifest.Valid() || !envelope.Valid() || !policy.Valid() {
		return PackageTrust{}, errors.New("node package signature verification input is invalid")
	}
	record := envelope.state.record
	if record.ManifestDigest != manifest.Digest() || record.PackageID != manifest.PackageID() || record.PublisherNamespace != manifest.PublisherNamespace() {
		return PackageTrust{}, errors.New("node package signature envelope does not match the manifest")
	}
	key, found := policy.state.keys[record.PublisherKeyID]
	if !found {
		return PackageTrust{}, errors.New("node package signature uses an unknown publisher key")
	}
	if key.namespace != manifest.PublisherNamespace() {
		return PackageTrust{}, errors.New("node package publisher key does not own the manifest namespace")
	}
	if reason, blocked := policy.blockReason(record.PublisherKeyID, manifest.Digest()); blocked {
		return PackageTrust{}, fmt.Errorf("node package trust policy blocks the package: %s", reason)
	}
	preimage, err := signaturePreimage(record)
	if err != nil {
		return PackageTrust{}, err
	}
	signature, _ := base64.RawStdEncoding.DecodeString(record.Signature)
	if !ed25519.Verify(key.publicKey, preimage, signature) {
		return PackageTrust{}, errors.New("node package signature verification failed")
	}
	return PackageTrust{Kind: TrustPublisherSignature, SignerKeyID: record.PublisherKeyID, EnvelopeDigest: envelope.Digest(), Envelope: record}, nil
}

func (p TrustPolicy) verifyStored(manifest Manifest, trust PackageTrust) error {
	if trust.Kind != TrustPublisherSignature || trust.SignerKeyID != trust.Envelope.PublisherKeyID {
		return errors.New("node package stored signature trust is invalid")
	}
	envelope, err := sealSignatureRecord(trust.Envelope)
	if err != nil || envelope.Digest() != trust.EnvelopeDigest {
		return errors.New("node package stored signature envelope is invalid")
	}
	_, err = verifySignatureCryptographic(manifest, envelope, p)
	return err
}

func verifySignatureCryptographic(manifest Manifest, envelope SignatureEnvelope, policy TrustPolicy) (publisherKey, error) {
	if !manifest.Valid() || !envelope.Valid() || !policy.Valid() {
		return publisherKey{}, errors.New("node package signature verification input is invalid")
	}
	record := envelope.state.record
	if record.ManifestDigest != manifest.Digest() || record.PackageID != manifest.PackageID() || record.PublisherNamespace != manifest.PublisherNamespace() {
		return publisherKey{}, errors.New("node package signature envelope does not match the manifest")
	}
	key, found := policy.state.keys[record.PublisherKeyID]
	if !found || key.namespace != manifest.PublisherNamespace() {
		return publisherKey{}, errors.New("node package stored publisher authority is invalid")
	}
	preimage, err := signaturePreimage(record)
	if err != nil {
		return publisherKey{}, err
	}
	signature, _ := base64.RawStdEncoding.DecodeString(record.Signature)
	if !ed25519.Verify(key.publicKey, preimage, signature) {
		return publisherKey{}, errors.New("node package signature verification failed")
	}
	return key, nil
}

func signaturePreimage(record SignatureRecord) ([]byte, error) {
	preimage := struct {
		Format             string          `json:"format"`
		Version            string          `json:"version"`
		Algorithm          string          `json:"algorithm"`
		PublisherKeyID     artifact.Digest `json:"publisherKeyId"`
		PublisherNamespace string          `json:"publisherNamespace"`
		PackageID          string          `json:"packageId"`
		ManifestDigest     artifact.Digest `json:"manifestDigest"`
	}{record.Format, record.Version, record.Algorithm, record.PublisherKeyID, record.PublisherNamespace, record.PackageID, record.ManifestDigest}
	canonical, err := artifact.Marshal(preimage)
	if err != nil {
		return nil, err
	}
	message := make([]byte, 0, len(signaturePreimageDomain)+1+len(canonical))
	message = append(message, signaturePreimageDomain...)
	message = append(message, 0)
	message = append(message, canonical...)
	return message, nil
}

func (p TrustPolicy) blockReason(keyID, manifestDigest artifact.Digest) (string, bool) {
	if reason, found := p.state.status.revokedKeys[keyID]; found {
		return reason, true
	}
	if reason, found := p.state.status.revokedManifests[manifestDigest]; found {
		return reason, true
	}
	if reason, found := p.state.status.quarantinedManifests[manifestDigest]; found {
		return reason, true
	}
	return "", false
}

func validateTrustTransition(previous, next TrustPolicy) error {
	if !previous.Valid() || !next.Valid() || next.Revision() != previous.Revision()+1 || next.state.document.PreviousDigest != previous.Digest() {
		return errors.New("node package trust policy transition does not extend the current policy")
	}
	for keyID, key := range previous.state.keys {
		nextKey, found := next.state.keys[keyID]
		if !found || nextKey.namespace != key.namespace || !bytes.Equal(nextKey.publicKey, key.publicKey) {
			return errors.New("node package trust policy cannot remove or reassign publisher authority")
		}
	}
	for _, pair := range [][2]map[artifact.Digest]string{
		{previous.state.status.revokedKeys, next.state.status.revokedKeys},
		{previous.state.status.revokedManifests, next.state.status.revokedManifests},
		{previous.state.status.quarantinedManifests, next.state.status.quarantinedManifests},
	} {
		for digest, reason := range pair[0] {
			if pair[1][digest] != reason {
				return errors.New("node package trust policy cannot remove or rewrite a trust status")
			}
		}
	}
	return nil
}

func normalizeTrustStatus(source []TrustStatusDraft) ([]trustStatusRecord, error) {
	records := make([]trustStatusRecord, 0, len(source))
	for _, status := range source {
		if !status.Digest.Valid() || !quarantineReasonPattern.MatchString(status.Reason) {
			return nil, errors.New("trust status identity or reason is invalid")
		}
		records = append(records, trustStatusRecord(status))
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Digest < records[j].Digest })
	for index := 1; index < len(records); index++ {
		if records[index].Digest == records[index-1].Digest {
			return nil, errors.New("trust status contains a duplicate digest")
		}
	}
	return records, nil
}

func trustStatusDrafts(source []trustStatusRecord) []TrustStatusDraft {
	drafts := make([]TrustStatusDraft, 0, len(source))
	for _, record := range source {
		drafts = append(drafts, TrustStatusDraft(record))
	}
	return drafts
}

func statusMap(source []trustStatusRecord) map[artifact.Digest]string {
	result := make(map[artifact.Digest]string, len(source))
	for _, status := range source {
		result[status.Digest] = status.Reason
	}
	return result
}
