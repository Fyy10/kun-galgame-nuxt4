package apiv1

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

type NullablePart struct {
	Photo *repr.Image `json:"photo" doc:"Photo, or null."`
}

type nullables struct {
	NullablePart
	Cover     *repr.Image     `json:"cover" doc:"Cover, or null."`
	Banner    *repr.Image     `json:"banner,omitempty" doc:"Absent when there is none."`
	EditedAt  *repr.DateTime  `json:"edited_at" doc:"Last edit, or null."`
	ReplyToID *repr.DecimalID `json:"reply_to_id" doc:"Parent, or null."`
	Tags      *[]string       `json:"tags" maxItems:"4" doc:"Tags."`
	Grade     *string         `json:"grade" enum:"low,high" doc:"Grade, or null."`
}

func TestNilPointersAreNullableInTheDocument(t *testing.T) {
	app, api := newTestAPI(t, Deps{}, func(api huma.API) {
		huma.Register(api, Public(huma.Operation{OperationID: "n", Method: http.MethodGet, Path: "/n", Summary: "N"}),
			func(context.Context, *struct{}) (*struct{ Body nullables }, error) {
				return &struct{ Body nullables }{}, nil
			})
	})
	raw, err := json.Marshal(api.OpenAPI().Components.Schemas.Map()["Nullables"])
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"cover":       `{"anyOf":[{"$ref":"#/components/schemas/Image"},{"type":"null"}],"description":"Cover, or null."}`,
		"photo":       `{"anyOf":[{"$ref":"#/components/schemas/Image"},{"type":"null"}],"description":"Photo, or null."}`,
		"banner":      `{"$ref":"#/components/schemas/Image","description":"Absent when there is none."}`,
		"edited_at":   `"type":["string","null"]`,
		"reply_to_id": `"type":["string","null"]`,
		"tags":        `"type":["array","null"]`,
		"grade":       `"enum":["low","high",null]`,
	} {
		if got := string(s.Properties[name]); !strings.Contains(got, want) {
			t.Errorf("%s = %s, want %s", name, got, want)
		}
	}

	var body map[string]any
	if err := json.Unmarshal(readBody(t, do(t, app, http.MethodGet, "/api/v1/n", "", nil)), &body); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cover", "edited_at", "reply_to_id", "tags", "grade"} {
		if v, ok := body[name]; !ok || v != nil {
			t.Errorf("wire %s = %v (present %v), want null", name, v, ok)
		}
	}
	if _, ok := body["banner"]; ok {
		t.Errorf("wire banner present, want absent")
	}
}

func TestStructSchemasOmitAdditionalPropertiesKey(t *testing.T) {
	type in struct {
		Body struct {
			Name string `json:"name" maxLength:"32" doc:"Name."`
		}
	}
	type out struct {
		Body struct {
			Name string `json:"name" maxLength:"32" doc:"Name."`
		}
	}
	_, api := newTestAPI(t, Deps{}, func(api huma.API) {
		huma.Register(api, Public(huma.Operation{
			OperationID: "testSealBody",
			Method:      http.MethodPost,
			Path:        "/_test/seal-body",
			Summary:     "Test seal body",
			Description: "Struct request and response bodies.",
		}), func(context.Context, *in) (*out, error) {
			return &out{}, nil
		})
	})
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
		if _, ok := m["additionalProperties"]; ok {
			hits = append(hits, path)
		}
	})
	if len(hits) > 0 {
		t.Fatalf("additionalProperties key at %v", hits)
	}
}

func TestMapValueSchemaKeepsAdditionalProperties(t *testing.T) {
	type out struct {
		Body struct {
			Counts map[string]int `json:"counts" doc:"Per-key counts."`
		}
	}
	_, api := newTestAPI(t, Deps{}, func(api huma.API) {
		huma.Register(api, Public(huma.Operation{
			OperationID: "testSealMap",
			Method:      http.MethodGet,
			Path:        "/_test/seal-map",
			Summary:     "Test seal map",
			Description: "Response struct with a map field.",
		}), func(context.Context, *struct{}) (*out, error) {
			return &out{}, nil
		})
	})
	raw, err := MarshalOpenAPI(api)
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var found bool
	walkJSON(doc, "$", func(_ string, v any) {
		m, ok := v.(map[string]any)
		if !ok {
			return
		}
		props, _ := m["properties"].(map[string]any)
		counts, _ := props["counts"].(map[string]any)
		if counts == nil {
			return
		}
		ap, ok := counts["additionalProperties"].(map[string]any)
		if !ok {
			t.Errorf("counts additionalProperties = %v, want object schema", counts["additionalProperties"])
			return
		}
		if ap["type"] != "integer" {
			t.Errorf("counts additionalProperties type %v, want integer", ap["type"])
		}
		found = true
	})
	if !found {
		t.Fatal("counts additionalProperties object schema missing")
	}
}
