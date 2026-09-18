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
