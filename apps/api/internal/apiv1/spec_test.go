package apiv1

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestSpecDeterministicAndShape(t *testing.T) {
	app := fiber.New()
	api := Setup(app, Deps{})
	a, err := MarshalOpenAPI(api)
	if err != nil {
		t.Fatal(err)
	}
	b, err := MarshalOpenAPI(api)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("MarshalOpenAPI is not deterministic")
	}
	p1, err := MarshalProblems()
	if err != nil {
		t.Fatal(err)
	}
	p2, err := MarshalProblems()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p1, p2) {
		t.Fatal("MarshalProblems is not deterministic")
	}

	var doc map[string]any
	if err := json.Unmarshal(a, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Errorf("openapi %v", doc["openapi"])
	}
	info, _ := doc["info"].(map[string]any)
	if info["title"] != infoTitle || info["version"] != infoVersion {
		t.Errorf("info %+v", info)
	}
	if info["x-stability"] != "preview" {
		t.Errorf("x-stability %v", info["x-stability"])
	}
	if info["description"] == "" {
		t.Error("empty description")
	}
	servers, _ := doc["servers"].([]any)
	if len(servers) != 1 {
		t.Fatalf("servers %v", servers)
	}
	srv, _ := servers[0].(map[string]any)
	if srv["url"] != productionServer {
		t.Errorf("server url %v", srv["url"])
	}
	paths, _ := doc["paths"].(map[string]any)
	for _, banned := range []string{"/openapi.json", "/docs", "/openapi.yaml", "/schemas"} {
		if _, ok := paths[banned]; ok {
			t.Errorf("spec includes %s", banned)
		}
	}
	if _, ok := paths["/problems"]; !ok {
		t.Error("missing /problems")
	}
	if _, ok := paths["/problems/reasons"]; !ok {
		t.Error("missing /problems/reasons")
	}
	comps, _ := doc["components"].(map[string]any)
	schemes, _ := comps["securitySchemes"].(map[string]any)
	if schemes["session"] == nil || schemes["bearer"] == nil {
		t.Errorf("securitySchemes %+v", schemes)
	}

}

func TestWriteSpecFilesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	app := fiber.New()
	api := Setup(app, Deps{})
	if err := WriteSpecFiles(dir, api); err != nil {
		t.Fatal(err)
	}
	dir2 := t.TempDir()
	if err := WriteSpecFiles(dir2, api); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"kungal-v1.json", "problems.json"} {
		a, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir2, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Errorf("%s not identical across two writes", name)
		}
	}
}
