package apiv1

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestUnknownBodyFieldsAccepted(t *testing.T) {
	type in struct {
		Body struct {
			Name string `json:"name" maxLength:"32" doc:"Name."`
		}
	}
	type out struct {
		Body struct {
			Name string `json:"name"`
		}
	}
	app, _ := newTestAPI(t, Deps{}, func(api huma.API) {
		huma.Register(api, Public(huma.Operation{
			OperationID: "testEcho",
			Method:      http.MethodPost,
			Path:        "/_test/echo",
			Summary:     "Test echo",
			Description: "Echoes name and ignores unknown fields.",
		}), func(_ context.Context, input *in) (*out, error) {
			return &out{Body: struct {
				Name string `json:"name"`
			}{Name: input.Body.Name}}, nil
		})
	})

	resp := do(t, app, http.MethodPost, "/api/v1/_test/echo", `{"name":"alice","extra":true,"nested":{"x":1}}`, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body %s", resp.StatusCode, readBody(t, resp))
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(readBody(t, resp), &body); err != nil {
		t.Fatal(err)
	}
	if body.Name != "alice" {
		t.Errorf("name %s", body.Name)
	}
}

func TestSpecHasNoAdditionalPropertiesFalse(t *testing.T) {
	_, api := newTestAPI(t, Deps{})
	raw, err := MarshalOpenAPI(api)
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var hits []string
	walkJSON(doc, "$", func(path string, v any) {
		m, ok := v.(map[string]any)
		if !ok {
			return
		}
		if ap, ok := m["additionalProperties"]; ok {
			if ap == false {
				hits = append(hits, path)
			}
		}
	})
	if len(hits) > 0 {
		t.Fatalf("additionalProperties: false at %v", hits)
	}
}

func walkJSON(v any, path string, fn func(string, any)) {
	fn(path, v)
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			walkJSON(child, path+"."+k, fn)
		}
	case []any:
		for i, child := range x {
			walkJSON(child, path+"[]", fn)
			_ = i
		}
	}
}
