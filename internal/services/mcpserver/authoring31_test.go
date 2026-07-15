package mcpserver

import (
	"bytes"
	"testing"
)

func TestCatalog31SearchAndDescribePreserveConcatChannels(t *testing.T) {
	search := searchCatalog31JSON("concat")
	if !bytes.Contains(search, []byte(`"version": "3.1"`)) || !bytes.Contains(search, []byte(`"executionClass": "pure-data"`)) ||
		!bytes.Contains(search, []byte(`"projectionDigest": "sha256:`)) || !bytes.Contains(search, []byte(`"generatorVersion": "v1"`)) {
		t.Fatalf("search = %s", search)
	}
	description := describeCatalog31JSON("https://schemas.yotta.dev/nodes/text/concat/v1")
	if !bytes.Contains(description, []byte(`"dataInputs": [`)) || !bytes.Contains(description, []byte(`"signals": []`)) {
		t.Fatalf("description = %s", description)
	}
	if bytes.Contains(description, []byte(`"id": "out"`)) {
		t.Fatalf("description invented out: %s", description)
	}
	builtins, err := builtinCatalog31()
	if err != nil {
		t.Fatal(err)
	}
	all := searchCatalog31JSON("")
	if got, want := bytes.Count(all, []byte(`"nodeTypeId":`)), len(builtins.Contracts); got != want {
		t.Fatalf("search returned %d of %d built-in projections: %s", got, want, all)
	}
}
