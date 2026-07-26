package nodepackage

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestTrustPolicyIsCanonicalStrictAndRejectsNamespaceConflict(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := SealTrustPolicy(TrustPolicyDraft{
		Revision:   1,
		Publishers: []PublisherAuthorityDraft{{Namespace: testNamespace, Keys: []ed25519.PublicKey{publicKey}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenTrustPolicy(policy.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Digest() != policy.Digest() || !bytes.Equal(reopened.Bytes(), policy.Bytes()) {
		t.Fatal("reopened trust policy changed identity")
	}
	var document map[string]any
	if err := json.Unmarshal(policy.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	document["unexpected"] = true
	tampered, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenTrustPolicy(tampered); err == nil {
		t.Fatal("trust policy accepted an unknown field")
	}
	if _, err := SealTrustPolicy(TrustPolicyDraft{
		Revision: 1,
		Publishers: []PublisherAuthorityDraft{
			{Namespace: testNamespace, Keys: []ed25519.PublicKey{publicKey}},
			{Namespace: testNamespace, Keys: []ed25519.PublicKey{publicKey}},
		},
	}); err == nil {
		t.Fatal("trust policy accepted conflicting namespace authorities")
	}
}

func TestSignatureVerificationRejectsTamperUnknownKeyAndNamespaceHijack(t *testing.T) {
	manifest, _ := lifecycleArchiveManifest(t, "1.0.0", "process-v1")
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	policy := trustPolicyForKey(t, 1, "", testNamespace, publicKey, nil, nil)
	envelope, err := SignManifest(manifest, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySignature(manifest, envelope, policy); err != nil {
		t.Fatal(err)
	}

	tamperedManifest, _ := lifecycleArchiveManifest(t, "2.0.0", "process-v2")
	if _, err := VerifySignature(tamperedManifest, envelope, policy); err == nil {
		t.Fatal("signature verified a different manifest")
	}
	tamperedRecord := envelope.state.record
	tamperedSignature, err := base64.RawStdEncoding.DecodeString(tamperedRecord.Signature)
	if err != nil {
		t.Fatal(err)
	}
	tamperedSignature[0] ^= 1
	tamperedRecord.Signature = base64.RawStdEncoding.EncodeToString(tamperedSignature)
	tamperedEnvelope, err := sealSignatureRecord(tamperedRecord)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifySignature(manifest, tamperedEnvelope, policy); err == nil {
		t.Fatal("tampered signature verified")
	}

	unknownPublicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	unknownPolicy := trustPolicyForKey(t, 1, "", testNamespace, unknownPublicKey, nil, nil)
	if _, err := VerifySignature(manifest, envelope, unknownPolicy); err == nil {
		t.Fatal("unknown publisher key verified")
	}
	hijackedPolicy := trustPolicyForKey(t, 1, "", "https://packages.example.test/other", publicKey, nil, nil)
	if _, err := VerifySignature(manifest, envelope, hijackedPolicy); err == nil {
		t.Fatal("publisher key verified outside its owned namespace")
	}
}

func TestStoreRevocationQuarantineRollbackAndReopenFailClosed(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "packages")
	policy, privateKey := lifecyclePolicy(t)
	store, err := CreateStore(ctx, root, policy)
	if err != nil {
		t.Fatal(err)
	}
	firstManifest, firstArchive := lifecycleArchive(t, privateKey, "1.0.0", "process-v1")
	secondManifest, secondArchive := lifecycleArchive(t, privateKey, "2.0.0", "process-v2")
	grantArchive(t, ctx, store, firstArchive)
	if _, err := store.InstallArchive(ctx, firstArchive); err != nil {
		t.Fatal(err)
	}
	if _, err := store.InstallArchive(ctx, secondArchive); err != nil {
		t.Fatal(err)
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	quarantine := trustPolicyForKey(t, 2, policy.Digest(), testNamespace, publicKey, nil, []TrustStatusDraft{{Digest: secondManifest.Digest(), Reason: "security.quarantine"}})
	if err := store.ApplyTrustPolicy(ctx, quarantine); err != nil {
		t.Fatal(err)
	}
	blocked, _ := store.Get(firstManifest.PackageID())
	if blocked.Enabled || blocked.Releases[releaseIndex(blocked.Releases, secondManifest.Digest())].QuarantineReason != "security.quarantine" {
		t.Fatalf("quarantined installation = %#v", blocked)
	}
	rolledBack, err := store.Rollback(ctx, firstManifest.PackageID())
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.Current != firstManifest.Digest() || rolledBack.Enabled {
		t.Fatalf("rollback after quarantine = %#v", rolledBack)
	}
	keyID, err := PublisherKeyID(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	revoked := trustPolicyForKey(t, 3, quarantine.Digest(), testNamespace, publicKey,
		[]TrustStatusDraft{{Digest: keyID, Reason: "security.key-revoked"}},
		[]TrustStatusDraft{{Digest: secondManifest.Digest(), Reason: "security.quarantine"}},
	)
	if err := store.ApplyTrustPolicy(ctx, revoked); err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyTrustPolicy(ctx, policy); err == nil {
		t.Fatal("store accepted a trust policy rollback")
	}
	if _, err := store.Enable(firstManifest.PackageID()); err == nil {
		t.Fatal("store enabled a release signed by a revoked key")
	}
	if _, err := store.Rollback(ctx, firstManifest.PackageID()); err == nil {
		t.Fatal("store rolled back to a release signed by a revoked key")
	}
	reopened, err := OpenStore(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	installed, found := reopened.Get(firstManifest.PackageID())
	if !found || installed.Enabled || reopened.TrustPolicy().Digest() != revoked.Digest() {
		t.Fatalf("reopened revoked installation = %#v, found=%v", installed, found)
	}
	for _, release := range installed.Releases {
		if release.QuarantineReason == "" {
			t.Fatalf("reopened release was not quarantined: %#v", release)
		}
	}
}

func TestStoreRejectsUnsignedArchive(t *testing.T) {
	ctx := context.Background()
	policy, _ := lifecyclePolicy(t)
	store, err := CreateStore(ctx, filepath.Join(t.TempDir(), "packages"), policy)
	if err != nil {
		t.Fatal(err)
	}
	manifest, payload := lifecycleArchiveManifest(t, "1.0.0", "process-v1")
	archivePath := writeArchive(t, []archiveTestEntry{
		{name: ArchiveManifestPath, data: manifest.Bytes()},
		{name: "bin/plugin.exe", data: payload},
	})
	if _, err := store.InstallArchive(ctx, archivePath); err == nil {
		t.Fatal("store installed an unsigned archive")
	}
}

func trustPolicyForKey(t *testing.T, revision uint64, previousDigest artifact.Digest, namespace string, publicKey ed25519.PublicKey, revokedKeys, quarantined []TrustStatusDraft) TrustPolicy {
	t.Helper()
	policy, err := SealTrustPolicy(TrustPolicyDraft{
		Revision: revision, PreviousDigest: previousDigest,
		Publishers:  []PublisherAuthorityDraft{{Namespace: namespace, Keys: []ed25519.PublicKey{publicKey}}},
		RevokedKeys: revokedKeys, QuarantinedManifests: quarantined,
	})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func lifecycleArchiveManifest(t *testing.T, version, payload string) (Manifest, []byte) {
	t.Helper()
	draft := testDraft(t, nodecontract.ABIProcess)
	draft.PackageVersion = version
	payloadBytes := []byte(payload)
	draft.Nodes[0].Implementation.Payload = testPayload(t, "bin/plugin.exe", "application/vnd.microsoft.portable-executable", payload)
	manifest, err := Seal(draft)
	if err != nil {
		t.Fatal(err)
	}
	return manifest, payloadBytes
}
