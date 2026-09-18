package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"kun-galgame-api/internal/apiv1"
)

func TestCommittedSpecIsCurrent(t *testing.T) {
	spec, err := apiv1.MarshalOpenAPI(V1Spec())
	if err != nil {
		t.Fatal(err)
	}
	problems, err := apiv1.MarshalProblems()
	if err != nil {
		t.Fatal(err)
	}
	for name, fresh := range map[string][]byte{"kungal-v1.json": spec, "problems.json": problems} {
		committed, err := os.ReadFile(filepath.Join("..", "..", "openapi", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(committed, fresh) {
			t.Errorf("openapi/%s is stale; run make openapi", name)
		}
	}
}
