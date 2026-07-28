package ai

import "testing"

func TestModelProfileSealsExactProviderEndpointIntoIdentity(t *testing.T) {
	base := ModelProfileDraft{
		Provider: ProviderOpenAIResponses, Model: "gpt-test", MaxOutputTokens: 128,
		Capabilities: ProfileCapabilities{}, Evaluation: EvaluationUnverified,
	}
	official, err := SealModelProfile(base)
	if err != nil {
		t.Fatal(err)
	}
	if got := official.Machine().Endpoint; got != OpenAIResponsesEndpoint {
		t.Fatalf("default endpoint = %q", got)
	}
	customDraft := base
	customDraft.Endpoint = "https://Gateway.Example/v1/responses"
	custom, err := SealModelProfile(customDraft)
	if err != nil {
		t.Fatal(err)
	}
	if got := custom.Machine().Endpoint; got != "https://gateway.example/v1/responses" {
		t.Fatalf("normalized endpoint = %q", got)
	}
	if custom.Digest() == official.Digest() {
		t.Fatal("endpoint change did not change Model Profile digest")
	}
}

func TestProviderEndpointRejectsAmbientOrAmbiguousNetworkAuthority(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		allowHTTP bool
		valid     bool
	}{
		{name: "https", endpoint: "https://gateway.example/v1/messages", valid: true},
		{name: "loopback explicit", endpoint: "http://127.0.0.1:8080/v1/responses", allowHTTP: true, valid: true},
		{name: "loopback implicit", endpoint: "http://localhost:8080/v1/responses"},
		{name: "remote http", endpoint: "http://192.168.1.4/v1/responses", allowHTTP: true},
		{name: "credentials", endpoint: "https://user:secret@gateway.example/v1/responses"},
		{name: "query", endpoint: "https://gateway.example/v1/responses?token=secret"},
		{name: "fragment", endpoint: "https://gateway.example/v1/responses#route"},
		{name: "provider root", endpoint: "https://gateway.example/", valid: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizeProviderEndpoint(ProviderOpenAIResponses, test.endpoint, test.allowHTTP)
			if (err == nil) != test.valid {
				t.Fatalf("NormalizeProviderEndpoint(%q) error = %v", test.endpoint, err)
			}
		})
	}
}
