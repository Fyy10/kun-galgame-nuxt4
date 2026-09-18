package app

import (
	"strings"
	"testing"

	"kun-galgame-api/internal/apiv1/gates"
)

func TestV1Gates(t *testing.T) {
	doc := V1Spec().OpenAPI()
	if errs := gates.CheckAll(doc); len(errs) > 0 {
		t.Fatalf("v1 gates:\n  %s", strings.Join(errs, "\n  "))
	}
}
