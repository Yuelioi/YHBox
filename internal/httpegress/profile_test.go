package httpegress

import (
	"bytes"
	"testing"
)

func TestProfileCanonicalizesOriginAndRoundTrips(t *testing.T) {
	profile, err := SealProfile(ProfileDraft{Origin: "https://EXAMPLE.com:443/", ResponseByteLimit: 4096, TimeoutMilliseconds: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if got := profile.Machine().Origin; got != "https://example.com" {
		t.Fatalf("origin = %q", got)
	}
	opened, err := OpenProfile(profile.Bytes(), profile.Digest())
	if err != nil || !bytes.Equal(opened.Bytes(), profile.Bytes()) {
		t.Fatalf("round trip failed: %v", err)
	}
}

func TestProfileRejectsNonOriginAndInvalidBudgets(t *testing.T) {
	cases := []ProfileDraft{
		{Origin: "https://user@example.com", ResponseByteLimit: 1, TimeoutMilliseconds: 100},
		{Origin: "https://example.com/api", ResponseByteLimit: 1, TimeoutMilliseconds: 100},
		{Origin: "ftp://example.com", ResponseByteLimit: 1, TimeoutMilliseconds: 100},
		{Origin: "https://example.com", ResponseByteLimit: MaxResponseBytes + 1, TimeoutMilliseconds: 100},
		{Origin: "https://example.com", ResponseByteLimit: 1, TimeoutMilliseconds: 99},
		{Origin: "http://example.com", ResponseByteLimit: 1, TimeoutMilliseconds: 100},
	}
	for _, draft := range cases {
		if _, err := SealProfile(draft); err == nil {
			t.Fatalf("accepted invalid profile: %+v", draft)
		}
	}
}

func TestProfileCanonicalizesIPv6Authority(t *testing.T) {
	profile, err := SealProfile(ProfileDraft{Origin: "http://[::1]:80", AllowPrivateNetwork: true, ResponseByteLimit: 1, TimeoutMilliseconds: 100})
	if err != nil {
		t.Fatal(err)
	}
	if got := profile.Machine().Origin; got != "http://[::1]" {
		t.Fatalf("origin = %q", got)
	}
}

func TestConfiguredInstallationsImmediatelyBindSlots(t *testing.T) {
	draft := ProfileDraft{Origin: "https://example.com", ResponseByteLimit: 4096, TimeoutMilliseconds: 1000}
	installed, err := Install([]InstallationDraft{{Slot: "primary", Profile: draft}, {Slot: "secondary", Profile: draft}})
	if err != nil {
		t.Fatal(err)
	}
	entries := installed.Entries()
	if len(entries) != 2 || entries[0].TargetID != "http-origin/primary" || entries[0].ProviderID != entries[1].ProviderID || entries[0].Provider != entries[1].Provider {
		t.Fatalf("unexpected installations: %+v", entries)
	}
}
